package sysutil

import (
	"os/exec"
	"runtime"
)

func StopSimpleFRPProcesses() error {
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("taskkill", "/IM", "simplefrp.exe", "/F").Run()
	case "darwin":
		_ = exec.Command("pkill", "-f", "simplefrp").Run()
	default:
		_ = exec.Command("pkill", "-f", "simplefrp").Run()
	}
	return nil
}
