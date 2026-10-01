package sysutil

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
)

func inspectWindowsProcess(pid int) (ProcessRecord, error) {
	r := ProcessRecord{PID: pid}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, uint32(pid))
	if err == windows.ERROR_INVALID_PARAMETER {
		return r, os.ErrNotExist
	}
	if err != nil {
		return r, err
	}
	defer windows.CloseHandle(handle)
	return inspectWindowsHandle(handle, pid)
}
func inspectWindowsHandle(handle windows.Handle, pid int) (ProcessRecord, error) {
	r := ProcessRecord{PID: pid}
	var code uint32
	if err := windows.GetExitCodeProcess(handle, &code); err != nil {
		return r, err
	}
	const stillActive = 259
	if code != stillActive {
		if err := waitWindowsExit(handle); err != nil {
			return r, err
		}
		return r, os.ErrNotExist
	}
	var created, exited, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &created, &exited, &kernel, &user); err != nil {
		return r, err
	}
	name := make([]uint16, 32768)
	size := uint32(len(name))
	if err := windows.QueryFullProcessImageName(handle, 0, &name[0], &size); err != nil {
		return r, err
	}
	r.Executable = windows.UTF16ToString(name[:size])
	r.Started = fmt.Sprintf("%d:%d", created.HighDateTime, created.LowDateTime)
	return r, nil
}
func waitWindowsExit(handle windows.Handle) error {
	state, err := windows.WaitForSingleObject(handle, 5000)
	if err != nil {
		return err
	}
	if state != windows.WAIT_OBJECT_0 {
		return fmt.Errorf("owned Windows process did not fully exit; files were retained")
	}
	return nil
}
func stopWindowsProcess(saved ProcessRecord) error {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, uint32(saved.PID))
	if err == windows.ERROR_INVALID_PARAMETER {
		return nil
	}
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	current, err := inspectWindowsHandle(handle, saved.PID)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if current != saved {
		return fmt.Errorf("process identity changed; it was not stopped")
	}
	if err := windows.TerminateProcess(handle, 1); err != nil {
		return err
	}
	return waitWindowsExit(handle)
}
