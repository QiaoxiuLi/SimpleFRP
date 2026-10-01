package sysutil

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go.etcd.io/bbolt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type ProcessRecord struct {
	PID         int
	Executable  string
	Started     string
	CommandHash string
}
type Receipt struct {
	Version         int    `json:"version"`
	Role            string `json:"role"`
	Binary          string `json:"binary"`
	OwnBinary       bool   `json:"own_binary"`
	PathAdded       bool   `json:"path_added"`
	PathFile        string `json:"path_file,omitempty"`
	PathFileCreated bool   `json:"path_file_created,omitempty"`
	ServiceUID      string `json:"service_uid,omitempty"`
	OwnServiceUser  bool   `json:"own_service_user,omitempty"`
	OwnServiceGroup bool   `json:"own_service_group,omitempty"`
}

func ReceiptPath() string { return filepath.Join(ConfigDir(), "installation.json") }
func LoadReceipt() (Receipt, error) {
	var r Receipt
	b, err := os.ReadFile(ReceiptPath())
	if err != nil {
		return r, err
	}
	err = json.Unmarshal(b, &r)
	if err == nil && (r.Version != 2 || (r.Role != "server" && r.Role != "client") || !filepath.IsAbs(r.Binary)) {
		err = fmt.Errorf("not a valid v2 installation receipt")
	}
	return r, err
}
func InstanceID() string { s := sha256.Sum256([]byte(ConfigDir())); return hex.EncodeToString(s[:8]) }
func SafeHome(path string) (string, error) {
	p, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	home, _ := os.UserHomeDir()
	if p == string(filepath.Separator) || p == home || filepath.Dir(p) == p {
		return "", fmt.Errorf("choose a dedicated SimpleFRP home, not a filesystem or user home root")
	}
	return p, nil
}
func TerminationSignal() os.Signal { return syscall.SIGTERM }
func ProcessLock(kind string) (func(), error) {
	if err := EnsureBaseDirs(); err != nil {
		return nil, err
	}
	db, err := bbolt.Open(filepath.Join(DataDir(), kind+".lock"), 0600, &bbolt.Options{Timeout: 100 * time.Millisecond})
	if err != nil {
		return nil, fmt.Errorf("this installation's %s is already running", kind)
	}
	return func() { db.Close() }, nil
}
func psLiteral(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func processPS(pid int, field string) ([]byte, error) {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", field+"=").Output()
	if failure, ok := err.(*exec.ExitError); ok && failure.ExitCode() == 1 && len(strings.TrimSpace(string(out))) == 0 {
		return nil, os.ErrNotExist
	}
	return out, err
}
func inspectProcess(pid int) (ProcessRecord, error) {
	r := ProcessRecord{PID: pid}
	var command string
	if runtime.GOOS == "windows" {
		return inspectWindowsProcess(pid)
	} else {
		out, err := processPS(pid, "lstart")
		if err != nil {
			if failure, ok := err.(*exec.ExitError); ok && failure.ExitCode() == 1 && len(strings.TrimSpace(string(out))) == 0 {
				return r, os.ErrNotExist
			}
			return r, err
		}
		r.Started = strings.TrimSpace(string(out))
		out, err = processPS(pid, "command")
		if err != nil {
			return r, err
		}
		command = strings.TrimSpace(string(out))
		if runtime.GOOS == "linux" {
			r.Executable, err = os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
			if err != nil {
				return r, err
			}
		} else {
			out, err = processPS(pid, "comm")
			if err != nil {
				return r, err
			}
			r.Executable = strings.TrimSpace(string(out))
		}
	}
	sum := sha256.Sum256([]byte(command))
	r.CommandHash = hex.EncodeToString(sum[:])
	if r.Executable == "" || r.Started == "" {
		return r, fmt.Errorf("process identity unavailable")
	}
	return r, nil
}
func WriteProcess(kind, instance string) error {
	r, err := inspectProcess(os.Getpid())
	if err != nil {
		return err
	}
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(DataDir(), kind+".pid.json"), b, 0600)
}
func RemoveProcess(kind string) {
	path := filepath.Join(DataDir(), kind+".pid.json")
	b, _ := os.ReadFile(path)
	var r ProcessRecord
	if json.Unmarshal(b, &r) == nil && r.PID == os.Getpid() {
		os.Remove(path)
	}
}
func ownedProcess(kind string) (ProcessRecord, bool, error) {
	path := filepath.Join(DataDir(), kind+".pid.json")
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ProcessRecord{}, false, nil
	}
	if err != nil {
		return ProcessRecord{}, false, err
	}
	var saved ProcessRecord
	if err = json.Unmarshal(b, &saved); err != nil {
		return saved, false, err
	}
	current, err := inspectProcess(saved.PID)
	if err != nil {
		if os.IsNotExist(err) {
			return saved, false, nil
		}
		return saved, false, fmt.Errorf("cannot verify %s process identity: %w", kind, err)
	}
	if current != saved {
		return saved, false, fmt.Errorf("stale %s PID now belongs to a different process; it was not stopped", kind)
	}
	return current, true, nil
}
func ProcessRunning(kind string) bool { _, alive, _ := ownedProcess(kind); return alive }
func StopOwnedProcess(kind string) error {
	r, alive, err := ownedProcess(kind)
	if err != nil {
		return err
	}
	if !alive {
		return nil
	}
	if r.PID == os.Getpid() {
		return nil
	}
	if runtime.GOOS == "windows" {
		return stopWindowsProcess(r)
	}
	p, err := os.FindProcess(r.PID)
	if err != nil {
		return err
	}
	err = p.Signal(syscall.SIGTERM)
	if err != nil {
		return err
	}
	for i := 0; i < 50; i++ {
		current, inspectErr := inspectProcess(r.PID)
		if os.IsNotExist(inspectErr) || (inspectErr == nil && current != r) {
			return nil
		}
		if inspectErr != nil {
			return inspectErr
		}
		time.Sleep(100 * time.Millisecond)
	}
	current, ok, err := ownedProcess(kind)
	if err != nil {
		return err
	}
	if ok && current == r {
		if err = p.Kill(); err != nil {
			return err
		}
		for i := 0; i < 50; i++ {
			current, inspectErr := inspectProcess(r.PID)
			if os.IsNotExist(inspectErr) || (inspectErr == nil && current != r) {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
		return fmt.Errorf("owned %s process did not exit; runtime files were retained", kind)
	}
	return nil
}
