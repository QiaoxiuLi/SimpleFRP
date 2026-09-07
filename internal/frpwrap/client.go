package frpwrap

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"reflect"
	"time"

	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/protocol"
)

type ClientEngine struct {
	ServerAddress string
	ClientID      string
	AuthKey       string
}

func NewClientEngine(cfg config.ClientConfig) ClientEngine {
	return ClientEngine{ServerAddress: cfg.ServerAddress, ClientID: cfg.ClientID, AuthKey: cfg.PasswordKey}
}

func (e ClientEngine) CreateTunnel(tunnelID, localPort, publicPort int) (protocol.Response, error) {
	conn, err := net.DialTimeout("tcp", e.ServerAddress, 5*time.Second)
	if err != nil {
		return protocol.Response{}, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	req := signMessage(e.AuthKey, protocol.Message{
		Type:                protocol.TypeCreateTunnel,
		ClientID:            e.ClientID,
		TunnelID:            tunnelID,
		LocalPort:           localPort,
		RequestedPublicPort: publicPort,
	})
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return protocol.Response{}, err
	}
	var resp protocol.Response
	err = json.NewDecoder(conn).Decode(&resp)
	return resp, err
}

func (e ClientEngine) DeleteTunnel(tunnelID, publicPort int) (protocol.Response, error) {
	conn, err := net.DialTimeout("tcp", e.ServerAddress, 5*time.Second)
	if err != nil {
		return protocol.Response{}, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	req := signMessage(e.AuthKey, protocol.Message{Type: protocol.TypeDeleteTunnel, ClientID: e.ClientID, TunnelID: tunnelID, PublicPort: publicPort})
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return protocol.Response{}, err
	}
	var resp protocol.Response
	err = json.NewDecoder(conn).Decode(&resp)
	return resp, err
}

func (e ClientEngine) Run(tunnels []config.Tunnel) error {
	if e.ServerAddress == "" {
		return fmt.Errorf("server address is not configured")
	}
	for {
		if err := e.runOnce(tunnels); err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
	}
}

func (e ClientEngine) runOnce(tunnels []config.Tunnel) error {
	return e.runOnceContext(context.Background(), tunnels)
}
func (e ClientEngine) runOnceContext(ctx context.Context, tunnels []config.Tunnel) error {
	conn, err := net.DialTimeout("tcp", e.ServerAddress, 10*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()
	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	req := signMessage(e.AuthKey, protocol.Message{Type: protocol.TypeAuth, ClientID: e.ClientID, Tunnels: toProtocolTunnels(tunnels)})
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return err
	}
	// A small heartbeat detects stale NAT mappings and silent disconnects.
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if json.NewEncoder(conn).Encode(protocol.Message{Type: protocol.TypeHeartbeat}) != nil {
					_ = conn.Close()
					return
				}
			}
		}
	}()
	dec := json.NewDecoder(conn)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(65 * time.Second))
		var msg protocol.Message
		if err := dec.Decode(&msg); err != nil {
			return err
		}
		if msg.Type == protocol.TypeOpenStream {
			for _, t := range tunnels {
				if t.ID == msg.TunnelID && t.LocalPort == msg.LocalPort && t.PublicPort == msg.PublicPort {
					go e.openStream(msg)
					break
				}
			}
		}
	}
}

func (e ClientEngine) openStream(msg protocol.Message) {
	localConn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", msg.LocalPort), 10*time.Second)
	if err != nil {
		return
	}
	serverConn, err := net.DialTimeout("tcp", e.ServerAddress, 10*time.Second)
	if err != nil {
		_ = localConn.Close()
		return
	}
	_ = serverConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	req := signMessage(e.AuthKey, protocol.Message{Type: protocol.TypeDataConn, ClientID: e.ClientID, RequestID: msg.RequestID, TunnelID: msg.TunnelID})
	if err := json.NewEncoder(serverConn).Encode(req); err != nil {
		_ = localConn.Close()
		_ = serverConn.Close()
		return
	}
	_ = serverConn.SetWriteDeadline(time.Time{})
	proxy(localConn, serverConn)
}

func toProtocolTunnels(tunnels []config.Tunnel) []protocol.TunnelStatus {
	out := make([]protocol.TunnelStatus, 0, len(tunnels))
	for _, t := range tunnels {
		out = append(out, protocol.TunnelStatus{TunnelID: t.ID, LocalPort: t.LocalPort, PublicPort: t.PublicPort, Status: t.Status})
	}
	return out
}

// Configuration commands take effect without requiring a manual daemon restart.
func RunConfiguredClient(load func() (config.ClientConfig, error)) error {
	for {
		cfg, err := load()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)
		go func() { result <- NewClientEngine(cfg).runOnceContext(ctx, cfg.Tunnels) }()
		ticker := time.NewTicker(2 * time.Second)
		changed := false
	wait:
		for {
			select {
			case err = <-result:
				if err != nil {
					log.Printf("Control connection ended; retrying in 3s: %v", err)
				}
				break wait
			case <-ticker.C:
				next, loadErr := load()
				if loadErr == nil && !reflect.DeepEqual(cfg, next) {
					changed = true
					cancel()
					<-result
					break wait
				}
			}
		}
		ticker.Stop()
		cancel()
		if !changed {
			time.Sleep(3 * time.Second)
		}
	}
}
