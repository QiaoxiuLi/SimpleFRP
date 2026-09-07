package service

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/simplefrp/simplefrp/internal/app"
)

func installService(role app.Role) error {
	if runtime.GOOS == "linux" {
		return run("systemctl", "daemon-reload")
	}
	return nil
}

func startService(role app.Role) error {
	switch runtime.GOOS {
	case "linux":
		return run("systemctl", "start", unitName(role))
	case "darwin":
		return startMacService()
	case "windows":
		return startWindowsService()
	default:
		return nil
	}
}

func stopService(role app.Role) error {
	switch runtime.GOOS {
	case "linux":
		return run("systemctl", "stop", unitName(role))
	case "darwin":
		return stopMacService()
	case "windows":
		return stopWindowsService()
	default:
		return nil
	}
}

func enableService(role app.Role) error {
	switch runtime.GOOS {
	case "linux":
		return run("systemctl", "enable", unitName(role))
	case "darwin":
		return installMacService()
	case "windows":
		return installWindowsService()
	default:
		return nil
	}
}

func disableService(role app.Role) error {
	switch runtime.GOOS {
	case "linux":
		_ = run("systemctl", "disable", unitName(role))
		for _, path := range linuxServicePaths(role) {
			_ = os.Remove(path)
		}
		_ = run("systemctl", "daemon-reload")
		return nil
	case "darwin":
		_ = stopMacService()
		_ = os.Remove(macPlistPath())
		return nil
	case "windows":
		_ = stopWindowsService()
		return os.Remove(windowsShortcut())
	default:
		return nil
	}
}

func unitName(role app.Role) string {
	return fmt.Sprintf("simplefrp-%s", role)
}

func linuxServicePaths(role app.Role) []string {
	name := unitName(role) + ".service"
	return []string{
		"/etc/systemd/system/" + name,
		"/usr/lib/systemd/system/" + name,
		"/lib/systemd/system/" + name,
	}
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}
