package main

import (
	"context"
	"embed"
	goruntime "runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"quotapanel/backend/base/platform"
)

//go:embed all:frontend/packages/ui/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

//go:embed build/trayicon.png
var trayIcon []byte

//go:embed build/trayicon_windows.ico
var trayIconWindows []byte

// Global app reference for status bar / single instance callbacks
var globalApp *App

// statusBarIcon returns the status bar icon bytes for the current platform.
// Windows requires an ICO file, while other platforms accept PNG.
func statusBarIcon() []byte {
	if goruntime.GOOS == "windows" {
		return trayIconWindows
	}
	return trayIcon
}

// createApplicationMenu creates the application menu with Edit menu
// Returns nil on Windows to hide the menu bar
func createApplicationMenu() *menu.Menu {
	// On Windows, return nil to hide menu bar
	// Keyboard shortcuts (Ctrl+C/V/X/A/Z) work natively via WebviewGpuIsDisabled=false
	if goruntime.GOOS == "windows" {
		return nil
	}

	appMenu := menu.NewMenu()

	if goruntime.GOOS == "darwin" {
		// App menu (macOS only)
		appMenu.Append(menu.AppMenu())
		// Edit menu with standard shortcuts using roles (macOS)
		appMenu.Append(menu.EditMenu())
	} else {
		// Edit menu with standard shortcuts (Linux)
		editMenu := appMenu.AddSubmenu("Edit")
		editMenu.AddText("Undo", keys.CmdOrCtrl("z"), func(_ *menu.CallbackData) {})
		editMenu.AddText("Redo", keys.CmdOrCtrl("shift+z"), func(_ *menu.CallbackData) {})
		editMenu.AddSeparator()
		editMenu.AddText("Cut", keys.CmdOrCtrl("x"), func(_ *menu.CallbackData) {})
		editMenu.AddText("Copy", keys.CmdOrCtrl("c"), func(_ *menu.CallbackData) {})
		editMenu.AddText("Paste", keys.CmdOrCtrl("v"), func(_ *menu.CallbackData) {})
		editMenu.AddText("Select All", keys.CmdOrCtrl("a"), func(_ *menu.CallbackData) {})
	}

	return appMenu
}

// wrapStartup wraps the app startup to initialize the menu bar / system tray item.
func wrapStartup(app *App) func(ctx context.Context) {
	return func(ctx context.Context) {
		// Call original startup
		app.startup(ctx)

		// Ensure window is shown on startup
		runtime.WindowShow(ctx)

		// Create the menu bar (macOS) / system tray (Windows, Linux) item:
		// left click shows the window, right click opens the Show / Restart / Quit menu.
		platform.InitStatusBar(statusBarIcon(), app.statusBarLabels(), platform.StatusBarHandlers{
			OnShow:    app.showWindow,
			OnRestart: app.restartApp,
			OnQuit:    app.quitApp,
		})
	}
}

// wrapShutdown wraps the app shutdown to cleanup resources
func wrapShutdown(app *App) func(ctx context.Context) {
	return func(ctx context.Context) {
		platform.QuitStatusBar()
		// Call original shutdown
		app.shutdown(ctx)
	}
}

func main() {
	// Create an instance of the app structure
	app := NewApp()
	globalApp = app

	// 测试模式：解析 --auto-test-port 并启动测试 HTTP 服务
	// （autotest build tag 下生效，其他构建为空操作）
	app.startAutoTestServer(parseAutoTestPort())

	// Create application with options
	err := wails.Run(&options.App{
		Title:         "QuotaPanel",
		Width:         1024,
		Height:        720,
		MinWidth:      400,
		MinHeight:     400,
		// macOS 保留系统边框（透明标题栏展示系统红绿灯），
		// Windows/Linux 使用无边框窗口由前端自绘窗口按钮。
		Frameless:     platform.Frameless(),
		DisableResize: false,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour:         &options.RGBA{R: 24, G: 24, B: 27, A: 1},
		EnableDefaultContextMenu: false,
		Menu:                     createApplicationMenu(),
		CSSDragValue:             "drag",
		CSSDragProperty:          "--wails-draggable",

		// Window close behavior is driven by app config (ask / quit / hide)
		OnBeforeClose: app.onBeforeClose,

		OnStartup:  wrapStartup(app),
		OnShutdown: wrapShutdown(app),
		Bind: []interface{}{
			app,
		},

		// macOS specific options
		// 透明标题栏 + 全尺寸内容：内容延伸到窗口顶部，系统红绿灯浮于内容左上角
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  true,
				HideTitleBar:               false,
				FullSizeContent:            true,
				UseToolbar:                 false,
				HideToolbarSeparator:       true,
			},
			About: &mac.AboutInfo{
				Title:   "QuotaPanel",
				Message: "AI Subscription Quota",
				Icon:    icon,
			},
		},

		// Windows specific options
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			// WebView2 用户数据（appData）写入数据根目录
			WebviewUserDataPath: app.dataDir,
		},

		// Linux specific options
		Linux: &linux.Options{
			Icon:                icon,
			WindowIsTranslucent: false,
		},

		// Single instance lock - show window when second instance is launched
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "quotapanel-app-unique-id",
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				if globalApp != nil && globalApp.ctx != nil {
					// Show Dock icon first (macOS)
					platform.ShowDockIcon()
					// Show the window
					runtime.WindowShow(globalApp.ctx)
					if goruntime.GOOS == "darwin" {
						runtime.WindowSetAlwaysOnTop(globalApp.ctx, true)
						runtime.WindowSetAlwaysOnTop(globalApp.ctx, false)
					}
				}
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
