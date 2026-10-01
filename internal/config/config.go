package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/simplefrp/simplefrp/internal/sysutil"
)

type Tunnel struct {
	ID         int    `mapstructure:"id" json:"id" toml:"id"`
	LocalPort  int    `mapstructure:"local_port" json:"local_port" toml:"local_port"`
	PublicPort int    `mapstructure:"public_port" json:"public_port" toml:"public_port"`
	Status     string `mapstructure:"status" json:"status" toml:"status"`
}

func writeTOML(path string, value any) error {
	b, err := toml.Marshal(value)
	if err != nil {
		return err
	}
	return writeSecure(path, b)
}
func WriteJSON(path string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return writeSecure(path, b)
}

func load(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return toml.Unmarshal(b, out)
}

func writeSecure(path string, content []byte) error {
	if err := os.MkdirAll(sysutil.ConfigDir(), 0750); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".simplefrp-*.toml")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(content); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	_ = sysutil.ChownToServiceUser(tmp)
	return os.Rename(tmp, path)
}
