package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"quotapanel/backend/base/platform"
	goruntime "runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"quotapanel/backend/base/config"
	"quotapanel/backend/base/llm"
	"quotapanel/backend/base/llmpx"
	"quotapanel/backend/base/logging"
	"quotapanel/backend/base/service"
	"quotapanel/backend/base/store"

	"quotapanel/backend/quota"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	AppName        = "quotapanel"
	AppDisplayName = "QuotaPanel"
	DefaultSlogan  = "AI Subscription Quota"
)

// App struct
type App struct {
	ctx          context.Context
	store        *store.Store
	config       config.AppConfig
	autoStartMgr *service.AutoStartManager
	systemLogger *logging.RollingStore
	dataDir      string
	quotaSvc     *quota.Service
	// quitting is set when the app exits programmatically (tray menu, restart,
	// resolved dialog) so onBeforeClose does not block the shutdown.
	quitting atomic.Bool
}

// NewApp creates a new App application struct
func NewApp() *App {
	dataDir := getAppDataDir()

	a := &App{
		store:        store.NewStore(dataDir),
		autoStartMgr: service.NewAutoStartManager(AppName, AppDisplayName),
		dataDir:      dataDir,
	}
	a.quotaSvc = quota.NewService()
	return a
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// panic 兜底上报
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("startup panic: %v", r)
			a.ReportError(msg, "", "startup")
			panic(r) // 重新抛出，让 Wails 框架处理
		}
	}()

	cfg, err := a.store.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}
	a.config = cfg

	env := runtime.Environment(ctx)
	var globalLogDir string
	if env.BuildType == "dev" {
		globalLogDir = filepath.Join(".", "logs")
	} else {
		globalLogDir = filepath.Join(a.dataDir, "logs")
	}
	if initErr := logging.Init(globalLogDir); initErr != nil {
		fmt.Printf("警告：初始化日志文件失败: %v\n", initErr)
	} else {
		logging.Info("日志系统已初始化，日志文件：%s", logging.GetLogFilePath())
	}

	if err != nil {
		logging.Error("加载配置失败，使用默认配置: %v", err)
	}

	systemLogDir := filepath.Join(a.dataDir, "system_logs")
	os.MkdirAll(systemLogDir, 0755)
	a.systemLogger = logging.NewRollingStore(systemLogDir, 1000, 10)

	logging.Info("应用启动成功，版本: %s，平台: %s", appConfig.Version, a.autoStartMgr.GetPlatform())

	a.initDatabase()
	a.quotaSvc.SetBrowserRunner(quota.NewBrowserSessionManager(filepath.Join(a.dataDir, "quota-browser")))
	go a.pollQuota(ctx)
}

func (a *App) shutdown(ctx context.Context) {
	a.quotaSvc.Close()
	a.logSystemError("shutdown", "Application is shutting down")
	// 强制上报队列中剩余的错误事件
	a.flushErrorQueue()
	a.logSystemError("shutdown", "Application shutdown complete")
}

// ─── 设置相关（私有，供 Call 调用）──────────────────────────────────────────

func (a *App) getLocale() string {
	return a.config.Locale
}

func (a *App) selectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Working Directory",
	})
}

func (a *App) selectFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Command/Executable",
		Filters: []runtime.FileFilter{
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
}

