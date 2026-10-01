package sysutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
func TestWindowsSleepingChild(t *testing.T) {
	if os.Getenv("SIMPLEFRP_PROCESS_CHILD") == "1" {
		time.Sleep(30 * time.Second)
	}
}
func TestWindowsOwnedStopReleasesFileHandles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "child.log")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestWindowsSleepingChild$")
	cmd.Env = append(os.Environ(), "SIMPLEFRP_PROCESS_CHILD=1")
	cmd.Stdout, cmd.Stderr = file, file
	if err := cmd.Start(); err != nil {
		file.Close()
		t.Fatal(err)
	}
	file.Close()
	defer cmd.Wait()
	defer cmd.Process.Kill()
	owned, err := inspectProcess(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	stale := owned
	stale.Started += "changed"
	if err := stopWindowsProcess(stale); err == nil {
		t.Fatal("stale identity was allowed to terminate a process")
	}
	if err := stopWindowsProcess(owned); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("process termination returned before its file handles were released: %v", err)
	}
}
