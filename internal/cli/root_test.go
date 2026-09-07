package cli

import (
	"github.com/simplefrp/simplefrp/internal/config"
	"net"
	"testing"
)

func TestLocalTargetMayAlreadyBeListening(t *testing.T) {
	t.Setenv("SIMPLEFRP_CONFIG_DIR", t.TempDir())
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	cfg := config.DefaultClientConfig()
	cfg.Tunnels = []config.Tunnel{{ID: 0, LocalPort: 25000, PublicPort: 25565}}
	if err = config.SaveClient(cfg); err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err = updateLocalPort(0, port); err != nil {
		t.Fatal(err)
	}
	saved, err := config.LoadClient()
	if err != nil {
		t.Fatal(err)
	}
	if saved.Tunnels[0].LocalPort != port {
		t.Fatal("target was not persisted")
	}
	if err = updateLocalPort(0, 3389); err != nil {
		t.Fatal("reserved service ports are valid local targets:", err)
	}
	if err = updateLocalPort(0, 0); err == nil {
		t.Fatal("invalid target accepted")
	}
}
