//go:build darwin

package platform

/*
#cgo LDFLAGS: -framework Cocoa -framework WebKit

#include <stdlib.h>

// Implemented in statusbar_darwin.m
void qpMiniPanelShow(const char *html, int height);
void qpMiniPanelReload(const char *html);
void qpMiniPanelHide(void);
int qpMiniPanelIsShown(void);
*/
import "C"

import "unsafe"

// miniPanelHandlers stores callbacks injected by the app, triggered from the
// mini panel web view through the "quotapanel" script message handler.
var miniPanelHandlers MiniPanelHandlers

//export quotapanelMiniPanelAction
func quotapanelMiniPanelAction(action *C.char, accountID C.int) {
	if miniPanelHandlers.OnAction == nil {
		return
	}
	miniPanelHandlers.OnAction(C.GoString(action), int(accountID))
}

// MiniPanelSupported reports whether the menu bar mini panel is available.
// It is implemented natively on macOS only.
func MiniPanelSupported() bool { return true }

// ShowMiniPanel shows or hides the menu bar mini panel with the given HTML.
// height is the initial estimated height in logical pixels; the page corrects
// it after load. Clicking the status item again hides the panel.
func ShowMiniPanel(html string, height int, handlers MiniPanelHandlers) {
	miniPanelHandlers = handlers
	cHTML := C.CString(html)
	defer C.free(unsafe.Pointer(cHTML))
	C.qpMiniPanelShow(cHTML, C.int(height))
}

// ReloadMiniPanel refreshes the panel content while it is shown; it is a no-op
// when the panel is hidden.
func ReloadMiniPanel(html string) {
	if C.qpMiniPanelIsShown() == 0 {
		return
	}
	cHTML := C.CString(html)
	defer C.free(unsafe.Pointer(cHTML))
	C.qpMiniPanelReload(cHTML)
}

// HideMiniPanel closes the menu bar mini panel.
func HideMiniPanel() { C.qpMiniPanelHide() }
