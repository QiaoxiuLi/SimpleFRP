package frpwrap

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/crypto"
	"github.com/simplefrp/simplefrp/internal/dashboard"
	"github.com/simplefrp/simplefrp/internal/protocol"
	"github.com/simplefrp/simplefrp/internal/storage"
	"io"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}
func fixture(t *testing.T) (*ServerEngine, config.ServerConfig, *storage.Store) {
	t.Helper()
	cert, key, pin, err := crypto.NewCertificate()
	if err != nil {
		t.Fatal(err)
	}
	invite, _ := crypto.RandomToken(16)
	admin, _ := crypto.RandomToken(32)
	cfg := config.ServerConfig{Version: 2, PublicIP: "127.0.0.1", ServerID: "0123456789abcdef", BindAddress: "127.0.0.1", ControlPort: freePort(t), DashboardPort: freePort(t), FirstPublicPort: freePort(t), InviteKey: invite, AdminKey: admin, Certificate: cert, PrivateKey: key, Fingerprint: pin}
	store, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	e := NewServerEngine(cfg, store)
	e.SaveConfig = func(config.ServerConfig) error { return nil }
	if err = e.Start(); err != nil {
		store.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { e.Close(); store.Close() })
	return e, cfg, store
}

type testClient struct {
	mu     sync.Mutex
	cfg    config.ClientConfig
	cancel context.CancelFunc
	done   chan error
}