func (a *App) selectZipFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select ZIP File",
		Filters: []runtime.FileFilter{
			{DisplayName: "ZIP Files (*.zip)", Pattern: "*.zip"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
}

func (a *App) showWindow() {
	platform.ShowDockIcon()
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
	runtime.WindowSetAlwaysOnTop(a.ctx, true)
	runtime.WindowSetAlwaysOnTop(a.ctx, false)
}

func (a *App) hideWindow() {
	runtime.WindowHide(a.ctx)
	platform.HideDockIcon()
}

func (a *App) quitApp() {
	a.quitting.Store(true)
	runtime.Quit(a.ctx)
}

func (a *App) restartApp() {
	a.quitting.Store(true)
	platform.RestartApp()
	runtime.Quit(a.ctx)
}

// resolveClose handles the user's choice in the close confirmation dialog.
// action is quit / hide; remember indicates whether to persist the choice.
func (a *App) resolveClose(action string, remember bool) {
	if remember {
		a.config.CloseAction = action
		a.store.Save(a.config)
	}
	if action == "hide" {
		a.hideWindow()
		return
	}
	a.quitApp()
}

// closeActionKind returns the normalized configured close action: "quit", "hide",
// or "" meaning the app should ask the user every time.
func (a *App) closeActionKind() string {
	switch a.config.CloseAction {
	case "quit", "hide":
		return a.config.CloseAction
	default:
		return ""
	}
}

// requestClose applies the configured close action for the self-drawn close button
// (Windows / Linux): quit exits, hide hides the window, otherwise the frontend
// shows the confirmation dialog.
func (a *App) requestClose() {
	switch a.closeActionKind() {
	case "quit":
		a.quitApp()
	case "hide":
		a.hideWindow()
	default:
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "window:closeRequested")
		}
	}
}

// onBeforeClose decides the native window close behavior on all platforms.
// quit exits directly; hide hides the window and keeps running in background;
// otherwise the close is prevented and the frontend shows a confirmation dialog.
func (a *App) onBeforeClose(ctx context.Context) (prevent bool) {
	// Programmatic quit / restart must not be intercepted by the close behavior.
	if a.quitting.Load() {
		return false
	}
	switch a.closeActionKind() {
	case "quit":
		return false
	case "hide":
		a.hideWindow()
		return true
	default:
		runtime.EventsEmit(ctx, "window:closeRequested")
		return true
	}
}

// statusBarLabels returns the menu bar right-click menu labels, following the app locale.
func (a *App) statusBarLabels() platform.StatusBarLabels {
	if a.config.Locale == "en" {
		return platform.StatusBarLabels{Show: "Show Window", Restart: "Restart", Quit: "Quit"}
	}
	return platform.StatusBarLabels{Show: "显示界面", Restart: "重启", Quit: "退出"}
}

// updateStatusBarMenu refreshes the menu bar labels (called on locale change).
func (a *App) updateStatusBarMenu() {
	platform.UpdateStatusBarMenu(a.statusBarLabels())
}

func (a *App) saveLogsToFile(processName string, content string) error {
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Logs",
		DefaultFilename: fmt.Sprintf("%s-logs.txt", processName),
		Filters: []runtime.FileFilter{
			{DisplayName: "Text Files", Pattern: "*.txt"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return err
	}
	if filePath == "" {
		return nil
	}
	return os.WriteFile(filePath, []byte(content), 0644)
}

func (a *App) getSystemVersion() map[string]string {
	info := make(map[string]string)
	info["os"] = goruntime.GOOS
	info["arch"] = goruntime.GOARCH
	info["platform"] = a.autoStartMgr.GetPlatform()

	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "darwin":
		cmd = exec.Command("sw_vers", "-productVersion")
	case "linux":
		cmd = exec.Command("lsb_release", "-ds")
		if _, err := exec.LookPath("lsb_release"); err != nil {
			cmd = exec.Command("sh", "-c", "cat /etc/os-release | grep PRETTY_NAME | cut -d'=' -f2 | tr -d '\"'")
		}
	case "windows":
		cmd = exec.Command("cmd", "/c", "ver")
	}
	if cmd != nil {
		if output, err := cmd.Output(); err == nil {
			info["osVersion"] = strings.TrimSpace(string(output))
		} else {
			info["osVersion"] = "unknown"
		}
	}
	if hostname, err := os.Hostname(); err == nil {
		info["hostname"] = hostname
	}
	info["goVersion"] = goruntime.Version()
	info["numCPU"] = fmt.Sprintf("%d", goruntime.NumCPU())
	return info
}

