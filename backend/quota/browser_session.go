package quota

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"quotapanel/backend/base/logging"

	"github.com/gorilla/websocket"
)

// BrowserSessionManager 用独立 Chrome 应用窗口承载网页登录。
// 每个账号使用独立的浏览器资料目录，Cookie 由 Chrome 自身保存，应用不复制登录令牌。
type BrowserSessionManager struct {
	root     string
	mu       sync.Mutex
	accounts map[uint]*browserAccount
}

type browserAccount struct {
	mu           sync.Mutex
	loginPending bool
	cmd          *exec.Cmd
	headless     bool
	done         chan struct{}
}

func (a *browserAccount) track(cmd *exec.Cmd, headless bool) {
	a.cmd, a.headless = cmd, headless
	done := make(chan struct{})
	a.done = done
	go func() {
		_ = cmd.Wait()
		close(done)
		a.mu.Lock()
		if a.cmd == cmd {
			a.cmd = nil
		}
		a.mu.Unlock()
	}()
}

func NewBrowserSessionManager(root string) *BrowserSessionManager {
	return &BrowserSessionManager{root: root, accounts: make(map[uint]*browserAccount)}
}

// 登录 Cookie 的落盘文件（放在账号的 profile 目录内，随账号删除一起清理）。
const profileCookieFile = ".quotapanel-cookies.json"

// killProfileBrowsers 结束仍占用该 profile 的浏览器进程。
// 部分站点登录窗口关闭后进程可能残留，导致后续无头会话因 profile 被锁而启动失败。
func killProfileBrowsers(profile string) {
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/F", "/FI", "COMMANDLINE eq *"+profile+"*").Run()
		return
	}
	_ = exec.Command("pkill", "-f", profile).Run()
	time.Sleep(300 * time.Millisecond)
}

// cleanProfileLocks 清理浏览器异常退出后残留的锁文件与调试端口文件。
func cleanProfileLocks(profile string) {
	for _, name := range []string{"DevToolsActivePort", "SingletonLock", "SingletonCookie", "SingletonSocket"} {
		_ = os.Remove(filepath.Join(profile, name))
	}
}

func loadProfileCookies(profile string) []map[string]any {
	data, err := os.ReadFile(filepath.Join(profile, profileCookieFile))
	if err != nil {
		return nil
	}
	var cookies []map[string]any
	if json.Unmarshal(data, &cookies) != nil || len(cookies) == 0 {
		return nil
	}
	return cookies
}

// saveProfileCookies 保存当前会话 Cookie。
// 小米等站点的登录 Cookie 是会话级的，浏览器进程退出即失效；这里统一续期为 7 天后再落盘。
func saveProfileCookies(profile string, conn *websocket.Conn) {
	raw, err := cdpCall(conn, 950, "Network.getAllCookies", map[string]any{})
	if err != nil {
		return
	}
	var res struct {
		Result struct {
			Cookies []map[string]any `json:"cookies"`
		} `json:"result"`
	}
	if json.Unmarshal(raw["result"], &res) != nil || len(res.Result.Cookies) == 0 {
		return
	}
	expiry := float64(time.Now().Add(7 * 24 * time.Hour).Unix())
	for _, cookie := range res.Result.Cookies {
		if v, ok := cookie["expires"].(float64); !ok || v <= 0 {
			cookie["expires"] = expiry
			delete(cookie, "session")
		}
	}
	if data, err := json.Marshal(res.Result.Cookies); err == nil {
		_ = os.WriteFile(filepath.Join(profile, profileCookieFile), data, 0o600)
	}
}

func (m *BrowserSessionManager) account(id uint) *browserAccount {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.accounts[id] == nil {
		m.accounts[id] = &browserAccount{}
	}
	return m.accounts[id]
}

func (m *BrowserSessionManager) profile(id uint) string {
	return filepath.Join(m.root, strconv.FormatUint(uint64(id), 10))
}

