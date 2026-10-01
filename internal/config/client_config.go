package config

import (
	"fmt"
	"github.com/simplefrp/simplefrp/internal/sysutil"
)

type ClientConfig struct {
	Version       int      `mapstructure:"version" toml:"version"`
	ServerID      string   `mapstructure:"server_id" toml:"server_id"`
	ServerAddress string   `mapstructure:"server_address" toml:"server_address"`
	Fingerprint   string   `mapstructure:"fingerprint" toml:"fingerprint"`
	ClientID      string   `mapstructure:"client_id" toml:"client_id"`
	ClientKey     string   `mapstructure:"client_key" toml:"client_key"`
	Tunnels       []Tunnel `mapstructure:"tunnels" toml:"tunnels"`
}

func DefaultClientConfig() ClientConfig { return ClientConfig{Version: 2, Tunnels: []Tunnel{}} }
func LoadClient() (ClientConfig, error) {
	var cfg ClientConfig
	if err := load(sysutil.ClientConfigPath(), &cfg); err != nil {
		return cfg, err
	}
	if cfg.Version != 2 {
		return cfg, fmt.Errorf("legacy client configuration detected; it has not been changed; use a separate v2 installation")
	}
	return cfg, nil
}
func SaveClient(cfg ClientConfig) error { return writeTOML(sysutil.ClientConfigPath(), cfg) }
func FindTunnel(tunnels []Tunnel, id int) (Tunnel, int, bool) {
	for i, t := range tunnels {
		if t.ID == id {
			return t, i, true
		}
	}
	return Tunnel{}, -1, false
}
