package service

import (
	"fmt"
	"github.com/simplefrp/simplefrp/internal/app"
	"github.com/simplefrp/simplefrp/internal/sysutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func run(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %w (%s)", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}
func execExists(name string, args ...string) bool { return exec.Command(name, args...).Run() == nil }
func unitName(role app.Role) string               { return "simplefrp-" + string(role) }
func ValidateStartup(role app.Role) error {
	if runtime.GOOS == "windows" {
		return validateWindows()
	}
	paths := []string{filepath.Join("/etc/systemd/system", unitName(role)+".service")}
	if runtime.GOOS == "darwin" {
		paths = []string{macPlist(macLabel()), macPlist(macLabel() + ".status")}
	}
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !strings.Contains(string(b), sysutil.InstanceID()) {
			return fmt.Errorf("existing startup entry was not changed: %s", path)
		}
	}
	return nil
}
func daemonArgs(role app.Role) []string {
	args := []string{"daemon", "--role", string(role), "--instance", sysutil.InstanceID()}
	if home := os.Getenv("SIMPLEFRP_HOME"); home != "" {
		args = append(args, "--home", home)
	}
	return args
}
func statusArgs() []string {
	args := []string{"state", "--watch", "--instance", sysutil.InstanceID()}
	if home := os.Getenv("SIMPLEFRP_HOME"); home != "" {
		args = append(args, "--home", home)
	}
	return args
}
func installService(role app.Role) error { return enableService(role) }
func startService(role app.Role) error {
	if sysutil.ProcessRunning("daemon") {
		return nil
	}
	switch runtime.GOOS {
	case "linux":
		return run("systemctl", "start", unitName(role))
	case "darwin":
		return startMac()
	case "windows":
		return StartIsolated(role)
	}
	return fmt.Errorf("unsupported operating system")
}
func stopService(role app.Role) error {
	switch runtime.GOOS {
	case "linux":
		if !execExists("systemctl", "cat", unitName(role)) {
			return sysutil.StopOwnedProcess("daemon")
		}
		if err := run("systemctl", "stop", unitName(role)); err != nil {
			return err
		}
	case "darwin":
		if err := stopMac(); err != nil {
			return err
		}
	}
	return sysutil.StopOwnedProcess("daemon")
}
func enableService(role app.Role) error {
	if _, err := sysutil.LoadReceipt(); err != nil {
		return fmt.Errorf("run the v2 installer first; existing startup entries were not changed")
	}
	switch runtime.GOOS {
	case "linux":
		return installLinux(role)
	case "darwin":
		return installMac()
	case "windows":
		return installWindows()
	}
	return fmt.Errorf("unsupported operating system")
}
func disableService(role app.Role) error {
	switch runtime.GOOS {
	case "linux":
		if execExists("systemctl", "cat", unitName(role)) {
			if err := run("systemctl", "disable", unitName(role)); err != nil {
				return err
			}
		}
		path := filepath.Join("/etc/systemd/system", unitName(role)+".service")
		if err := removeOwnedStartup(path); err != nil {
			return err
		}
		return run("systemctl", "daemon-reload")
	case "darwin":
		return disableMac()
	case "windows":
		return disableWindows()
	}
	return nil
}
func removeOwnedStartup(path string) error {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !strings.Contains(string(b), sysutil.InstanceID()) {
		return fmt.Errorf("startup entry is not owned by this installation: %s", path)
	}
	return os.Remove(path)
}
func installLinux(role app.Role) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	path := filepath.Join("/etc/systemd/system", unitName(role)+".service")
	if b, err := os.ReadFile(path); err == nil && !strings.Contains(string(b), sysutil.InstanceID()) {
		return fmt.Errorf("existing service was not overwritten")
	}
	body := "[Unit]\nDescription=SimpleFRP v2\nAfter=network-online.target\nWants=network-online.target\n[Service]\nType=simple\nUser=simplefrp\nGroup=simplefrp\nExecStart=" + exe + " daemon --role " + string(role) + " --instance " + sysutil.InstanceID() + "\nRestart=on-failure\nRestartSec=3\n[Install]\nWantedBy=multi-user.target\n"
	if err = os.WriteFile(path, []byte(body), 0644); err != nil {
		return err
	}
	if err = run("systemctl", "daemon-reload"); err != nil {
		return err
	}
	return run("systemctl", "enable", unitName(role))
}
func StartIsolated(role app.Role) error {
	if sysutil.ProcessRunning("daemon") {
		return nil
	}
	if err := sysutil.EnsureBaseDirs(); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	log, err := os.OpenFile(filepath.Join(sysutil.LogDir(), "daemon.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer log.Close()
	cmd := exec.Command(exe, daemonArgs(role)...)
	cmd.Stdout = log
	cmd.Stderr = log
	detach(cmd)
	if err = cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	for i := 0; i < 40; i++ {
		if sysutil.ProcessRunning("daemon") {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("daemon did not start; see %s", filepath.Join(sysutil.LogDir(), "daemon.log"))
}
func OpenStatus() error {
	if runtime.GOOS == "linux" {
		return nil
	}
	if os.Getenv("SIMPLEFRP_HOME") != "" && os.Getenv("SIMPLEFRP_DESKTOP_TEST") != "1" {
		return nil
	}
	if sysutil.ProcessRunning("monitor") {
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		arguments := []string{}
		for _, arg := range statusArgs() {
			arguments = append(arguments, `"`+strings.ReplaceAll(arg, `"`, `\"`)+`"`)
		}
		if err := windowsScript("Start-Process -FilePath " + psQuote(exe) + " -ArgumentList " + psQuote(strings.Join(arguments, " ")) + " -WindowStyle Normal"); err != nil {
			return err
		}
		return waitStatusViewer()
	}
	path := filepath.Join(sysutil.DataDir(), "SimpleFRP-Status.command")
	parts := []string{shQuote(exe)}
	for _, a := range statusArgs() {
		parts = append(parts, shQuote(a))
	}
	body := "#!/bin/sh\nexec " + strings.Join(parts, " ") + "\n"
	if err = os.WriteFile(path, []byte(body), 0700); err != nil {
		return err
	}
	if err = run("open", "-a", "Terminal", path); err != nil {
		return err
	}
	return waitStatusViewer()
}
func waitStatusViewer() error {
	for attempt := 0; attempt < 80; attempt++ {
		if sysutil.ProcessRunning("monitor") {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("status viewer did not register; forwarding daemon remains independent")
}
func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }
func RemoveInstalledBinary(r sysutil.Receipt) error {
	if !r.OwnBinary {
		return nil
	}
	if runtime.GOOS == "linux" && os.Getenv("SIMPLEFRP_HOME") == "" {
		if os.Getenv("SIMPLEFRP_PACKAGE_REMOVING") == "1" {
			return nil
		}
		if _, err := exec.LookPath("dpkg"); err == nil {
			return run("dpkg", "--purge", "simplefrp-"+r.Role)
		}
		return run("rpm", "-e", "simplefrp-"+r.Role)
	}
	if runtime.GOOS == "windows" {
		cleanup := "Start-Sleep -Seconds 2;for($i=0;$i -lt 30;$i++){try{if(Test-Path -LiteralPath " + psQuote(r.Binary) + "){Remove-Item -LiteralPath " + psQuote(r.Binary) + " -Force};break}catch{Start-Sleep -Milliseconds 200}};if(Test-Path -LiteralPath " + psQuote(r.Binary) + "){exit 1};$d=" + psQuote(filepath.Dir(r.Binary)) + ";if((Get-ChildItem -LiteralPath $d -Force | Measure-Object).Count -eq 0){Remove-Item -LiteralPath $d}"
		return windowsScript("Start-Process powershell.exe -ArgumentList @('-NoProfile','-NonInteractive','-EncodedCommand'," + psQuote(windowsEncoded(cleanup)) + ") -WindowStyle Hidden")
	}
	if err := os.Remove(r.Binary); err != nil && !os.IsNotExist(err) {
		return err
	}
	parent := filepath.Dir(r.Binary)
	if home := os.Getenv("SIMPLEFRP_HOME"); home != "" {
		relative, err := filepath.Rel(home, parent)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative) {
			_ = os.Remove(parent)
		}
	}
	return nil
}
