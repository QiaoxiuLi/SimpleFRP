package config

import (
	"fmt"
	"strings"

	"github.com/simplefrp/simplefrp/internal/crypto"
	"github.com/simplefrp/simplefrp/internal/sysutil"
)

type ClientConfig struct {
	ServerAddress string   `mapstructure:"server_address"`
	PasswordKey   string   `mapstructure:"password_key"`
	ClientID      string   `mapstructure:"client_id"`
	Tunnels       []Tunnel `mapstructure:"tunnels"`
}

const SharedAuthSalt = "simplefrp-control-v1"

func DefaultClientConfig() ClientConfig {
	id, _ := crypto.RandomToken(12)
	return ClientConfig{ClientID: id, Tunnels: []Tunnel{}}
}

func LoadClient() (ClientConfig, error) {
	var cfg ClientConfig
	err := load(sysutil.ClientConfigPath(), &cfg)
	return cfg, err
}

func SaveClient(cfg ClientConfig) error {
	var b strings.Builder
	fmt.Fprintf(&b, "server_address = %q\n", cfg.ServerAddress)
	fmt.Fprintf(&b, "password_key = %q\n", cfg.PasswordKey)
	fmt.Fprintf(&b, "client_id = %q\n\n", cfg.ClientID)
	for _, t := range cfg.Tunnels {
		fmt.Fprintf(&b, "[[tunnels]]\nid = %d\nlocal_port = %d\npublic_port = %d\nstatus = %q\n\n", t.ID, t.LocalPort, t.PublicPort, t.Status)
	}
	return writeSecure(sysutil.ClientConfigPath(), []byte(b.String()))
}

func NextTunnelID(tunnels []Tunnel) int {
	max := -1
	for _, t := range tunnels {
		if t.ID > max {
			max = t.ID
		}
	}
	return max + 1
}

func FindTunnel(tunnels []Tunnel, id int) (Tunnel, int, bool) {
	for i, t := range tunnels {
		if t.ID == id {
			return t, i, true
		}
	}
	return Tunnel{}, -1, false
}
