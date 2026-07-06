package sysutil

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func StopSimpleFRPProcesses() error {
	currentPID := os.Getpid()
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("taskkill", "/F", "/IM", "simplefrp.exe", "/FI", fmt.Sprintf("PID ne %d", currentPID)).Run()
	default:
		out, err := exec.Command("pgrep", "-f", "simplefrp").Output()
		if err != nil {
			return nil
		}
		for _, line := range strings.Fields(string(out)) {
			pid, err := strconv.Atoi(line)
			if err != nil || pid == currentPID {
				continue
			}
			if proc, err := os.FindProcess(pid); err == nil {
				_ = proc.Kill()
			}
		}
	}
	return nil
}
