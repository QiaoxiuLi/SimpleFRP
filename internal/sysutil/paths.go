package sysutil

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
)

func ConfigDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("ProgramData"), "SimpleFRP")
	case "darwin":
		return "/Library/Application Support/SimpleFRP"
	default:
		return "/etc/simplefrp"
	}
}

func DataDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(ConfigDir(), "data")
	case "darwin":
		return filepath.Join(ConfigDir(), "data")
	default:
		return "/var/lib/simplefrp"
	}
}

func LogDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(ConfigDir(), "logs")
	case "darwin":
		return "/Library/Logs/SimpleFRP"
	default:
		return "/var/log/simplefrp"
	}
}

func RolePath() string         { return filepath.Join(ConfigDir(), "role") }
func ServerConfigPath() string { return filepath.Join(ConfigDir(), "server.toml") }
func ClientConfigPath() string { return filepath.Join(ConfigDir(), "client.toml") }
func DatabasePath() string     { return filepath.Join(DataDir(), "simplefrp.db") }

func EnsureBaseDirs() error {
	for _, dir := range []string{ConfigDir(), DataDir(), LogDir()} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return err
		}
		_ = ChownToServiceUser(dir)
	}
	return nil
}

func FileWritable(path string) bool {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

func ChownToServiceUser(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	u, err := user.Lookup("simplefrp")
	if err != nil {
		return nil
	}
	g, err := user.LookupGroup("simplefrp")
	if err != nil {
		return nil
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return nil
	}
	gid, err := strconv.Atoi(g.Gid)
	if err != nil {
		return nil
	}
	return os.Chown(path, uid, gid)
}