func (a *App) getSystemLogs() (string, error) {
	var logs strings.Builder
	systemLogDir := filepath.Join(a.dataDir, "system_logs")
	logs.WriteString("=== Application System Logs ===\n")
	if entries, err := os.ReadDir(systemLogDir); err == nil {
		now := time.Now()
		yesterday := now.Add(-24 * time.Hour)
		totalSize := 0
		maxSize := 500 * 1024
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(yesterday) {
				filePath := filepath.Join(systemLogDir, entry.Name())
				content, err := os.ReadFile(filePath)
				if err == nil {
					if totalSize+len(content) > maxSize {
						logs.WriteString(fmt.Sprintf("\n... (remaining logs truncated, limit %dKB reached)\n", maxSize/1024))
						break
					}
					logs.WriteString(fmt.Sprintf("\n--- %s ---\n", entry.Name()))
					logs.Write(content)
					logs.WriteString("\n")
					totalSize += len(content)
				}
			}
		}
		if totalSize == 0 {
			logs.WriteString("No system logs found in the last 24 hours\n")
		}
	} else {
		logs.WriteString(fmt.Sprintf("Unable to read system logs directory: %v\n", err))
	}
	return logs.String(), nil
}

func (a *App) logSystemError(component, message string) {
	if a.systemLogger == nil {
		return
	}
	a.systemLogger.Append(logging.Entry{
		Timestamp: time.Now(),
		Stream:    component,
		Line:      message,
	})
}

// ─── 用户 & Token ────────────────────────────────────────────────────────────

// UserInfo holds basic user information returned from the server.
type UserInfo struct {
	Id       int    `json:"id"`
	Avatar   string `json:"avatar"`
	ViewName string `json:"viewName"`
}

