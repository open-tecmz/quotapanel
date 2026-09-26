//go:build windows

package platform

import (
	"os"
	"os/exec"
)

// RestartApp relaunches the app after a short delay: the current process must exit
// first to release the single-instance lock.
func RestartApp() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	command := "ping -n 2 127.0.0.1 >nul & start \"\" \"" + exe + "\""
	_ = exec.Command("cmd", "/c", command).Start()
}
