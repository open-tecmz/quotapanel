package platform

// StatusBarLabels holds the menu bar context menu labels, following the app locale.
type StatusBarLabels struct {
	Show    string
	Restart string
	Quit    string
}

// StatusBarHandlers holds menu bar icon callbacks.
// OnShow is used for the left click and the "Show" menu item,
// OnRestart for restart, OnQuit for quit.
type StatusBarHandlers struct {
	OnShow    func()
	OnRestart func()
	OnQuit    func()
}
