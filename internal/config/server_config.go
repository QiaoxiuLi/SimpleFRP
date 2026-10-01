package config

import (
	"fmt"
	"github.com/simplefrp/simplefrp/internal/sysutil"
)

type ServerConfig struct {
	Version              int    `mapstructure:"version" toml:"version"`
	ServerID             string `mapstructure:"server_id" toml:"server_id"`
	PublicIP             string `mapstructure:"public_ip" toml:"public_ip"`
	BindAddress          string `mapstructure:"bind_address" toml:"bind_address"`
	ControlPort          int    `mapstructure:"control_port" toml:"control_port"`
	DashboardBindAddress string `mapstructure:"dashboard_bind_address" toml:"dashboard_bind_address"`
	DashboardPort        int    `mapstructure:"dashboard_port" toml:"dashboard_port"`
	FirstPublicPort      int    `mapstructure:"first_public_port" toml:"first_public_port"`
	InviteKey            string `mapstructure:"invite_key" toml:"invite_key"`
	AdminKey             string `mapstructure:"admin_key" toml:"admin_key"`
	Certificate          string `mapstructure:"certificate" toml:"certificate"`
	PrivateKey           string `mapstructure:"private_key" toml:"private_key"`
	Fingerprint          string `mapstructure:"fingerprint" toml:"fingerprint"`
}

func LoadServer() (ServerConfig, error) {
	var cfg ServerConfig
	if err := load(sysutil.ServerConfigPath(), &cfg); err != nil {
		return cfg, err
	}
	if cfg.Version != 2 {
		return cfg, fmt.Errorf("legacy server configuration detected; it has not been changed; use a separate v2 installation")
	}
	return cfg, nil
}
func SaveServer(cfg ServerConfig) error { return writeTOML(sysutil.ServerConfigPath(), cfg) }
