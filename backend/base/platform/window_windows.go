//go:build windows

package platform

// Frameless 表示是否使用无边框窗口。
// Windows 使用无边框窗口，由前端自绘模拟系统窗口按钮。
func Frameless() bool { return true }

// HasNativeWindowControls 表示窗口控制按钮是否由系统原生绘制。
func HasNativeWindowControls() bool { return false }
