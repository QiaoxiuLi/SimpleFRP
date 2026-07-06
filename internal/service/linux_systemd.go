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
		return run("launchctl", "load", "/Library/LaunchDaemons/com.simplefrp.client.plist")
	case "windows":
		return run("schtasks", "/Run", "/TN", windowsTaskName)
	default:
		return nil
	}
}

func stopService(role app.Role) error {
	switch runtime.GOOS {
	case "linux":
		return run("systemctl", "stop", unitName(role))
	case "darwin":
		return run("launchctl", "unload", "/Library/LaunchDaemons/com.simplefrp.client.plist")
	case "windows":
		return run("schtasks", "/End", "/TN", windowsTaskName)
	default:
		return nil
	}
}

func enableService(role app.Role) error {
	switch runtime.GOOS {
	case "linux":
		return run("systemctl", "enable", unitName(role))
	case "windows":
		return nil
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
		_ = run("launchctl", "unload", "/Library/LaunchDaemons/com.simplefrp.client.plist")
		_ = os.Remove("/Library/LaunchDaemons/com.simplefrp.client.plist")
		return nil
	case "windows":
		return run("schtasks", "/Delete", "/TN", windowsTaskName, "/F")
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
