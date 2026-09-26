//go:build darwin

package platform

// Frameless 表示是否使用无边框窗口。
// macOS 保留系统窗口边框，通过透明标题栏 + 全尺寸内容展示系统原生红绿灯。
func Frameless() bool { return false }

// HasNativeWindowControls 表示窗口控制按钮是否由系统原生绘制。
func HasNativeWindowControls() bool { return true }