func browserExecutable() (string, error) {
	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		candidates = []string{"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"}
	case "windows":
		candidates = []string{filepath.Join(os.Getenv("PROGRAMFILES"), "Google", "Chrome", "Application", "chrome.exe"), filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Microsoft", "Edge", "Application", "msedge.exe")}
	default:
		candidates = []string{"google-chrome", "chromium", "microsoft-edge"}
	}
	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", errors.New("未找到 Chrome 或 Edge，请安装浏览器后重试")
}

func (m *BrowserSessionManager) Open(id uint, pageURL string) error {
	acc := m.account(id)
	acc.mu.Lock()
	defer acc.mu.Unlock()
	path, err := browserExecutable()
	if err != nil {
		return err
	}
	profile := m.profile(id)
	if err := os.MkdirAll(profile, 0700); err != nil {
		return err
	}
	if acc.cmd != nil {
		m.stopBrowser(acc, profile)
		_ = os.Remove(filepath.Join(profile, "DevToolsActivePort"))
	}
	// 清理可能残留的浏览器进程与锁文件，确保登录窗口能正常打开
	killProfileBrowsers(profile)
	cleanProfileLocks(profile)
	cmd := exec.Command(path, "--app="+pageURL, "--user-data-dir="+profile, "--remote-debugging-address=127.0.0.1", "--remote-debugging-port=0", "--no-first-run", "--no-default-browser-check")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("打开登录窗口失败: %w", err)
	}
	acc.track(cmd, false)
	acc.loginPending = true
	go m.watchLogin(id, pageURL)
	return nil
}

func (m *BrowserSessionManager) Complete(id uint) {
	acc := m.account(id)
	acc.mu.Lock()
	// 登录完成时立刻保存 Cookie：部分站点的登录 Cookie 是会话级的，窗口关闭后会丢失
	profile := m.profile(id)
	if target, err := m.target(profile); err == nil {
		if conn, _, derr := websocket.DefaultDialer.Dial(target.WebSocketURL, nil); derr == nil {
			_ = conn.SetReadDeadline(time.Now().Add(6 * time.Second))
			saveProfileCookies(profile, conn)
			_ = conn.Close()
		}
	}
	acc.loginPending = false
	acc.mu.Unlock()
}

// watchLogin 在登录窗口打开后自动检测登录完成：一旦页面进入目标地址，
// 立即保存会话 Cookie 并结束"待登录"状态，避免用户必须手动点「验证登录」。
func (m *BrowserSessionManager) watchLogin(id uint, pageURL string) {
	profile := m.profile(id)
	for i := 0; i < 150; i++ { // 最多观察约 5 分钟
		time.Sleep(2 * time.Second)
		acc := m.account(id)
		acc.mu.Lock()
		pending := acc.loginPending
		acc.mu.Unlock()
		if !pending {
			return
		}
		target, err := m.target(profile)
		if err != nil {
			continue
		}
		conn, _, err := websocket.DefaultDialer.Dial(target.WebSocketURL, nil)
		if err != nil {
			continue
		}
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		raw, err := cdpCall(conn, 1, "Runtime.evaluate", map[string]any{"expression": "location.href", "returnByValue": true})
		if err == nil {
			var res struct {
				Result struct {
					Value string `json:"value"`
				} `json:"result"`
			}
			_ = json.Unmarshal(raw["result"], &res)
			if strings.HasPrefix(res.Result.Value, pageURL) {
				_ = conn.SetReadDeadline(time.Now().Add(6 * time.Second))
				saveProfileCookies(profile, conn)
				acc.mu.Lock()
				acc.loginPending = false
				acc.mu.Unlock()
				logging.Info("quota browser: account=%d 已检测到登录并保存会话", id)
				_ = conn.Close()
				return
			}
		}
		_ = conn.Close()
	}
}

func (m *BrowserSessionManager) Remove(id uint) error {
	acc := m.account(id)
	acc.mu.Lock()
	defer acc.mu.Unlock()
	m.stopBrowser(acc, m.profile(id))
	return os.RemoveAll(m.profile(id))
}

func (m *BrowserSessionManager) Close() {
	m.mu.Lock()
	accounts := make(map[uint]*browserAccount, len(m.accounts))
	for id, acc := range m.accounts {
		accounts[id] = acc
	}
	m.mu.Unlock()
	for id, acc := range accounts {
		acc.mu.Lock()
		m.stopBrowser(acc, m.profile(id))
		acc.mu.Unlock()
	}
}

