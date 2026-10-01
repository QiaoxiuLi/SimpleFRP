package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/crypto"
	"github.com/simplefrp/simplefrp/internal/sysutil"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func BootstrapServer(publicIP string) (config.ServerConfig, error) {
	if _, err := os.Stat(sysutil.ServerConfigPath()); err == nil {
		return config.LoadServer()
	} else if !os.IsNotExist(err) {
		return config.ServerConfig{}, err
	}
	if publicIP == "" {
		client := &http.Client{Timeout: 5 * time.Second}
		for _, endpoint := range []string{"https://checkip.amazonaws.com", "https://api.ipify.org"} {
			response, err := client.Get(endpoint)
			if err != nil {
				continue
			}
			b, readErr := io.ReadAll(io.LimitReader(response.Body, 128))
			response.Body.Close()
			candidate := strings.TrimSpace(string(b))
			ip := net.ParseIP(candidate)
			if readErr == nil && response.StatusCode == 200 && ip != nil && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsUnspecified() && !ip.IsMulticast() {
				publicIP = candidate
				break
			}
		}
	}
	ip := net.ParseIP(publicIP)
	if ip == nil || ip.IsUnspecified() || ip.IsMulticast() {
		return config.ServerConfig{}, fmt.Errorf("public IP discovery failed; retry with simplefrp run --public-ip YOUR_SERVER_IP")
	}
	cert, key, pin, err := crypto.NewCertificate()
	if err != nil {
		return config.ServerConfig{}, err
	}
	id := make([]byte, 8)
	if _, err = rand.Read(id); err != nil {
		return config.ServerConfig{}, err
	}
	invite, err := crypto.RandomToken(16)
	if err != nil {
		return config.ServerConfig{}, err
	}
	admin, err := crypto.RandomToken(32)
	if err != nil {
		return config.ServerConfig{}, err
	}
	ports := []int{}
	for len(ports) < 3 {
		p, err := sysutil.RandomAvailablePort()
		if err != nil {
			return config.ServerConfig{}, err
		}
		exists := false
		for _, used := range ports {
			if used == p {
				exists = true
			}
		}
		if !exists {
			ports = append(ports, p)
		}
	}
	bind := "0.0.0.0"
	if ip.To4() == nil {
		bind = "::"
	}
	cfg := config.ServerConfig{Version: 2, ServerID: hex.EncodeToString(id), PublicIP: publicIP, BindAddress: bind, ControlPort: ports[0], DashboardBindAddress: "127.0.0.1", DashboardPort: ports[1], FirstPublicPort: ports[2], InviteKey: invite, AdminKey: admin, Certificate: cert, PrivateKey: key, Fingerprint: pin}
	if err = sysutil.EnsureBaseDirs(); err != nil {
		return cfg, err
	}
	if err = config.SaveServer(cfg); err != nil {
		return cfg, err
	}
	if err = WriteRole(RoleServer); err != nil {
		return cfg, err
	}
	return cfg, nil
}
func ConnectionCode(cfg config.ServerConfig) (string, error) {
	return crypto.EncodeInvite(crypto.Invite{IP: cfg.PublicIP, Port: cfg.ControlPort, ServerID: cfg.ServerID, Key: cfg.InviteKey, Fingerprint: cfg.Fingerprint})
}
