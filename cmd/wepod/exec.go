package main

import "os/exec"

// newExecCmd is a tiny shim so main_test could fake interactive sudo if needed.
var newExecCmd = func(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
