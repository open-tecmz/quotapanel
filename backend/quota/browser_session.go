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
	cmd := exec.Command(path, "--app="+pageURL, "--user-data-dir="+profile, "--remote-debugging-address=127.0.0.1", "--remote-debugging-port=0", "--no-first-run", "--no-default-browser-check")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("打开登录窗口失败: %w", err)
	}
	acc.track(cmd, false)
	acc.loginPending = true
	return nil
}

func (m *BrowserSessionManager) Complete(id uint) {
	acc := m.account(id)
	acc.mu.Lock()
	acc.loginPending = false
	acc.mu.Unlock()
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
	_ = os.Remove(filepath.Join(profile, "DevToolsActivePort"))
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
	target, err := m.ensureTarget(acc, profile, pageURL)
	if err != nil {
		return "", err
	}
	conn, _, err := websocket.DefaultDialer.Dial(target.WebSocketURL, nil)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(25 * time.Second))
	_ = conn.SetWriteDeadline(time.Now().Add(25 * time.Second))
	if !strings.HasPrefix(target.URL, pageURL) {
		if _, err := cdpCall(conn, 1, "Page.navigate", map[string]any{"url": pageURL}); err != nil {
			return "", err
		}
	}
	wanted, err := url.Parse(pageURL)
	if err != nil {
		return "", err
	}
	ready := false
	for i := 0; i < 30; i++ {
		raw, err := cdpCall(conn, 10+i, "Runtime.evaluate", map[string]any{"expression": "location.origin + '|' + document.readyState", "returnByValue": true})
		if err == nil {
			var state struct {
				Result struct {
					Value string `json:"value"`
				} `json:"result"`
			}
			_ = json.Unmarshal(raw["result"], &state)
			if strings.HasPrefix(state.Result.Value, wanted.Scheme+"://"+wanted.Host+"|") && strings.HasSuffix(state.Result.Value, "|complete") {
				ready = true
				break
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !ready {
		return "", errors.New("尚未进入供应商页面，请在登录窗口完成登录")
	}
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
	return value, nil
}