func (m *BrowserSessionManager) stopBrowser(acc *browserAccount, profile string) {
	if acc.cmd == nil {
		return
	}
	done := acc.done
	if port, err := m.port(profile); err == nil {
		client := &http.Client{Timeout: 2 * time.Second}
		if resp, err := client.Get("http://127.0.0.1:" + port + "/json/version"); err == nil {
			var info struct {
				WebSocketURL string `json:"webSocketDebuggerUrl"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&info)
			_ = resp.Body.Close()
			if info.WebSocketURL != "" {
				if conn, _, err := websocket.DefaultDialer.Dial(info.WebSocketURL, nil); err == nil {
					_ = conn.WriteJSON(map[string]any{"id": 1, "method": "Browser.close"})
					_ = conn.Close()
				}
			}
		}
	}
	for i := 0; i < 20; i++ {
		if _, err := m.target(profile); err != nil {
			select {
			case <-done:
				return
			case <-time.After(2 * time.Second):
				_ = acc.cmd.Process.Kill()
				<-done
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = acc.cmd.Process.Kill()
	<-done
}

type cdpTarget struct {
	Type         string `json:"type"`
	URL          string `json:"url"`
	WebSocketURL string `json:"webSocketDebuggerUrl"`
}

func (m *BrowserSessionManager) port(profile string) (string, error) {
	data, err := os.ReadFile(filepath.Join(profile, "DevToolsActivePort"))
	if err != nil {
		return "", err
	}
	port := strings.TrimSpace(strings.SplitN(string(data), "\n", 2)[0])
	if _, err := strconv.Atoi(port); err != nil {
		return "", fmt.Errorf("浏览器调试端口无效")
	}
	return port, nil
}

func (m *BrowserSessionManager) target(profile string) (*cdpTarget, error) {
	port, err := m.port(profile)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/json/list")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var targets []cdpTarget
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return nil, err
	}
	for _, t := range targets {
		if t.Type == "page" && t.WebSocketURL != "" {
			return &t, nil
		}
	}
	return nil, errors.New("登录窗口尚未准备好")
}

func (m *BrowserSessionManager) ensureTarget(acc *browserAccount, profile, pageURL string) (*cdpTarget, error) {
	if t, err := m.target(profile); err == nil {
		return t, nil
	}
	path, err := browserExecutable()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(profile, 0700); err != nil {
		return nil, err
	}
	// 结束占用 profile 的残留进程并清理锁，否则无头会话会因 profile 被锁而启动失败
	killProfileBrowsers(profile)
	cleanProfileLocks(profile)
	cmd := exec.Command(path, "--headless=new", "--disable-gpu", "--user-data-dir="+profile, "--remote-debugging-address=127.0.0.1", "--remote-debugging-port=0", "--no-first-run", pageURL)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	acc.track(cmd, true)
	for i := 0; i < 40; i++ {
		time.Sleep(200 * time.Millisecond)
		if t, err := m.target(profile); err == nil {
			return t, nil
		}
	}
	return nil, errors.New("浏览器会话启动超时")
}

func cdpCall(conn *websocket.Conn, id int, method string, params any) (map[string]json.RawMessage, error) {
	if err := conn.WriteJSON(map[string]any{"id": id, "method": method, "params": params}); err != nil {
		return nil, err
	}
	for {
		var raw map[string]json.RawMessage
		if err := conn.ReadJSON(&raw); err != nil {
			return nil, err
		}
		var got int
		_ = json.Unmarshal(raw["id"], &got)
		if got != id {
			continue
		}
		if e := raw["error"]; len(e) > 0 {
			return nil, fmt.Errorf("浏览器执行失败: %s", e)
		}
		return raw, nil
	}
}

func (m *BrowserSessionManager) Run(id uint, pageURL, script string) (string, error) {
	acc := m.account(id)
	acc.mu.Lock()
	defer acc.mu.Unlock()
	if acc.loginPending {
		return "", errors.New("请先在登录窗口完成登录，然后点击“验证登录”")
	}
	profile := m.profile(id)

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		value, err := m.runOnce(acc, profile, pageURL, script, id)
		if err == nil {
			return value, nil
		}
		lastErr = err
		if !isTransientCDPError(err) {
			return "", err
		}
		// 页面在跳转（SSO / SPA 路由）导致本次求值作废，稍后重新解析目标再试
		time.Sleep(600 * time.Millisecond)
	}
	return "", lastErr
}

// isTransientCDPError 判断是否是「页面正在跳转」这类可重试的 CDP 错误。
func isTransientCDPError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "navigated or closed") || strings.Contains(msg, "-32000")
}

func (m *BrowserSessionManager) runOnce(acc *browserAccount, profile, pageURL, script string, id uint) (string, error) {
	target, err := m.ensureTarget(acc, profile, pageURL)
	if err != nil {
		return "", err
	}
	conn, _, err := websocket.DefaultDialer.Dial(target.WebSocketURL, nil)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	_ = conn.SetWriteDeadline(time.Now().Add(30 * time.Second))

	// 注入已保存的登录 Cookie：会话级 Cookie 在浏览器进程退出后会丢失，
	// 这里在各站点导航前统一补齐，保证无头查询仍处于登录态。
	_, _ = cdpCall(conn, 900, "Network.enable", map[string]any{})
	if cookies := loadProfileCookies(profile); len(cookies) > 0 {
		_, _ = cdpCall(conn, 901, "Network.setCookies", map[string]any{"cookies": cookies})
	}

	if !strings.HasPrefix(target.URL, pageURL) {
		if _, err := cdpCall(conn, 1, "Page.navigate", map[string]any{"url": pageURL}); err != nil && !isTransientCDPError(err) {
			return "", err
		}
	}
	wanted, err := url.Parse(pageURL)
	if err != nil {
		return "", err
	}
	expected := wanted.Scheme + "://" + wanted.Host + wanted.Path

	// 等待页面稳定：连续两次地址与 readyState 都符合，避免在跳转过程中求值
	ready := false
	lastState := ""
	stable := 0
	for i := 0; i < 60; i++ {
		raw, err := cdpCall(conn, 10+i, "Runtime.evaluate", map[string]any{"expression": "location.href + '|' + document.readyState", "returnByValue": true})
		if err == nil {
			var state struct {
				Result struct {
					Value string `json:"value"`
				} `json:"result"`
			}
			_ = json.Unmarshal(raw["result"], &state)
			lastState = state.Result.Value
			if strings.HasPrefix(state.Result.Value, expected) && strings.HasSuffix(state.Result.Value, "|complete") {
				stable++
				if stable >= 2 {
					ready = true
					break
				}
			} else {
				stable = 0
			}
		} else {
			stable = 0
		}
		time.Sleep(300 * time.Millisecond)
	}
	if !ready {
		logging.Warn("quota browser: account=%d 未进入目标页面（当前 %s）", id, lastState)
		return "", fmt.Errorf("尚未进入供应商页面（当前地址：%s），请在登录窗口完成登录后重试", lastState)
	}
	// 已确认处于目标页面，先保存会话 Cookie，保证窗口关闭后无头查询仍可用
	saveProfileCookies(profile, conn)
	raw, err := cdpCall(conn, 100, "Runtime.evaluate", map[string]any{"expression": script, "awaitPromise": true, "returnByValue": true})
	if err != nil {
		return "", err
	}
	var result struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		Exception json.RawMessage `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(raw["result"], &result); err != nil {
		return "", fmt.Errorf("浏览器响应格式错误: %w", err)
	}
	if len(result.Exception) > 0 {
		return "", errors.New("登录状态或页面查询失败，请重新登录或稍后刷新")
	}
	var value string
	if err := json.Unmarshal(result.Result.Value, &value); err != nil {
		return "", fmt.Errorf("浏览器返回值类型不正确: %w", err)
	}
	if value == "" {
		return "", errors.New("额度页面未返回数据，请确认已登录")
	}
	saveProfileCookies(profile, conn)
	return value, nil
}
