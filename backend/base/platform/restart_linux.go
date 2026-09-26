//go:build linux

package platform

import (
	"os"
	"os/exec"
	"strings"
)

// RestartApp relaunches the app after a short delay: the current process must exit
// first to release the single-instance lock.
func RestartApp() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	command := "sleep 1; nohup " + shellQuote(exe) + quoteArgs(os.Args[1:]) + " >/dev/null 2>&1 &"
	_ = exec.Command("/bin/sh", "-c", command).Start()
}

func quoteArgs(args []string) string {
	quoted := ""
	for _, arg := range args {
		quoted += " " + shellQuote(arg)
	}
	return quoted
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
