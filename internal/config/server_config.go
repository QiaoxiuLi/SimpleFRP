package config

import (
	"fmt"

	"github.com/simplefrp/simplefrp/internal/crypto"
	"github.com/simplefrp/simplefrp/internal/sysutil"
)

type ServerConfig struct {
	BindAddress          string `mapstructure:"bind_address"`
	ControlPort          int    `mapstructure:"control_port"`
	DashboardBindAddress string `mapstructure:"dashboard_bind_address"`
	DashboardPort        int    `mapstructure:"dashboard_port"`
	PasswordHash         string `mapstructure:"password_hash"`
	AuthKey              string `mapstructure:"auth_key"`
	DashboardToken       string `mapstructure:"dashboard_token"`
	PortMin              int    `mapstructure:"port_min"`
	PortMax              int    `mapstructure:"port_max"`
}

func NewServerConfig(password string) (ServerConfig, error) {
	if err := ValidatePassword(password); err != nil {
		return ServerConfig{}, err
	}
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return ServerConfig{}, err
	}
	token, err := crypto.RandomToken(32)
	if err != nil {
		return ServerConfig{}, err
	}
	authKey := crypto.RandomPasswordKey(password, SharedAuthSalt)
	return ServerConfig{
		BindAddress:          "0.0.0.0",
		ControlPort:          8388,
		DashboardPort:        8387,
		DashboardBindAddress: "127.0.0.1",
		PasswordHash:         hash,
		AuthKey:              authKey,
		DashboardToken:       token,
		PortMin:              20000,
		PortMax:              60000,
	}, nil
}

func LoadServer() (ServerConfig, error) {
	var cfg ServerConfig
	err := load(sysutil.ServerConfigPath(), &cfg)
	return cfg, err
}

func SaveServer(cfg ServerConfig) error {
	content := []byte(fmt.Sprintf(`bind_address = %q
control_port = %d
dashboard_port = %d
dashboard_bind_address = %q
password_hash = %q
auth_key = %q
dashboard_token = %q
port_min = %d
port_max = %d
`, cfg.BindAddress, cfg.ControlPort, cfg.DashboardPort, cfg.DashboardBindAddress, cfg.PasswordHash, cfg.AuthKey, cfg.DashboardToken, cfg.PortMin, cfg.PortMax))
	return writeSecure(sysutil.ServerConfigPath(), content)
}
