//go:build linux

package platform

import "os/exec"

// RevealPath opens the given directory in the system file manager.
func RevealPath(path string) error {
	return exec.Command("xdg-open", path).Start()
}