func attach(t *testing.T, e *ServerEngine, cfg config.ServerConfig, local int) *testClient {
	t.Helper()
	paired, err := Pair(crypto.Invite{IP: cfg.PublicIP, Port: cfg.ControlPort, ServerID: cfg.ServerID, Key: cfg.InviteKey, Fingerprint: cfg.Fingerprint}, local)
	if err != nil {
		t.Fatal(err)
	}
	c := &testClient{cfg: paired, done: make(chan error, 1)}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	go func() { c.done <- RunClient(ctx, ClientCallbacks{Load: c.load, Save: c.save}) }()
	wait(t, func() bool { e.mu.Lock(); defer e.mu.Unlock(); return e.clients[paired.ClientID] != nil })
	t.Cleanup(func() {
		cancel()
		select {
		case <-c.done:
		case <-time.After(5 * time.Second):
			t.Error("client leaked")
		}
	})
	return c
}
func (c *testClient) load() (config.ClientConfig, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v := c.cfg
	v.Tunnels = append([]config.Tunnel(nil), v.Tunnels...)
	return v, nil
}
func (c *testClient) save(cfg config.ClientConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cfg = cfg
	return nil
}
func wait(t *testing.T, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("condition timed out")
}
func startEcho(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				c.SetDeadline(time.Now().Add(5 * time.Second))
				line, err := bufio.NewReader(c).ReadString('\n')
				if err == nil {
					c.Write([]byte("echo:" + line))
				}
			}()
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port
}
func echo(t *testing.T, port int) {
	t.Helper()
	c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(5 * time.Second))
	c.Write([]byte("test\n"))
	got, err := bufio.NewReader(c).ReadString('\n')
	if err != nil || got != "echo:test\n" {
		t.Fatalf("forwarding failed: %q %v", got, err)
	}
}
func TestV2ForwardingMutationMetricsAndClientIsolation(t *testing.T) {
	e, cfg, _ := fixture(t)
	first := attach(t, e, cfg, startEcho(t))
	second := attach(t, e, cfg, startEcho(t))
	a, _ := first.load()
	b, _ := second.load()
	if a.ClientID == b.ClientID || a.Tunnels[0].ID == b.Tunnels[0].ID {
		t.Fatal("identities or global IDs collide")
	}
	echo(t, a.Tunnels[0].PublicPort)
	admin := NewAdmin(cfg)
	status, err := admin.Request(protocol.Message{Type: protocol.TypeStatus})
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Status.Tunnels) != 2 || status.Status.Tunnels[0].TotalBytes == 0 {
		t.Fatal("live traffic was not counted")
	}
	client := NewClientEngine(b)
	if _, err = client.Request(protocol.Message{Type: protocol.TypeUpdate, TunnelID: a.Tunnels[0].ID, Kind: "local", Port: 22}); err == nil {
		t.Fatal("cross-client mutation accepted")
	}
	target := startEcho(t)
	resp, err := admin.Request(protocol.Message{Type: protocol.TypeUpdate, TunnelID: a.Tunnels[0].ID, Kind: "local", Port: target})
	if err != nil || resp.LocalPort != target {
		t.Fatal("server-side local update failed", err)
	}
	echo(t, a.Tunnels[0].PublicPort)
	a, _ = first.load()
	public := freePort(t)
	if _, err = NewClientEngine(a).Request(protocol.Message{Type: protocol.TypeUpdate, TunnelID: a.Tunnels[0].ID, Kind: "public", Port: public}); err != nil {
		t.Fatal(err)
	}
	echo(t, public)
	created, err := NewClientEngine(a).CreateTunnel(startEcho(t))
	if err != nil {
		t.Fatal(err)
	}
	echo(t, created.PublicPort)
	if _, err = admin.Request(protocol.Message{Type: protocol.TypeDeleteTunnel, TunnelID: created.TunnelID}); err != nil {
		t.Fatal(err)
	}
	if c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", created.PublicPort), 200*time.Millisecond); err == nil {
		c.Close()
		t.Fatal("deleted listener still active")
	}
}
func TestPortConflictsPreserveMappingAndTLSRejectsWrongPin(t *testing.T) {
	e, cfg, _ := fixture(t)
	client := attach(t, e, cfg, startEcho(t))
	a, _ := client.load()
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	if _, err = NewAdmin(cfg).Request(protocol.Message{Type: protocol.TypeUpdate, TunnelID: a.Tunnels[0].ID, Kind: "public", Port: busy.Addr().(*net.TCPAddr).Port}); err == nil {
		t.Fatal("busy public port accepted")
	}
	echo(t, a.Tunnels[0].PublicPort)
	wrong := NewAdmin(cfg)
	wrong.Fingerprint = fmt.Sprintf("%064x", 1)
	if _, err = wrong.Request(protocol.Message{Type: protocol.TypeStatus}); err == nil {
		t.Fatal("untrusted server accepted")
	}
}
func TestDashboardAndHeartbeatPortChanges(t *testing.T) {
	e, cfg, _ := fixture(t)
	client := attach(t, e, cfg, startEcho(t))
	before, _ := client.load()
	e.mu.Lock()
	nextID := e.nextID
	e.mu.Unlock()
	web := dashboard.New("127.0.0.1", func() protocol.Status { return e.Status("", true) })
	if err := web.Start(cfg.DashboardPort); err != nil {
		t.Fatal(err)
	}
	defer web.Close()
	e.RebindDashboard = web.Start
	admin := NewAdmin(cfg)
	next := freePort(t)
	if _, err := admin.Request(protocol.Message{Type: protocol.TypePort, Kind: "dashboard", Port: next}); err != nil {
		t.Fatal(err)
	}
	heartbeat := freePort(t)
	if _, err := admin.Request(protocol.Message{Type: protocol.TypePort, Kind: "heartbeat", Port: heartbeat}); err != nil {
		t.Fatal(err)
	}
	wait(t, func() bool { a, _ := client.load(); return a.ServerAddress == fmt.Sprintf("127.0.0.1:%d", heartbeat) })
	wait(t, func() bool {
		a, _ := client.load()
		e.mu.Lock()
		defer e.mu.Unlock()
		session := e.clients[a.ClientID]
		return session != nil && session.conn.LocalAddr().(*net.TCPAddr).Port == heartbeat
	})
	a, _ := client.load()
	if len(a.Tunnels) != len(before.Tunnels) || len(a.Tunnels) != 1 || a.Tunnels[0] != before.Tunnels[0] {
		t.Fatal("changing a service port altered or added a tunnel")
	}
	e.mu.Lock()
	unchanged := e.nextID == nextID && len(e.bindings) == 1 && len(e.listeners) == 1
	e.mu.Unlock()
	if !unchanged {
		t.Fatal("changing a service port allocated another tunnel or forwarding listener")
	}
	oldWeb, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", cfg.DashboardPort), time.Second)
	if err == nil {
		oldWeb.Close()
		t.Fatal("the old dashboard port is still listening")
	}
	echo(t, a.Tunnels[0].PublicPort)
	if _, err := NewClientEngine(a).Request(protocol.Message{Type: protocol.TypeStatus}); err != nil {
		t.Fatal(err)
	}
}
func TestAuthenticationTamperAndReplay(t *testing.T) {
	e, cfg, _ := fixture(t)
	msg := signMessage(cfg.AdminKey, protocol.Message{Type: protocol.TypePort, Admin: true, Kind: "heartbeat", Port: 27001})
	changed := msg
	changed.Port = 22
	if e.authorized(changed) {
		t.Fatal("tampered request accepted")
	}
	if !e.authorized(msg) {
		t.Fatal("valid request rejected")
	}
	if e.authorized(msg) {
		t.Fatal("replayed request accepted")
	}
}

