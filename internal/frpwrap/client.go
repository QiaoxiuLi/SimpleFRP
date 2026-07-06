package frpwrap

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
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
	conn, err := net.DialTimeout("tcp", e.ServerAddress, 10*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	req := signMessage(e.AuthKey, protocol.Message{Type: protocol.TypeAuth, ClientID: e.ClientID, Tunnels: toProtocolTunnels(tunnels)})
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return err
	}
	dec := json.NewDecoder(conn)
	for {
		var msg protocol.Message
		if err := dec.Decode(&msg); err != nil {
			return err
		}
		if msg.Type == protocol.TypeOpenStream {
			go e.openStream(msg)
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
	req := signMessage(e.AuthKey, protocol.Message{Type: protocol.TypeDataConn, ClientID: e.ClientID, RequestID: msg.RequestID, TunnelID: msg.TunnelID})
	if err := json.NewEncoder(serverConn).Encode(req); err != nil {
		_ = localConn.Close()
		_ = serverConn.Close()
		return
	}
	pipe(localConn, serverConn)
}

func toProtocolTunnels(tunnels []config.Tunnel) []protocol.TunnelStatus {
	out := make([]protocol.TunnelStatus, 0, len(tunnels))
	for _, t := range tunnels {
		out = append(out, protocol.TunnelStatus{TunnelID: t.ID, LocalPort: t.LocalPort, PublicPort: t.PublicPort, Status: t.Status})
	}
	return out
}

func pipe(a, b net.Conn) {
	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(b, a)
		_ = b.Close()
		_ = a.Close()
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(a, b)
		_ = a.Close()
		_ = b.Close()
		done <- struct{}{}
	}()
	<-done
}
