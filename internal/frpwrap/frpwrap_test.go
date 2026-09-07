package frpwrap

import (
	"bufio"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/simplefrp/simplefrp/internal/config"
)

func TestReverseTCPForwarding(t *testing.T) {
	localPort := freePort(t)
	publicPort := freePort(t)
	controlPort := freePort(t)
	authKey := "test-auth-key"

	stopEcho := startEchoServer(t, localPort)
	defer stopEcho()

	server := NewServerEngine(config.ServerConfig{
		BindAddress:   "127.0.0.1",
		ControlPort:   controlPort,
		DashboardPort: freePort(t),
		AuthKey:       authKey,
		PortMin:       publicPort,
		PortMax:       publicPort,
	})
	defer server.Close()
	go func() {
		_ = server.ListenAndServe()
	}()
	waitForPort(t, controlPort)

	client := ClientEngine{
		ServerAddress: fmt.Sprintf("127.0.0.1:%d", controlPort),
		ClientID:      "client-1",
		AuthKey:       authKey,
	}
	go func() {
		_ = client.runOnce([]config.Tunnel{{ID: 0, LocalPort: localPort, PublicPort: publicPort, Status: "success"}})
	}()
	waitForPort(t, publicPort)

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", publicPort), 3*time.Second)
	if err != nil {
		t.Fatalf("dial public tunnel: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))

	if _, err := conn.Write([]byte("hello\n")); err != nil {
		t.Fatalf("write public tunnel: %v", err)
	}
	got, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatalf("read public tunnel: %v", err)
	}
	if got != "echo:hello\n" {
		t.Fatalf("unexpected echo response: %q", got)
	}
}

func TestCreateTunnelRejectsWrongAuth(t *testing.T) {
	controlPort := freePort(t)
	server := NewServerEngine(config.ServerConfig{
		BindAddress: "127.0.0.1",
		ControlPort: controlPort,
		AuthKey:     "correct-key",
		PortMin:     freePort(t),
		PortMax:     freePort(t),
	})
	defer server.Close()
	go func() {
		_ = server.ListenAndServe()
	}()
	waitForPort(t, controlPort)

	client := ClientEngine{
		ServerAddress: fmt.Sprintf("127.0.0.1:%d", controlPort),
		ClientID:      "client-1",
		AuthKey:       "wrong-key",
	}
	resp, err := client.CreateTunnel(1, freePort(t), 0)
	if err != nil {
		t.Fatalf("create tunnel with wrong auth returned transport error: %v", err)
	}
	if resp.OK || resp.Code != "AUTH_FAILED" {
		t.Fatalf("expected AUTH_FAILED, got %+v", resp)
	}
}

func startEchoServer(t *testing.T, port int) func() {
	t.Helper()
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("start echo server: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				reader := bufio.NewReader(c)
				line, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				_, _ = c.Write([]byte("echo:" + line))
			}(conn)
		}
	}()
	return func() {
		_ = ln.Close()
		<-done
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func waitForPort(t *testing.T, port int) {
	t.Helper()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("port did not open: %s", addr)
}
