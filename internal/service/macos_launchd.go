package service

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/simplefrp/simplefrp/internal/sysutil"
	"os"
	"path/filepath"
)

const macOSLaunchAgent = "com.simplefrp.client"

func macPlistPath() string {
	if os.Getuid() == 0 {
		return "/Library/LaunchDaemons/" + macOSLaunchAgent + ".plist"
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", macOSLaunchAgent+".plist")
}
func macDomain() string {
	if os.Getuid() == 0 {
		return "system"
	}
	return fmt.Sprintf("gui/%d", os.Getuid())
}
func xmlText(text string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(text))
	return b.String()
}
func installMacService() error {
	if err := sysutil.EnsureBaseDirs(); err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	path := macPlistPath()
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>com.simplefrp.client</string>
<key>ProgramArguments</key><array><string>%s</string><string>daemon</string><string>--role</string><string>client</string></array>
<key>RunAtLoad</key><true/><key>KeepAlive</key><true/>
<key>ThrottleInterval</key><integer>5</integer>
<key>EnvironmentVariables</key><dict><key>SIMPLEFRP_CONFIG_DIR</key><string>%s</string></dict>
<key>StandardOutPath</key><string>%s</string><key>StandardErrorPath</key><string>%s</string>
</dict></plist>
`, xmlText(executable), xmlText(sysutil.ConfigDir()), xmlText(filepath.Join(sysutil.LogDir(), "client.log")), xmlText(filepath.Join(sysutil.LogDir(), "client-error.log")))
	return os.WriteFile(path, []byte(body), 0644)
}
func startMacService() error {
	target := macDomain() + "/" + macOSLaunchAgent
	if run("launchctl", "print", target) == nil {
		return run("launchctl", "kickstart", "-k", target)
	}
	return run("launchctl", "bootstrap", macDomain(), macPlistPath())
}
func stopMacService() error {
	target := macDomain() + "/" + macOSLaunchAgent
	if run("launchctl", "print", target) != nil {
		return nil
	}
	return run("launchctl", "bootout", target)
}
