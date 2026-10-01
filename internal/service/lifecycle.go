package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/sysutil"
)

func installedBinary() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "SimpleFRP", "simplefrp.exe")
	case "darwin":
		return filepath.Join(home, ".local", "bin", "simplefrp")
	default:
		return "/usr/bin/simplefrp"
	}
}
func OwnsInstalledBinary(binary string) bool {
	if home := os.Getenv("SIMPLEFRP_HOME"); home != "" {
		rel, err := filepath.Rel(home, binary)
		return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
	}
	return samePath(binary, installedBinary())
}
func samePath(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
	}
	return filepath.Clean(a) == filepath.Clean(b)
}
func PrepareAccount(r *sysutil.Receipt) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	if _, err := user.LookupGroup("simplefrp"); err != nil {
		if err = run("groupadd", "--system", "simplefrp"); err != nil {
			return err
		}
		r.OwnServiceGroup = true
	}
	u, err := user.Lookup("simplefrp")
	if err != nil {
		if err = run("useradd", "--system", "--gid", "simplefrp", "--home-dir", "/var/lib/simplefrp", "--shell", "/usr/sbin/nologin", "simplefrp"); err != nil {
			return err
		}
		r.OwnServiceUser = true
		u, err = user.Lookup("simplefrp")
	}
	if err != nil {
		return err
	}
	if r.OwnServiceUser {
		r.ServiceUID = u.Uid
	}
	for _, dir := range []string{sysutil.ConfigDir(), sysutil.DataDir(), sysutil.LogDir()} {
		if err = sysutil.ChownToServiceUser(dir); err != nil {
			return err
		}
	}
	if err = sysutil.ChownToServiceUser(sysutil.RolePath()); err != nil {
		return err
	}
	return nil
}
func RemoveAccount(r sysutil.Receipt) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	if r.OwnServiceUser {
		u, err := user.Lookup("simplefrp")
		if err == nil {
			if u.Uid != r.ServiceUID {
				return fmt.Errorf("service account identity changed; it was not removed")
			}
			if err = run("userdel", "simplefrp"); err != nil {
				return err
			}
		} else if _, ok := err.(user.UnknownUserError); !ok {
			return err
		}
	}
	if r.OwnServiceGroup {
		if _, err := user.LookupGroup("simplefrp"); err == nil {
			return run("groupdel", "simplefrp")
		}
	}
	return nil
}

const pathBlock = "# BEGIN SimpleFRP managed PATH\nexport PATH=\"$HOME/.local/bin:$PATH\"\n# END SimpleFRP managed PATH\n"

func pathContains(value, entry string, separator string) bool {
	for _, p := range strings.Split(value, separator) {
		if samePath(strings.TrimSpace(p), entry) {
			return true
		}
	}
	return false
}
func removePathEntry(value, entry string) string {
	parts := strings.Split(value, ";")
	for i := len(parts) - 1; i >= 0; i-- {
		if samePath(strings.TrimSpace(parts[i]), entry) {
			return strings.Join(append(parts[:i], parts[i+1:]...), ";")
		}
	}
	return value
}
func windowsUserPath() (string, error) {
	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", windowsEncoded("$j=[Environment]::GetEnvironmentVariable('Path','User') | ConvertTo-Json -Compress;[Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($j))")).Output()
	if err != nil {
		return "", err
	}
	var value string
	out, err = base64.StdEncoding.DecodeString(strings.TrimSpace(string(out)))
	if err != nil {
		return "", err
	}
	if len(bytes.TrimSpace(out)) == 0 || string(bytes.TrimSpace(out)) == "null" {
		return "", nil
	}
	err = json.Unmarshal(out, &value)
	return value, err
}
func writeWindowsPath(value string) error {
	return windowsScript("[Environment]::SetEnvironmentVariable('Path'," + psQuote(value) + ",'User')")
}
func ConfigureCommandPath() error {
	r, err := sysutil.LoadReceipt()
	if err != nil {
		return err
	}
	if r.PathAdded || runtime.GOOS == "linux" {
		return nil
	}
	directory := filepath.Dir(installedBinary())
	if runtime.GOOS == "windows" {
		value, err := windowsUserPath()
		if err != nil {
			return err
		}
		if pathContains(value, directory, ";") {
			return nil
		}
		next := directory
		if value != "" {
			next = value + ";" + directory
		}
		if err = writeWindowsPath(next); err != nil {
			return err
		}
		r.PathAdded = true
		if err = config.WriteJSON(sysutil.ReceiptPath(), r); err != nil {
			_ = writeWindowsPath(value)
			return err
		}
		return nil
	}
	if pathContains(os.Getenv("PATH"), directory, ":") {
		return nil
	}
	home, _ := os.UserHomeDir()
	name := ".zprofile"
	if filepath.Base(os.Getenv("SHELL")) == "bash" {
		name = ".bash_profile"
	}
	path := filepath.Join(home, name)
	content, err := os.ReadFile(path)
	created := os.IsNotExist(err)
	if err != nil && !created {
		return err
	}
	if !bytes.Contains(content, []byte(pathBlock)) {
		if len(content) > 0 && content[len(content)-1] != '\n' {
			content = append(content, '\n')
		}
		content = append(content, []byte(pathBlock)...)
		if err = os.WriteFile(path, content, 0600); err != nil {
			return err
		}
	}
	r.PathAdded = true
	r.PathFile = path
	r.PathFileCreated = created
	return config.WriteJSON(sysutil.ReceiptPath(), r)
}
func RemoveCommandPath(r sysutil.Receipt) error {
	if !r.PathAdded {
		return nil
	}
	if runtime.GOOS == "windows" {
		value, err := windowsUserPath()
		if err != nil {
			return err
		}
		return writeWindowsPath(removePathEntry(value, filepath.Dir(installedBinary())))
	}
	if runtime.GOOS != "darwin" {
		return nil
	}
	home, _ := os.UserHomeDir()
	if r.PathFile != filepath.Join(home, ".zprofile") && r.PathFile != filepath.Join(home, ".bash_profile") {
		return fmt.Errorf("unverified shell profile path was not changed")
	}
	content, err := os.ReadFile(r.PathFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !bytes.Contains(content, []byte(pathBlock)) {
		if bytes.Contains(content, []byte("# BEGIN SimpleFRP managed PATH")) {
			return fmt.Errorf("SimpleFRP PATH block was edited; profile was not changed")
		}
		return nil
	}
	content = bytes.Replace(content, []byte(pathBlock), nil, 1)
	if r.PathFileCreated && len(content) == 0 {
		return os.Remove(r.PathFile)
	}
	return os.WriteFile(r.PathFile, content, 0600)
}
