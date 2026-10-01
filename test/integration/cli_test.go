package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/protocol"
)

type installation struct {
	binary, home string
	t            *testing.T
}

func (i installation) execute(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	commandArgs := args
	if i.home != "" {
		commandArgs = append([]string{"--home", i.home}, args...)
	}
	cmd := exec.CommandContext(ctx, i.binary, commandArgs...)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "SIMPLEFRP_") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	if i.home != "" {
		cmd.Env = append(cmd.Env, "SIMPLEFRP_TEST_SERVER=1")
	}
	return cmd.CombinedOutput()
}
func (i installation) must(args ...string) []byte {
	i.t.Helper()
	out, err := i.execute(args...)
	if err != nil {
		if args[0] == "uninstall" || args[0] == "install" {
			i.t.Fatalf("isolated %s command failed: %v: %s", args[0], err, out)
		}
		i.t.Fatalf("isolated %s command failed: %v", args[0], err)
	}
	return out
}
func (i installation) state() protocol.Status {
	i.t.Helper()
	var status protocol.Status
	if err := json.Unmarshal(i.must("state", "--json"), &status); err != nil {
		i.t.Fatal(err)
	}
	return status
}
func (i installation) waitOnline() protocol.Status {
	i.t.Helper()
	deadline := time.Now().Add(25 * time.Second)
	for time.Now().Before(deadline) {
		status := i.state()
		if status.Online && len(status.Tunnels) > 0 && status.Tunnels[0].Status == "connected" {
			return status
		}
		time.Sleep(200 * time.Millisecond)
	}
	i.t.Fatal("isolated client did not connect")
	return protocol.Status{}
}
func availablePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}
func forward(t *testing.T, port int) {
	forwardHost(t, "127.0.0.1", port)
}
func forwardHost(t *testing.T, host string, port int) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, fmt.Sprint(port)), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	message := bytes.Repeat([]byte("SimpleFRP-isolated-acceptance\n"), 2000)
	if _, err = conn.Write(message); err != nil {
		t.Fatal(err)
	}
	response := make([]byte, len(message))
	if _, err = io.ReadFull(conn, response); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(message, response) {
		t.Fatal("forwarded application bytes changed")
	}
}

