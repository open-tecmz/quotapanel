//go:build windows || linux

package platform

// MiniPanelSupported reports whether the menu bar mini panel is available.
// The mini panel is implemented natively on macOS only; Windows / Linux keep
// the existing tray behavior (left click shows the main window).
func MiniPanelSupported() bool { return false }

// ShowMiniPanel is a no-op on Windows / Linux.
func ShowMiniPanel(html string, height int, handlers MiniPanelHandlers) {}

// ReloadMiniPanel is a no-op on Windows / Linux.
func ReloadMiniPanel(html string) {}

// HideMiniPanel is a no-op on Windows / Linux.
func HideMiniPanel() {}
