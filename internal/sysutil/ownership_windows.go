package sysutil

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
)

func inspectWindowsProcess(pid int) (ProcessRecord, error) {
	r := ProcessRecord{PID: pid}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err == windows.ERROR_INVALID_PARAMETER {
		return r, os.ErrNotExist
	}
	if err != nil {
		return r, err
	}
	defer windows.CloseHandle(handle)
	var code uint32
	if err = windows.GetExitCodeProcess(handle, &code); err != nil {
		return r, err
	}
	const stillActive = 259
	if code != stillActive {
		return r, os.ErrNotExist
	}
	var created, exited, kernel, user windows.Filetime
	if err = windows.GetProcessTimes(handle, &created, &exited, &kernel, &user); err != nil {
		return r, err
	}
	name := make([]uint16, 32768)
	size := uint32(len(name))
	if err = windows.QueryFullProcessImageName(handle, 0, &name[0], &size); err != nil {
		return r, err
	}
	r.Executable = windows.UTF16ToString(name[:size])
	r.Started = fmt.Sprintf("%d:%d", created.HighDateTime, created.LowDateTime)
	return r, nil
}