func TestIsolatedCLIWorkflow(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "simplefrp")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if supplied := os.Getenv("SIMPLEFRP_TEST_BINARY"); supplied != "" {
		var err error
		binary, err = filepath.Abs(supplied)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = os.Stat(binary); err != nil {
			t.Fatal("prebuilt test executable unavailable")
		}
	} else {
		goTool := filepath.Join(runtime.GOROOT(), "bin", "go")
		if runtime.GOOS == "windows" {
			goTool += ".exe"
		}
		build := exec.Command(goTool, "build", "-o", binary, "./cmd/simplefrp")
		build.Dir = filepath.Join("..", "..")
		if out, err := build.CombinedOutput(); err != nil {
			t.Fatalf("build failed: %v\n%s", err, out)
		}
	}
	server := installation{binary, filepath.Join(root, "server"), t}
	a := installation{binary, filepath.Join(root, "client-a"), t}
	b := installation{binary, filepath.Join(root, "client-b"), t}
	for _, i := range []installation{server, a, b} {
		role := "client"
		if i.home == server.home {
			role = "server"
		}
		i.must("install", "--role", role, "--no-start")
		owned := i
		t.Cleanup(func() {
			if _, err := os.Stat(filepath.Join(owned.home, "config", "installation.json")); err == nil {
				if out, cleanupErr := owned.execute("uninstall"); cleanupErr != nil {
					t.Errorf("isolated cleanup failed: %v: %s", cleanupErr, out)
				}
			}
		})
	}
	runOutput := server.must("run", "--public-ip", "127.0.0.1")
	var invitation string
	for _, word := range strings.Fields(string(runOutput)) {
		if strings.HasPrefix(word, "SF2") {
			invitation = word
		}
	}
	if invitation == "" {
		t.Fatal("server did not output a connection string")
	}
	var serverConfig config.ServerConfig
	contents, err := os.ReadFile(filepath.Join(server.home, "config", "server.toml"))
	if err != nil || toml.Unmarshal(contents, &serverConfig) != nil {
		t.Fatal("server configuration could not be read")
	}
	if serverConfig.ControlPort == serverConfig.DashboardPort || serverConfig.FirstPublicPort == serverConfig.ControlPort || serverConfig.FirstPublicPort == serverConfig.DashboardPort {
		t.Fatal("initial ports are not distinct")
	}
	// Repeating initialization must not change server identity or its ports.
	server.must("run")
	unchanged, _ := os.ReadFile(filepath.Join(server.home, "config", "server.toml"))
	if !bytes.Equal(contents, unchanged) {
		t.Fatal("repeated run modified initialized configuration")
	}

	target, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	go func() {
		for {
			conn, err := target.Accept()
			if err != nil {
				return
			}
			go func() { defer conn.Close(); io.Copy(conn, conn) }()
		}
	}()
	local := target.Addr().(*net.TCPAddr).Port
	a.must("set", invitation)
	first := a.waitOnline().Tunnels[0]
	a.must(fmt.Sprint(first.ID), "local", fmt.Sprint(local))
	forward(t, first.PublicPort)
	if a.state().Tunnels[0].TotalBytes == 0 {
		t.Fatal("forwarded traffic was not counted")
	}
	a.must("set", invitation)
	if len(a.state().Tunnels) != 1 {
		t.Fatal("repeated set created another tunnel")
	}
	b.must("set", invitation)
	second := b.waitOnline().Tunnels[0]
	if first.ID == second.ID {
		t.Fatal("tunnel IDs collide across clients")
	}
	if _, err = b.execute(fmt.Sprint(first.ID), "local", fmt.Sprint(local)); err == nil {
		t.Fatal("client modified another client's tunnel")
	}
	a.must("next")
	if len(a.state().Tunnels) != 2 {
		t.Fatal("next did not create exactly one new tunnel")
	}
	public := availablePort(t)
	server.must(fmt.Sprint(first.ID), "public", fmt.Sprint(public))
	forward(t, public)
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	if _, err = a.execute(fmt.Sprint(first.ID), "public", fmt.Sprint(busy.Addr().(*net.TCPAddr).Port)); err == nil {
		t.Fatal("occupied public port was accepted")
	}
	forward(t, public)
	webPort := availablePort(t)
	server.must("port", "dashboard", fmt.Sprint(webPort))
	response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/status", webPort))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal("status website not available")
	}
	response, err = http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/status", webPort), "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatal("status website accepted a write request")
	}
	heartbeat := availablePort(t)
	server.must("port", "heartbeat", fmt.Sprint(heartbeat))
	a.waitOnline()
	b.waitOnline()
	if a.state().HeartbeatPort != heartbeat || len(a.state().Tunnels) != 2 {
		t.Fatal("heartbeat change did not preserve tunnels")
	}
	forward(t, public)
	if len(server.state().Tunnels) != 3 {
		t.Fatal("changing service ports created a tunnel")
	}
	beforeRestart := a.state().Tunnels[0].TotalBytes
	if runtime.GOOS == "windows" {
		// This test-only Windows server is force-stopped; production servers are Linux.
		// Let the periodic durable sample finish before simulating that abrupt exit.
		time.Sleep(1100 * time.Millisecond)
	}
	server.must("install", "--role", "server", "--prepare-upgrade")
	server.must("run")
	a.waitOnline()
	b.waitOnline()
	forward(t, public)
	if len(server.state().Tunnels) != 3 || a.state().Tunnels[0].TotalBytes < beforeRestart {
		t.Fatal("restart lost mappings or counters")
	}
	// Uninstalling one isolated client must not stop another client's forwarding.
	a.must("uninstall")
	if len(server.state().Tunnels) != 1 {
		t.Fatal("client uninstall left remote tunnel records")
	}
	b.must(fmt.Sprint(second.ID), "local", fmt.Sprint(local))
	forward(t, second.PublicPort)
	for _, dir := range []string{"config", "data", "logs"} {
		if _, err = os.Stat(filepath.Join(a.home, dir)); !os.IsNotExist(err) {
			t.Fatal("client uninstall left runtime files")
		}
	}
	if _, err = os.Stat(binary); err != nil {
		t.Fatal("uninstall removed the shared test executable")
	}
	b.must("uninstall")
	server.must("uninstall")
	for _, i := range []installation{b, server} {
		for _, dir := range []string{"config", "data", "logs"} {
			if _, err = os.Stat(filepath.Join(i.home, dir)); !os.IsNotExist(err) {
				t.Fatal("uninstall left runtime files")
			}
		}
	}
	for _, port := range []int{heartbeat, webPort, public, second.PublicPort} {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
		if err == nil {
			conn.Close()
			t.Fatalf("uninstall left an owned port listening: %d", port)
		}
	}
}