func (a *App) getUserInfo() UserInfo {
	if a.config.ApiToken == "" {
		return UserInfo{}
	}
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", appConfig.ApiBaseUrl+"/app_manager/user_web_info", bytes.NewBufferString("{}"))
	if err != nil {
		return UserInfo{}
	}
	req.Header.Set("api-token", a.config.ApiToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return UserInfo{}
	}
	defer resp.Body.Close()
	var result struct {
		Code int `json:"code"`
		Data struct {
			MemberUser UserInfo `json:"memberUser"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return UserInfo{}
	}
	if result.Code != 0 {
		return UserInfo{}
	}
	return result.Data.MemberUser
}

// ─── 版本检查 ────────────────────────────────────────────────────────────────

// VersionInfo represents version information from the server
type VersionInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Time    string `json:"time"`
	Url     string `json:"url,omitempty"`
}

type versionCheckResponse struct {
	Code int         `json:"code"`
	Data VersionInfo `json:"data"`
}

func (a *App) checkVersion() (VersionInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	userAgent := fmt.Sprintf("AppOpen/%s/%s Platform/%s/%s/%s/%s",
		appConfig.Name, appConfig.Version,
		getPlatform(), getPlatformArch(), getPlatformVersion(), a.getDeviceUUID(),
	)
	formData := map[string]interface{}{
		"uuid":    a.getDeviceUUID(),
		"version": appConfig.Version,
		"platform": map[string]string{
			"name":    getPlatform(),
			"arch":    getPlatformArch(),
			"version": getPlatformVersion(),
		},
	}
	jsonData, err := json.Marshal(formData)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("failed to marshal request: %w", err)
	}
	req, err := http.NewRequest("POST", appConfig.VersionCheckUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return VersionInfo{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return VersionInfo{}, fmt.Errorf("HTTP error: status %d", resp.StatusCode)
	}
	var result versionCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return VersionInfo{}, fmt.Errorf("failed to decode response: %w", err)
	}
	if result.Code != 0 {
		return VersionInfo{}, fmt.Errorf("API error: code %d", result.Code)
	}
	return result.Data, nil
}

// ─── 分析埋点 ────────────────────────────────────────────────────────────────

// AnalyticsEvent represents an analytics event to be sent
type AnalyticsEvent struct {
	Name string                 `json:"name"`
	Data map[string]interface{} `json:"data,omitempty"`
}

// LLMCallResult 是 llm.CallResult 的别名，供 callCommon 路由使用
type LLMCallResult = llm.CallResult

// ─── 错误上报 ────────────────────────────────────────────────────────────────

const (
	errorBeaconURL  = "https://g.tecmz.com/grow/load.gif"
	errorBeaconApp  = "quotapanel"
	errorFlushDelay = 10 * time.Second
)

// errorReportEvent 对应 Beacon 接口的单条事件结构
type errorReportEvent struct {
	Et    string           `json:"et"`
	Path  string           `json:"path"`
	Did   string           `json:"did"`
	Sid   string           `json:"sid"`
	Ts    int64            `json:"ts"`
	Type  string           `json:"type"`
	Props errorReportProps `json:"props"`
}

type errorReportProps struct {
	Msg     string `json:"msg"`
	Stack   string `json:"stack,omitempty"`
	Src     string `json:"src,omitempty"`
	BuildID string `json:"build_id,omitempty"`
}

// 进程级会话 ID，每次启动重新生成
var errorSessionID = func() string {
	id, err := uuid.NewRandom()
	if err != nil {
		return fmt.Sprintf("sid_%d", time.Now().UnixNano())
	}
	return "sid_" + id.String()
}()

// errorQueue 和相关同步字段，由 errorMu 保护
var (
	errorMu    sync.Mutex
	errorQueue []errorReportEvent
	errorTimer *time.Timer
)

// ReportError 将一条后端错误加入批量队列，10s 后统一上报
func (a *App) ReportError(msg, stack, src string) {
	event := errorReportEvent{
		Et:   "error",
		Path: "/backend",
		Did:  a.getDeviceUUID(),
		Sid:  errorSessionID,
		Ts:   time.Now().UnixMilli(),
		Type: "app-" + appConfig.Version,
		Props: errorReportProps{
			Msg:     msg,
			Stack:   stack,
			Src:     src,
			BuildID: appConfig.Version,
		},
	}
	errorMu.Lock()
	errorQueue = append(errorQueue, event)
	if errorTimer == nil {
		errorTimer = time.AfterFunc(errorFlushDelay, a.flushErrorQueue)
	}
	errorMu.Unlock()
}

// flushErrorQueue 将队列中的错误批量发送到 Beacon 接口
func (a *App) flushErrorQueue() {
	errorMu.Lock()
	if len(errorQueue) == 0 {
		errorTimer = nil
		errorMu.Unlock()
		return
	}
	batch := make([]errorReportEvent, len(errorQueue))
	copy(batch, errorQueue)
	errorQueue = errorQueue[:0]
	errorTimer = nil
	errorMu.Unlock()

	go func() {
		data, err := json.Marshal(batch)
		if err != nil {
			return
		}
		encoded := base64.StdEncoding.EncodeToString(data)
		urlStr := fmt.Sprintf("%s?app=%s&data=%s", errorBeaconURL, errorBeaconApp, url.QueryEscape(encoded))
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(urlStr)
		if err != nil {
			return
		}
		resp.Body.Close()
	}()
}

func (a *App) sendAnalytics(events []AnalyticsEvent) {
	go func() {
		client := &http.Client{Timeout: 10 * time.Second}
		userAgent := fmt.Sprintf("AppOpen/%s/%s Platform/%s/%s/%s/%s",
			appConfig.Name, appConfig.Version,
			getPlatform(), getPlatformArch(), getPlatformVersion(), a.getDeviceUUID(),
		)
		formData := map[string]interface{}{
			"uuid":    a.getDeviceUUID(),
			"version": appConfig.Version,
			"data":    events,
			"platform": map[string]string{
				"name":    getPlatform(),
				"arch":    getPlatformArch(),
				"version": getPlatformVersion(),
			},
		}
		jsonData, err := json.Marshal(formData)
		if err != nil {
			return
		}
		req, err := http.NewRequest("POST", appConfig.AnalyticsUrl, bytes.NewBuffer(jsonData))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", userAgent)
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()
	}()
}

// ─── 应用元数据 ──────────────────────────────────────────────────────────────

// baseURL：应用开放接口前缀（open_app 代理，用于统计/更新/引导/帮助/反馈）
// siteBaseURL：站点根地址，用户中心页面与用户接口使用
const baseURL = "https://open.tecmz.com/open_app/QuotaPanel"
const siteBaseURL = "https://open.tecmz.com"

var appConfig = struct {
	Name            string
	Title           string
	Slogan          string
	Version         string
	Website         string
	WebsiteGithub   string
	WebsiteGitee    string
	ApiBaseUrl      string
	UserBaseUrl     string
	AnalyticsUrl    string
	VersionCheckUrl string
	FeedbackUrl     string
	GuideUrl        string
	HelpUrl         string
}{
	Name:            "quotapanel",
	Title:           "QuotaPanel",
	Slogan:          DefaultSlogan,
	Version:         "v0.1.0",
	Website:         siteBaseURL + "/quotapanel",
	WebsiteGithub:   "https://github.com/open-tecmz/quotapanel",
	WebsiteGitee:    "https://gitee.com/open-tecmz/quotapanel",
	ApiBaseUrl:      siteBaseURL + "/api",
	UserBaseUrl:     siteBaseURL,
	AnalyticsUrl:    baseURL + "/app_manager/collect",
	VersionCheckUrl: baseURL + "/app_manager/updater",
	FeedbackUrl:     baseURL + "/feedback_ticket",
	GuideUrl:        baseURL + "/app_manager/guide",
	HelpUrl:         baseURL + "/app_manager/help",
}

func (a *App) getDeviceUUID() string {
	if a.config.DeviceUUID != "" {
		return a.config.DeviceUUID
	}
	newUUID := uuid.New().String()
	a.config.DeviceUUID = newUUID
	a.store.Save(a.config)
	return newUUID
}

func getPlatform() string {
	switch goruntime.GOOS {
	case "darwin":
		return "mac"
	case "windows":
		return "win"
	case "linux":
		return "linux"
	default:
		return goruntime.GOOS
	}
}

func getPlatformArch() string {
	switch goruntime.GOARCH {
	case "amd64":
		return "x64"
	case "arm64":
		return "arm64"
	case "386":
		return "x86"
	default:
		return goruntime.GOARCH
	}
}

// getPlatformName 返回标准化平台标识：win / osx / linux / none
func getPlatformName() string {
	switch goruntime.GOOS {
	case "darwin":
		return "osx"
	case "windows":
		return "win"
	case "linux":
		return "linux"
	default:
		return "none"
	}
}

// getPlatformArchName 返回标准化架构标识：x86 / arm64 / none
func getPlatformArchName() string {
	switch goruntime.GOARCH {
	case "amd64", "386":
		return "x86"
	case "arm64", "arm":
		return "arm64"
	default:
		return "none"
	}
}

func getPlatformVersion() string {
	switch goruntime.GOOS {
	case "darwin":
		out, err := exec.Command("sw_vers", "-productVersion").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	case "windows":
		out, err := exec.Command("cmd", "/c", "ver").Output()
		if err == nil {
			s := string(out)
			if start := strings.Index(s, "[Version "); start != -1 {
				s = s[start+9:]
				if end := strings.Index(s, "]"); end != -1 {
					return strings.TrimSpace(s[:end])
				}
			}
		}
	case "linux":
		data, err := os.ReadFile("/etc/os-release")
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "VERSION_ID=") {
					v := strings.TrimPrefix(line, "VERSION_ID=")
					return strings.Trim(v, "\"")
				}
			}
		}
	}
	return "0"
}

// ─── 通用 Call 路由 ──────────────────────────────────────────────────────────

// callCommon 处理所有模块共用的 Call 路由（llm.*、setting.*）。
// 返回值：(result, error, matched)，matched=false 表示该 name 不属于通用路由。
func (a *App) callCommon(name string, decode func(interface{}) error) (interface{}, error, bool) {
	switch name {

	// ── llm：大模型 ─────────────────────────────────────────────────────────

	case "llm.getLLMConfigs":
		if a.config.LLMConfigs == nil {
			return []config.LLMConfig{}, nil, true
		}
		return a.config.LLMConfigs, nil, true

	case "llm.saveLLMConfigs":
		var p struct {
			Configs []config.LLMConfig `json:"configs"`
		}
		if err := decode(&p); err != nil {
			return nil, err, true
		}
		a.config.LLMConfigs = p.Configs
		return nil, a.store.Save(a.config), true

	case "llm.testLLMConfig":
		var p struct {
			Provider string `json:"provider"`
			APIKey   string `json:"apiKey"`
			BaseURL  string `json:"baseURL"`
			Model    string `json:"model"`
		}
		if err := decode(&p); err != nil {
			return nil, err, true
		}
		return llm.TestConfig(p.Provider, p.APIKey, p.BaseURL, p.Model), nil, true

	case "llm.callLLM":
		var p struct {
			ConfigID     string `json:"configID"`
			SystemPrompt string `json:"systemPrompt"`
			UserPrompt   string `json:"userPrompt"`
		}
		if err := decode(&p); err != nil {
			return nil, err, true
		}
		result, err := llm.Call(p.ConfigID, a.config.LLMConfigs, appConfig.ApiBaseUrl, a.config.ApiToken, p.SystemPrompt, p.UserPrompt)
		return result, err, true

	case "llm.callLLMVision":
		var p struct {
			ConfigID     string `json:"configID"`
			SystemPrompt string `json:"systemPrompt"`
			UserPrompt   string `json:"userPrompt"`
			ImageBase64  string `json:"imageBase64"`
		}
		if err := decode(&p); err != nil {
			return nil, err, true
		}
		result, err := llm.CallVision(p.ConfigID, a.config.LLMConfigs, appConfig.ApiBaseUrl, a.config.ApiToken, p.SystemPrompt, p.UserPrompt, p.ImageBase64)
		return result, err, true

	case "llm.getLLMPXInfo":
		result, err := llmpx.FetchInfo(appConfig.ApiBaseUrl, a.config.ApiToken)
		return result, err, true

	case "llm.getBuiltinModelSettings":
		if a.config.BuiltinModelSettings == nil {
			return []config.BuiltinModelSetting{}, nil, true
		}
		return a.config.BuiltinModelSettings, nil, true

	case "llm.saveBuiltinModelSettings":
		var p struct {
			Settings []config.BuiltinModelSetting `json:"settings"`
		}
		if err := decode(&p); err != nil {
			return nil, err, true
		}
		a.config.BuiltinModelSettings = p.Settings
		return nil, a.store.Save(a.config), true

	// ── setting：应用设置 ───────────────────────────────────────────────────

	case "setting.getAppConfig":
		return map[string]interface{}{
			"name":            appConfig.Name,
			"title":           appConfig.Title,
			"slogan":          appConfig.Slogan,
			"version":         appConfig.Version,
			"website":         appConfig.Website,
			"websiteGithub":   appConfig.WebsiteGithub,
			"websiteGitee":    appConfig.WebsiteGitee,
			"apiBaseUrl":      appConfig.ApiBaseUrl,
			"userBaseUrl":     appConfig.UserBaseUrl,
			"deviceUuid":      a.getDeviceUUID(),
			"analyticsUrl":    appConfig.AnalyticsUrl,
			"versionCheckUrl": appConfig.VersionCheckUrl,
			"feedbackUrl":     appConfig.FeedbackUrl,
			"guideUrl":        appConfig.GuideUrl,
			"helpUrl":         appConfig.HelpUrl,
		}, nil, true

	case "setting.checkVersion":
		result, err := a.checkVersion()
		return result, err, true

	case "setting.getAutoStartEnabled":
		enabled, err := a.autoStartMgr.IsEnabled()
		return enabled, err, true

	case "setting.setAutoStartEnabled":
		var p struct {
			Enabled bool `json:"enabled"`
		}
		if err := decode(&p); err != nil {
			return nil, err, true
		}
		if p.Enabled {
			return nil, a.autoStartMgr.Enable(), true
		}
		return nil, a.autoStartMgr.Disable(), true

	case "setting.getApiToken":
		return a.config.ApiToken, nil, true

	case "setting.setApiToken":
		var p struct {
			Token string `json:"token"`
		}
		if err := decode(&p); err != nil {
			return nil, err, true
		}
		a.config.ApiToken = p.Token
		return nil, a.store.Save(a.config), true

	case "setting.getSystemVersion":
		return a.getSystemVersion(), nil, true

	case "setting.getUserInfo":
		return a.getUserInfo(), nil, true

	case "setting.sendAnalytics":
		var p struct {
			Events []AnalyticsEvent `json:"events"`
		}
		if err := decode(&p); err != nil {
			return nil, err, true
		}
		a.sendAnalytics(p.Events)
		return nil, nil, true

	case "setting.getLocale":
		return a.config.Locale, nil, true

	case "setting.setLocale":
		var p struct {
			Locale string `json:"locale"`
		}
		if err := decode(&p); err != nil {
			return nil, err, true
		}
		a.config.Locale = p.Locale
		err := a.store.Save(a.config)
		a.updateStatusBarMenu()
		return nil, err, true

	case "setting.showWindow":
		a.showWindow()
		return nil, nil, true

	case "setting.hideWindow":
		a.hideWindow()
		return nil, nil, true

	case "setting.quitApp":
		a.quitApp()
		return nil, nil, true

	case "setting.requestClose":
		a.requestClose()
		return nil, nil, true

	case "setting.restartApp":
		a.restartApp()
		return nil, nil, true

	case "setting.getCloseAction":
		return a.config.CloseAction, nil, true

	case "setting.setCloseAction":
		var p struct {
			Action string `json:"action"`
		}
		if err := decode(&p); err != nil {
			return nil, err, true
		}
		a.config.CloseAction = p.Action
		return nil, a.store.Save(a.config), true

	case "setting.resolveClose":
		var p struct {
			Action   string `json:"action"`
			Remember bool   `json:"remember"`
		}
		if err := decode(&p); err != nil {
			return nil, err, true
		}
		a.resolveClose(p.Action, p.Remember)
		return nil, nil, true

	case "setting.selectDirectory":
		result, err := a.selectDirectory()
		return result, err, true

	case "setting.selectFile":
		result, err := a.selectFile()
		return result, err, true

	case "setting.selectZipFile":
		result, err := a.selectZipFile()
		return result, err, true

	case "setting.saveLogsToFile":
		var p struct {
			ProcessName string `json:"processName"`
			Content     string `json:"content"`
		}
		if err := decode(&p); err != nil {
			return nil, err, true
		}
		return nil, a.saveLogsToFile(p.ProcessName, p.Content), true

	case "setting.getSystemLogs":
		result, err := a.getSystemLogs()
		return result, err, true

	case "setting.getPlatformName":
		return getPlatformName(), nil, true

	case "setting.getPlatformArch":
		return getPlatformArchName(), nil, true

	}
	return nil, nil, false
}
