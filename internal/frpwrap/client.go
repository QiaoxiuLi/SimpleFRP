package frpwrap

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/crypto"
	"github.com/simplefrp/simplefrp/internal/protocol"
)

type ClientEngine struct {
	ServerAddress string
	ClientID      string
	AuthKey       string
	Fingerprint   string
	Admin         bool
}

func NewClientEngine(cfg config.ClientConfig) ClientEngine {
	return ClientEngine{ServerAddress: cfg.ServerAddress, ClientID: cfg.ClientID, AuthKey: cfg.ClientKey, Fingerprint: cfg.Fingerprint}
}
func NewAdmin(cfg config.ServerConfig) ClientEngine {
	return ClientEngine{ServerAddress: net.JoinHostPort("127.0.0.1", strconv.Itoa(cfg.ControlPort)), AuthKey: cfg.AdminKey, Fingerprint: cfg.Fingerprint, Admin: true}
}
func (e ClientEngine) dial(ctx context.Context) (net.Conn, error) {
	pinned, err := crypto.PinnedTLS(e.Fingerprint)
	if err != nil {
		return nil, err
	}
	dialer := tls.Dialer{NetDialer: &net.Dialer{Timeout: 5 * time.Second}, Config: pinned}
	return dialer.DialContext(ctx, "tcp", e.ServerAddress)
}
func (e ClientEngine) Request(msg protocol.Message) (protocol.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	return e.RequestContext(ctx, msg)
}
func (e ClientEngine) RequestContext(ctx context.Context, msg protocol.Message) (protocol.Response, error) {
	conn, err := e.dial(ctx)
	if err != nil {
		return protocol.Response{}, fmt.Errorf("server unreachable or identity verification failed: %w", err)
	}
	defer conn.Close()
	deadline := time.Now().Add(25 * time.Second)
	if limit, ok := ctx.Deadline(); ok && limit.Before(deadline) {
		deadline = limit
	}
	conn.SetDeadline(deadline)
	msg.ClientID = e.ClientID
	msg.Admin = e.Admin
	msg = signMessage(e.AuthKey, msg)
	if err = json.NewEncoder(conn).Encode(msg); err != nil {
		return protocol.Response{}, err
	}
	var resp protocol.Response
	if err = json.NewDecoder(bufio.NewReader(conn)).Decode(&resp); err != nil {
		return resp, err
	}
	if !resp.OK {
		return resp, errors.New(resp.Message)
	}
	return resp, nil
}
func Pair(invite crypto.Invite, local int) (config.ClientConfig, error) {
	engine := ClientEngine{ServerAddress: invite.Address(), AuthKey: invite.Key, Fingerprint: invite.Fingerprint}
	resp, err := engine.Request(protocol.Message{Type: protocol.TypePair, LocalPort: local})
	if err != nil {
		return config.ClientConfig{}, err
	}
	return config.ClientConfig{Version: 2, ServerID: invite.ServerID, ServerAddress: invite.Address(), Fingerprint: invite.Fingerprint, ClientID: resp.ClientID, ClientKey: resp.ClientKey, Tunnels: resp.Tunnels}, nil
}
func (e ClientEngine) CreateTunnel(local int) (protocol.Response, error) {
	return e.Request(protocol.Message{Type: protocol.TypeCreateTunnel, LocalPort: local})
}

type ClientCallbacks struct {
	Load   func() (config.ClientConfig, error)
	Save   func(config.ClientConfig) error
	Status func(protocol.Status)
}

var errServerMoved = errors.New("server endpoint updated")

func RunClient(ctx context.Context, callbacks ClientCallbacks) error {
	for {
		if ctx.Err() != nil {
			return nil
		}
		cfg, err := callbacks.Load()
		if err == nil && cfg.ServerAddress != "" {
			err = NewClientEngine(cfg).run(ctx, cfg, callbacks)
		}
		if errors.Is(err, errServerMoved) {
			continue
		}
		if callbacks.Status != nil {
			status := protocol.Status{Tunnels: []protocol.TunnelStatus{}}
			_, p, _ := net.SplitHostPort(cfg.ServerAddress)
			status.HeartbeatPort, _ = strconv.Atoi(p)
			for _, t := range cfg.Tunnels {
				status.Tunnels = append(status.Tunnels, protocol.TunnelStatus{ID: t.ID, LocalPort: t.LocalPort, PublicPort: t.PublicPort, Status: "offline"})
			}
			callbacks.Status(status)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(3 * time.Second):
		}
	}
}
func (e ClientEngine) run(ctx context.Context, cfg config.ClientConfig, callbacks ClientCallbacks) error {
	conn, err := e.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		case <-done:
		}
	}()
	var writeMu sync.Mutex
	send := func(msg protocol.Message) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		return json.NewEncoder(conn).Encode(msg)
	}
	if err = send(signMessage(e.AuthKey, protocol.Message{Type: protocol.TypeAuth, ClientID: e.ClientID})); err != nil {
		return err
	}
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if send(protocol.Message{Type: protocol.TypeHeartbeat}) != nil {
					conn.Close()
					return
				}
			}
		}
	}()
	var cfgMu sync.Mutex
	decoder := json.NewDecoder(conn)
	for {
		conn.SetReadDeadline(time.Now().Add(65 * time.Second))
		var msg protocol.Message
		if err = decoder.Decode(&msg); err != nil {
			return err
		}
		switch msg.Type {
		case protocol.TypeConfigure:
			cfgMu.Lock()
			candidate := cfg
			candidate.Tunnels = msg.Tunnels
			saveErr := callbacks.Save(candidate)
			if saveErr == nil {
				cfg = candidate
			}
			cfgMu.Unlock()
			if err = send(protocol.Message{Type: protocol.TypeAck, RequestID: msg.RequestID, OK: saveErr == nil}); err != nil {
				return err
			}
			if saveErr != nil && msg.Kind == "initial" {
				return saveErr
			}
		case protocol.TypeMoved:
			host, port, splitErr := net.SplitHostPort(msg.Address)
			oldHost, _, _ := net.SplitHostPort(e.ServerAddress)
			n, _ := strconv.Atoi(port)
			if splitErr != nil || host != oldHost || n < 1025 || n > 65535 {
				return errors.New("invalid server endpoint update")
			}
			cfgMu.Lock()
			candidate := cfg
			candidate.ServerAddress = msg.Address
			err = callbacks.Save(candidate)
			cfgMu.Unlock()
			if err != nil {
				return err
			}
			return errServerMoved
		case protocol.TypeOpenStream:
			cfgMu.Lock()
			t, _, ok := config.FindTunnel(cfg.Tunnels, msg.TunnelID)
			cfgMu.Unlock()
			if ok && t.LocalPort == msg.LocalPort && t.PublicPort == msg.PublicPort {
				go e.openStream(ctx, msg)
			}
		}
	}
}
func (e ClientEngine) openStream(ctx context.Context, msg protocol.Message) {
	local, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(msg.LocalPort)))
	if err != nil {
		return
	}
	server, err := e.dial(ctx)
	if err != nil {
		local.Close()
		return
	}
	server.SetWriteDeadline(time.Now().Add(10 * time.Second))
	req := signMessage(e.AuthKey, protocol.Message{Type: protocol.TypeDataConn, ClientID: e.ClientID, RequestID: msg.RequestID, TunnelID: msg.TunnelID})
	if json.NewEncoder(server).Encode(req) != nil {
		local.Close()
		server.Close()
		return
	}
	server.SetWriteDeadline(time.Time{})
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			local.Close()
			server.Close()
		case <-done:
		}
	}()
	proxy(local, server)
}
