package service

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/simplefrp/simplefrp/internal/sysutil"
	"os"
	"path/filepath"
	"strings"
)

func xmlText(s string) string { var b bytes.Buffer; xml.EscapeText(&b, []byte(s)); return b.String() }
func macLabel() string        { return "com.simplefrp.client" }
func macDomain() string       { return fmt.Sprintf("gui/%d", os.Getuid()) }
func macPlist(label string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist")
}
func installMac() error {
	if err := ValidateStartup("client"); err != nil {
		return err
	}
	if os.Getuid() == 0 {
		return fmt.Errorf("install the macOS client as the logged-in user, not root")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	for _, view := range []bool{false, true} {
		label := macLabel()
		arguments := []string{exe, "daemon", "--role", "client", "--instance", sysutil.InstanceID()}
		keep := "<true/>"
		if view {
			label += ".status"
			arguments = []string{exe, "desktop", "--instance", sysutil.InstanceID()}
			keep = "<false/>"
		}
		path := macPlist(label)
		if b, err := os.ReadFile(path); err == nil && !bytes.Contains(b, []byte(sysutil.InstanceID())) {
			return fmt.Errorf("existing launch agent was not changed: %s", path)
		}
		if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		var argumentXML strings.Builder
		for _, arg := range arguments {
			argumentXML.WriteString("<string>" + xmlText(arg) + "</string>")
		}
		body := fmt.Sprintf(`<?xml version="1.0"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>Label</key><string>%s</string><key>ProgramArguments</key><array>%s</array><key>RunAtLoad</key><true/><key>KeepAlive</key>%s<key>ThrottleInterval</key><integer>5</integer><key>StandardOutPath</key><string>%s</string><key>StandardErrorPath</key><string>%s</string></dict></plist>`, label, argumentXML.String(), keep, xmlText(filepath.Join(sysutil.LogDir(), "launch.log")), xmlText(filepath.Join(sysutil.LogDir(), "launch-error.log")))
		if err = os.WriteFile(path, []byte(body), 0644); err != nil {
			return err
		}
	}
	return nil
}
func startMac() error {
	target := macDomain() + "/" + macLabel()
	if execExists("launchctl", "print", target) {
		return nil
	}
	return run("launchctl", "bootstrap", macDomain(), macPlist(macLabel()))
}
func stopMac() error {
	for _, label := range []string{macLabel(), macLabel() + ".status"} {
		target := macDomain() + "/" + label
		if execExists("launchctl", "print", target) {
			if err := run("launchctl", "bootout", target); err != nil {
				return err
			}
		}
	}
	return nil
}
func disableMac() error {
	for _, label := range []string{macLabel(), macLabel() + ".status"} {
		if err := removeOwnedStartup(macPlist(label)); err != nil {
			return err
		}
	}
	return nil
}
