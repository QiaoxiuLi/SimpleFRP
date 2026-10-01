package sysutil

import (
	"os"
	"strings"
	"testing"
)

func TestWindowsNativeProcessIdentity(t *testing.T) {
	first, err := inspectProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	second, err := inspectProcess(os.Getpid())
	if err != nil || first != second {
		t.Fatal("process identity is not stable")
	}
	executable, _ := os.Executable()
	if !strings.EqualFold(first.Executable, executable) || first.Started == "" {
		t.Fatal("native process identity is incomplete")
	}
	if _, err := inspectProcess(1 << 30); !os.IsNotExist(err) {
		t.Fatalf("missing process not distinguished from access failure: %v", err)
	}
}
