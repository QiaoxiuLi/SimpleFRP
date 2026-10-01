package service

import (
	"encoding/base64"
	"encoding/binary"
	"github.com/simplefrp/simplefrp/internal/sysutil"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func windowsScript(script string) error {
	return run("powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", windowsEncoded(script))
}
func windowsEncoded(script string) string {
	runes := utf16.Encode([]rune("$ErrorActionPreference='Stop';" + script))
	b := make([]byte, len(runes)*2)
	for i, v := range runes {
		binary.LittleEndian.PutUint16(b[2*i:], v)
	}
	return base64.StdEncoding.EncodeToString(b)
}
func windowsShortcut() string {
	return filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "SimpleFRP.lnk")
}
func installWindows() error {
	if err := validateWindows(); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return windowsScript("$w=New-Object -ComObject WScript.Shell;$s=$w.CreateShortcut(" + psQuote(windowsShortcut()) + ");$s.TargetPath=" + psQuote(exe) + ";$s.Arguments='desktop --instance " + sysutil.InstanceID() + "';$s.Description='SimpleFRP v2 " + sysutil.InstanceID() + "';$s.WindowStyle=7;$s.Save()")
}
func validateWindows() error {
	path := windowsShortcut()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return windowsScript("$w=New-Object -ComObject WScript.Shell;$s=$w.CreateShortcut(" + psQuote(path) + ");if($s.Description -ne 'SimpleFRP v2 " + sysutil.InstanceID() + "'){throw 'Existing startup entry was not changed'}")
}
func disableWindows() error {
	path := windowsShortcut()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return windowsScript("$p=" + psQuote(path) + ";$w=New-Object -ComObject WScript.Shell;$s=$w.CreateShortcut($p);if($s.Description -ne 'SimpleFRP v2 " + sysutil.InstanceID() + "'){throw 'Startup entry is not owned by this installation'};Remove-Item -LiteralPath $p")
}
