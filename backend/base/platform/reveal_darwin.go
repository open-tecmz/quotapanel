//go:build darwin

package platform

import "os/exec"

// RevealPath opens the given directory in the system file manager.
func RevealPath(path string) error {
	return exec.Command("open", path).Start()
}
