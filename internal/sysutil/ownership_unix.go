//go:build !windows

package sysutil

import "fmt"

func inspectWindowsProcess(pid int) (ProcessRecord, error) {
	return ProcessRecord{}, fmt.Errorf("Windows process inspection is unavailable on this platform")
}
