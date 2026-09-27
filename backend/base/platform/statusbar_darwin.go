//go:build darwin

package platform

/*
// The menu bar / mini panel implementation is written in ARC style (strong
// properties, no manual retain/release), so this package must be compiled with
// ARC: cgo otherwise compiles Objective-C in manual reference counting mode,
// where dispatch_async blocks do not retain the objects they capture and the
// autoreleased strings passed to them are freed before the block runs.
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#include <stdlib.h>

// Implemented in statusbar_darwin.m
void qpStatusBarInit(const void *iconBytes, int length, const char *tooltip);
void qpStatusBarSetMenu(const char *show, const char *restart, const char *quit);
*/
import "C"

import "unsafe"

// statusBarHandlers stores callbacks injected by the app, triggered from Objective-C.
var statusBarHandlers StatusBarHandlers

//export quotapanelStatusBarLeftClick
func quotapanelStatusBarLeftClick() {
	if statusBarHandlers.OnShow != nil {
		statusBarHandlers.OnShow()
	}
}

//export quotapanelStatusBarShow
func quotapanelStatusBarShow() {
	if statusBarHandlers.OnShow != nil {
		statusBarHandlers.OnShow()
	}
}

//export quotapanelStatusBarRestart
func quotapanelStatusBarRestart() {
	if statusBarHandlers.OnRestart != nil {
		statusBarHandlers.OnRestart()
	}
}

//export quotapanelStatusBarQuit
func quotapanelStatusBarQuit() {
	if statusBarHandlers.OnQuit != nil {
		statusBarHandlers.OnQuit()
	}
}

// InitStatusBar creates the macOS menu bar status item.
func InitStatusBar(icon []byte, labels StatusBarLabels, handlers StatusBarHandlers) {
	statusBarHandlers = handlers
	var iconPtr unsafe.Pointer
	if len(icon) > 0 {
		iconPtr = unsafe.Pointer(&icon[0])
	}
	C.qpStatusBarInit(iconPtr, C.int(len(icon)), C.CString("QuotaPanel"))
	UpdateStatusBarMenu(labels)
}

// UpdateStatusBarMenu refreshes the right-click menu labels (called on locale change).
func UpdateStatusBarMenu(labels StatusBarLabels) {
	C.qpStatusBarSetMenu(C.CString(labels.Show), C.CString(labels.Restart), C.CString(labels.Quit))
}

// QuitStatusBar needs no manual cleanup on macOS: the menu bar item is removed with the process.
func QuitStatusBar() {}