func TestOversizedAuthenticatedControlFrameIsClosed(t *testing.T) {
	_, cfg, _ := fixture(t)
	paired, err := Pair(crypto.Invite{IP: cfg.PublicIP, Port: cfg.ControlPort, ServerID: cfg.ServerID, Key: cfg.InviteKey, Fingerprint: cfg.Fingerprint}, freePort(t))
	if err != nil {
		t.Fatal(err)
	}
	engine := NewClientEngine(paired)
	conn, err := engine.dial(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	if err := json.NewEncoder(conn).Encode(signMessage(engine.AuthKey, protocol.Message{Type: protocol.TypeAuth, ClientID: paired.ClientID})); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(conn)
	var response protocol.Message
	if err := decoder.Decode(&response); err != nil {
		t.Fatal(err)
	}
	io.WriteString(conn, `{"type":"heartbeat","padding":"`+strings.Repeat("x", 80*1024)+"\"}\n")
	if err := decoder.Decode(&response); err == nil {
		t.Fatal("oversized frame accepted")
	} else if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
		t.Fatal("oversized frame was not promptly closed")
	}
	if _, err := NewAdmin(cfg).Request(protocol.Message{Type: protocol.TypeStatus}); err != nil {
		t.Fatal("bad client disrupted the server")
	}
}
func TestTCPHalfClosePreservesResponse(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	target, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	go func() { c, _ := target.Accept(); defer c.Close(); io.ReadAll(c); c.Write([]byte("response-after-fin")) }()
	done := make(chan struct{})
	go func() {
		defer close(done)
		a, _ := ln.Accept()
		b, _ := net.Dial("tcp", target.Addr().String())
		proxy(a, b)
	}()
	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(3 * time.Second))
	c.Write([]byte("request"))
	c.(*net.TCPConn).CloseWrite()
	b, err := io.ReadAll(c)
	if err != nil || string(b) != "response-after-fin" {
		t.Fatal("half-close lost response", err)
	}
	<-done
}
func TestCoalescedDataHandshakePreservesPayload(t *testing.T) {
	e, cfg, _ := fixture(t)
	paired, err := Pair(crypto.Invite{IP: cfg.PublicIP, Port: cfg.ControlPort, ServerID: cfg.ServerID, Key: cfg.InviteKey, Fingerprint: cfg.Fingerprint}, 3000)
	if err != nil {
		t.Fatal(err)
	}
	ch := make(chan net.Conn, 1)
	e.mu.Lock()
	e.pending["fixture"] = &pendingStream{paired.ClientID, 0, ch}
	e.mu.Unlock()
	server, client := net.Pipe()
	defer client.Close()
	go e.handle(server)
	msg := signMessage(paired.ClientKey, protocol.Message{Type: protocol.TypeDataConn, ClientID: paired.ClientID, TunnelID: 0, RequestID: "fixture"})
	encoded, _ := json.Marshal(msg)
	go client.Write(append(append(encoded, '\n'), []byte("banner")...))
	select {
	case c := <-ch:
		defer c.Close()
		got := make([]byte, 6)
		if _, err = io.ReadFull(c, got); err != nil || string(got) != "banner" {
			t.Fatal("buffered bytes lost")
		}
	case <-time.After(time.Second):
		t.Fatal("data handshake failed")
	}
}
