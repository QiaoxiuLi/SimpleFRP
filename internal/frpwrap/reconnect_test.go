package frpwrap

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/protocol"
	"io"
	"net"
	"testing"
	"time"
)

func echoThrough(t *testing.T, port int) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	_, _ = conn.Write([]byte("test\n"))
	got, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil || got != "echo:test\n" {
		t.Fatalf("forward: %q %v", got, err)
	}
}
func TestReconnectRebindsListenerAndUpdatedTarget(t *testing.T) {
	control, public, first, second := freePort(t), freePort(t), freePort(t), freePort(t)
	stop := startEchoServer(t, first)
	defer stop()
	stop2 := startEchoServer(t, second)
	defer stop2()
	server := NewServerEngine(config.ServerConfig{BindAddress: "127.0.0.1", ControlPort: control, AuthKey: "test"})
	defer server.Close()
	go server.ListenAndServe()
	waitForPort(t, control)
	client := ClientEngine{ServerAddress: fmt.Sprintf("127.0.0.1:%d", control), ClientID: "same-device", AuthKey: "test"}
	var previous *clientSession
	for _, local := range []int{first, second, first} {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() {
			done <- client.runOnceContext(ctx, []config.Tunnel{{ID: 0, LocalPort: local, PublicPort: public}})
		}()
		deadline := time.Now().Add(3 * time.Second)
		for {
			server.mu.Lock()
			current := server.clients[client.ClientID]
			binding := server.bindings[public]
			server.mu.Unlock()
			if current != nil && current != previous && binding.record.LocalPort == local {
				previous = current
				break
			}
			if time.Now().After(deadline) {
				cancel()
				t.Fatal("client did not register")
			}
			time.Sleep(10 * time.Millisecond)
		}
		echoThrough(t, public)
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("cancel did not close control connection")
		}
	}
}
func TestDataHandshakePreservesCoalescedPayload(t *testing.T) {
	server := NewServerEngine(config.ServerConfig{AuthKey: "test"})
	ch := make(chan net.Conn, 1)
	server.pending["request"] = &pendingStream{clientID: "device", tunnelID: 2, channel: ch}
	serverSide, clientSide := net.Pipe()
	defer clientSide.Close()
	go server.handle(serverSide)
	msg := signMessage("test", protocol.Message{Type: protocol.TypeDataConn, ClientID: "device", TunnelID: 2, RequestID: "request"})
	encoded, _ := json.Marshal(msg)
	payload := append(append(encoded, '\n'), []byte("server-first-banner")...)
	go clientSide.Write(payload)
	select {
	case data := <-ch:
		defer data.Close()
		_ = data.SetReadDeadline(time.Now().Add(time.Second))
		got := make([]byte, len("server-first-banner"))
		if _, err := io.ReadFull(data, got); err != nil || string(got) != "server-first-banner" {
			t.Fatalf("buffer lost: %q %v", got, err)
		}
	case <-time.After(time.Second):
		t.Fatal("handshake failed")
	}
}
func TestAuthProofBindsTunnelTargets(t *testing.T) {
	msg := signMessage("test", protocol.Message{Type: protocol.TypeAuth, ClientID: "device", Tunnels: []protocol.TunnelStatus{{TunnelID: 0, LocalPort: 3000, PublicPort: 9290}}})
	if !verifyMessage("test", msg) {
		t.Fatal("valid signature rejected")
	}
	msg.Tunnels[0].LocalPort = 22
	if verifyMessage("test", msg) {
		t.Fatal("modified target list accepted")
	}
}
func TestTCPHalfClosePreservesResponse(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	target, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	go func() {
		c, _ := target.Accept()
		defer c.Close()
		_, _ = io.ReadAll(c)
		_, _ = c.Write([]byte("response-after-fin"))
	}()
	done := make(chan struct{})
	go func() {
		defer close(done)
		a, _ := listener.Accept()
		b, _ := net.Dial("tcp", target.Addr().String())
		proxy(a, b)
	}()
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	_, _ = conn.Write([]byte("request"))
	_ = conn.(*net.TCPConn).CloseWrite()
	body, err := io.ReadAll(conn)
	if err != nil || string(body) != "response-after-fin" {
		t.Fatalf("response truncated: %q %v", body, err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("bridge leaked")
	}
}
