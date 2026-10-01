package cli

import (
	"github.com/simplefrp/simplefrp/internal/sysutil"
	"golang.org/x/sys/windows"
	"os"
	"unsafe"
)

func configureTerminal() bool {
	title, _ := windows.UTF16PtrFromString("SimpleFRP Status - " + sysutil.InstanceID())
	windows.NewLazySystemDLL("kernel32.dll").NewProc("SetConsoleTitleW").Call(uintptr(unsafe.Pointer(title)))
	handle := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if windows.GetConsoleMode(handle, &mode) != nil {
		return false
	}
	return windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) == nil
}
