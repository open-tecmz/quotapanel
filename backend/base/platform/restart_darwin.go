//go:build darwin

package platform

import (
	"os"
	"os/exec"
	"strings"
)

// RestartApp relaunches the app after a short delay: the current process must exit
// first to release the single-instance lock. Inside a .app bundle it reopens the
// bundle via open -n so the full macOS app context is restored.
func RestartApp() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	quotedArgs := quoteArgs(os.Args[1:])
	if idx := strings.Index(exe, ".app/Contents/MacOS/"); idx >= 0 {
		bundle := exe[:idx+len(".app")]
		command := "sleep 1; open -n " + shellQuote(bundle)
		if quotedArgs != "" {
			command += " --args" + quotedArgs
		}
		_ = exec.Command("/bin/sh", "-c", command).Start()
		return
	}
	command := "sleep 1; nohup " + shellQuote(exe) + quotedArgs + " >/dev/null 2>&1 &"
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
