//go:build windows || linux

package platform

import (
	"runtime"

	"fyne.io/systray"
)

// statusBarItems keeps tray menu item references so labels can be refreshed on locale change.
var statusBarItems struct {
	show    *systray.MenuItem
	restart *systray.MenuItem
	quit    *systray.MenuItem
}

// InitStatusBar creates the Windows / Linux system tray icon.
// Left click shows the main window; right click opens the "Show / Restart / Quit" menu.
func InitStatusBar(icon []byte, labels StatusBarLabels, handlers StatusBarHandlers) {
	// The left click callback must be set before the event loop starts: on Linux the
	// registration decides whether the icon is a plain menu entry (ItemIsMenu).
	systray.SetOnTapped(func() {
		if handlers.OnShow != nil {
			handlers.OnShow()
		}
	})

	go func() {
		// On Windows the tray builds its own window and message loop, which must run on the
		// same OS thread that created the window.
		runtime.LockOSThread()
		systray.Run(func() {
			systray.SetIcon(icon)
			systray.SetTooltip("QuotaPanel")
			statusBarItems.show = systray.AddMenuItem(labels.Show, labels.Show)
			statusBarItems.restart = systray.AddMenuItem(labels.Restart, labels.Restart)
			systray.AddSeparator()
			statusBarItems.quit = systray.AddMenuItem(labels.Quit, labels.Quit)
			go func() {
				for {
					select {
					case <-statusBarItems.show.ClickedCh:
						if handlers.OnShow != nil {
							handlers.OnShow()
						}
					case <-statusBarItems.restart.ClickedCh:
						if handlers.OnRestart != nil {
							handlers.OnRestart()
						}
					case <-statusBarItems.quit.ClickedCh:
						if handlers.OnQuit != nil {
							handlers.OnQuit()
						}
						return
					}
				}
			}()
		}, func() {})
	}()
}

// UpdateStatusBarMenu refreshes the tray context menu labels (called on locale change).
func UpdateStatusBarMenu(labels StatusBarLabels) {
	if statusBarItems.show != nil {
		statusBarItems.show.SetTitle(labels.Show)
	}
	if statusBarItems.restart != nil {
		statusBarItems.restart.SetTitle(labels.Restart)
	}
	if statusBarItems.quit != nil {
		statusBarItems.quit.SetTitle(labels.Quit)
	}
}

// QuitStatusBar removes the tray icon when the application exits.
func QuitStatusBar() {
	systray.Quit()
}
