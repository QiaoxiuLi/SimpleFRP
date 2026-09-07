package service

import (
	"encoding/base64"
	"encoding/binary"
	"github.com/simplefrp/simplefrp/internal/sysutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

const windowsTaskName = "SimpleFRP"

func psQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }
func windowsScript(script string) error {
	encoded := utf16.Encode([]rune("$ErrorActionPreference='Stop';" + script))
	data := make([]byte, len(encoded)*2)
	for i, v := range encoded {
		binary.LittleEndian.PutUint16(data[i*2:], v)
	}
	return exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(data)).Run()
}
func windowsShortcut() string {
	return filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "SimpleFRP.lnk")
}
func installWindowsService() error {
	if err := sysutil.EnsureBaseDirs(); err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	return windowsScript("$w=New-Object -ComObject WScript.Shell;$s=$w.CreateShortcut(" + psQuote(windowsShortcut()) + ");$s.TargetPath=" + psQuote(executable) + ";$s.Arguments='daemon --role client';$s.WindowStyle=7;$s.Save()")
}
func startWindowsService() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	return windowsScript("$p=Start-Process -FilePath " + psQuote(executable) + " -ArgumentList @('daemon','--role','client') -WindowStyle Hidden -PassThru;[IO.File]::WriteAllText(" + psQuote(filepath.Join(sysutil.DataDir(), "client.pid")) + ",[string]$p.Id)")
}
func stopWindowsService() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	// A stale PID must never terminate a different executable or a CLI configuration command.
	return windowsScript("$f=" + psQuote(filepath.Join(sysutil.DataDir(), "client.pid")) + ";if(Test-Path $f){$n=[int]([IO.File]::ReadAllText($f));$p=Get-CimInstance Win32_Process -Filter ('ProcessId='+$n);if($p -and $p.ExecutablePath -eq " + psQuote(executable) + " -and $p.CommandLine -match 'daemon --role client'){Stop-Process -Id $n};Remove-Item -LiteralPath $f}")
}
