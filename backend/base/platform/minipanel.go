package platform

// MiniPanelHandlers holds menu bar mini panel interaction callbacks.
//
// OnAction receives interactions from the panel: action is "refresh" (reload
// quota data) or "open" (open the main window). accountID is greater than zero
// when the user clicked a specific account row, zero otherwise.
type MiniPanelHandlers struct {
	OnAction func(action string, accountID int)
}
