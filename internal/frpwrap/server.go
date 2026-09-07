package frpwrap

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/crypto"
	"github.com/simplefrp/simplefrp/internal/protocol"
	"github.com/simplefrp/simplefrp/internal/storage"
)

type ServerEngine struct {
	cfg       config.ServerConfig
	store     *storage.Store
	mu        sync.Mutex
	clients   map[string]*clientSession
	listeners map[int]net.Listener
	pending   map[string]*pendingStream
	bindings  map[int]tunnelBinding
	control   net.Listener
}

type tunnelBinding struct {
	clientID string
	record   storage.TunnelRecord
}
type pendingStream struct {
	clientID string
	tunnelID int
	channel  chan net.Conn
}

type clientSession struct {
	conn net.Conn
	id   string
	enc  *json.Encoder
	mu   sync.Mutex
}

func NewServerEngine(cfg config.ServerConfig, store ...*storage.Store) *ServerEngine {
	var s *storage.Store
	if len(store) > 0 {
		s = store[0]
	}
	return &ServerEngine{
		cfg:       cfg,
		store:     s,
		clients:   map[string]*clientSession{},
		listeners: map[int]net.Listener{},
		pending:   map[string]*pendingStream{},
		bindings:  map[int]tunnelBinding{},
	}
}

func (e *ServerEngine) ListenAndServe() error {
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", e.cfg.BindAddress, e.cfg.ControlPort))
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.control = ln
	e.mu.Unlock()
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go e.handle(conn)
	}
}

func (e *ServerEngine) handle(conn net.Conn) {
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	reader := bufio.NewReaderSize(conn, 64*1024)
	line, err := reader.ReadSlice('\n')
	var msg protocol.Message
	if err != nil || json.Unmarshal(line, &msg) != nil {
		_ = conn.Close()
		return
	}
	conn = &bufferedConn{Conn: conn, reader: reader}
	if !e.authorized(msg) {
		_ = json.NewEncoder(conn).Encode(protocol.Response{OK: false, Code: "AUTH_FAILED", Message: "Authentication failed."})
		_ = conn.Close()
		return
	}
	_ = conn.SetDeadline(time.Time{})
	switch msg.Type {
	case protocol.TypeAuth:
		e.handleControl(conn, msg)
	case protocol.TypeDataConn:
		e.handleDataConn(conn, msg)
	case protocol.TypeCreateTunnel:
		e.handleCreateTunnel(conn, msg)
	case protocol.TypeDeleteTunnel:
		e.handleDeleteTunnel(conn, msg)
	default:
		_ = json.NewEncoder(conn).Encode(protocol.Response{OK: false, Code: "UNKNOWN_COMMAND", Message: "Unsupported request."})
		_ = conn.Close()
	}
}

func (e *ServerEngine) authorized(msg protocol.Message) bool {
	return verifyMessage(e.cfg.AuthKey, msg)
}

func (e *ServerEngine) handleControl(conn net.Conn, msg protocol.Message) {
	defer conn.Close()
	session := &clientSession{id: msg.ClientID, conn: conn, enc: json.NewEncoder(conn)}
	e.mu.Lock()
	old := e.clients[msg.ClientID]
	e.clients[msg.ClientID] = session
	e.mu.Unlock()
	if old != nil {
		_ = old.conn.Close()
	}
	defer func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		if e.clients[msg.ClientID] != session {
			return
		}
		delete(e.clients, msg.ClientID)
		for _, binding := range e.bindings {
			if binding.clientID == msg.ClientID && e.store != nil {
				record := binding.record
				record.Status = "offline"
				_ = e.store.SetTunnel(record)
			}
		}
	}()
	wanted := map[int]bool{}
	for _, t := range msg.Tunnels {
		wanted[t.PublicPort] = true
		if err := e.ensureTunnel(session, storage.TunnelRecord{ID: t.TunnelID, LocalPort: t.LocalPort, PublicPort: t.PublicPort, Status: "success"}); err != nil {
			return
		}
	}
	// Reconnecting with an updated configuration releases this client's removed ports.
	e.mu.Lock()
	for port, binding := range e.bindings {
		if binding.clientID == msg.ClientID && !wanted[port] {
			if ln := e.listeners[port]; ln != nil {
				_ = ln.Close()
			}
			delete(e.listeners, port)
			delete(e.bindings, port)
			if e.store != nil {
				_ = e.store.DeleteTunnel(port)
			}
		}
	}
	e.mu.Unlock()
	if session.send(protocol.Message{Type: protocol.TypeHeartbeat}) != nil {
		return
	}
	decoder := json.NewDecoder(conn)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(65 * time.Second))
		var heartbeat protocol.Message
		if decoder.Decode(&heartbeat) != nil {
			return
		}
		if heartbeat.Type != protocol.TypeHeartbeat {
			return
		}
		if session.send(protocol.Message{Type: protocol.TypeHeartbeat}) != nil {
			return
		}
	}
}

func (e *ServerEngine) handleCreateTunnel(conn net.Conn, msg protocol.Message) {
	defer conn.Close()
	publicPort := msg.RequestedPublicPort
	if publicPort == 0 {
		next, err := e.nextPublicPort()
		if err != nil {
			_ = json.NewEncoder(conn).Encode(protocol.Response{OK: false, Code: "SERVER_PORT_IN_USE", Message: "No public port is available."})
			return
		}
		publicPort = next
	}
	session := e.client(msg.ClientID)
	if session == nil {
		session = &clientSession{id: msg.ClientID}
	}
	record := storage.TunnelRecord{ID: msg.TunnelID, LocalPort: msg.LocalPort, PublicPort: publicPort, Status: "success"}
	if err := e.ensureTunnel(session, record); err != nil {
		_ = json.NewEncoder(conn).Encode(protocol.Response{OK: false, Code: "SERVER_PORT_IN_USE", Message: err.Error()})
		return
	}
	_ = json.NewEncoder(conn).Encode(protocol.Response{OK: true, TunnelID: msg.TunnelID, LocalPort: msg.LocalPort, PublicPort: publicPort, Message: "Tunnel created successfully."})
}

func (e *ServerEngine) handleDeleteTunnel(conn net.Conn, msg protocol.Message) {
	defer conn.Close()
	e.mu.Lock()
	binding, exists := e.bindings[msg.PublicPort]
	if exists && (binding.clientID != msg.ClientID || binding.record.ID != msg.TunnelID) {
		e.mu.Unlock()
		_ = json.NewEncoder(conn).Encode(protocol.Response{OK: false, Code: "AUTH_FAILED", Message: "Tunnel belongs to another client."})
		return
	}
	ln := e.listeners[msg.PublicPort]
	delete(e.listeners, msg.PublicPort)
	delete(e.bindings, msg.PublicPort)
	e.mu.Unlock()
	if ln != nil {
		_ = ln.Close()
	}
	if e.store != nil {
		_ = e.store.DeleteTunnel(msg.PublicPort)
	}
	_ = json.NewEncoder(conn).Encode(protocol.Response{OK: true, Message: "Tunnel deleted successfully."})
}

func (e *ServerEngine) handleDataConn(conn net.Conn, msg protocol.Message) {
	e.mu.Lock()
	defer e.mu.Unlock()
	pending := e.pending[msg.RequestID]
	if pending == nil || pending.clientID != msg.ClientID || pending.tunnelID != msg.TunnelID {
		_ = conn.Close()
		return
	}
	delete(e.pending, msg.RequestID)
	pending.channel <- conn
}

func (e *ServerEngine) ensureTunnel(session *clientSession, record storage.TunnelRecord) error {
	if record.PublicPort <= 1024 || record.PublicPort > 65535 || record.LocalPort <= 0 || record.LocalPort > 65535 {
		return errors.New("invalid tunnel ports")
	}
	if e.cfg.PortMin > 0 && (record.PublicPort < e.cfg.PortMin || record.PublicPort > e.cfg.PortMax) {
		return errors.New("public port is outside the configured range")
	}
	e.mu.Lock()
	if binding, ok := e.bindings[record.PublicPort]; ok && (binding.clientID != session.id || binding.record.ID != record.ID) {
		e.mu.Unlock()
		return errors.New("public port belongs to another tunnel")
	}
	e.bindings[record.PublicPort] = tunnelBinding{session.id, record}
	if _, ok := e.listeners[record.PublicPort]; ok {
		e.mu.Unlock()
		if e.store != nil {
			_ = e.store.SetTunnel(record)
		}
		return nil
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", e.cfg.BindAddress, record.PublicPort))
	if err != nil {
		delete(e.bindings, record.PublicPort)
		e.mu.Unlock()
		return err
	}
	e.listeners[record.PublicPort] = ln
	e.mu.Unlock()
	if e.store != nil {
		_ = e.store.SetTunnel(record)
	}
	go e.acceptTunnel(ln, record.PublicPort)
	return nil
}

func (e *ServerEngine) acceptTunnel(ln net.Listener, port int) {
	for {
		publicConn, err := ln.Accept()
		if err != nil {
			return
		}
		e.mu.Lock()
		binding, ok := e.bindings[port]
		session := e.clients[binding.clientID]
		e.mu.Unlock()
		if !ok || session == nil {
			_ = publicConn.Close()
			continue
		}
		go e.bridgePublicConn(session, publicConn, binding.record)
	}
}

func (e *ServerEngine) bridgePublicConn(session *clientSession, publicConn net.Conn, record storage.TunnelRecord) {
	started := time.Now()
	requestID, _ := crypto.RandomToken(18)
	dataCh := make(chan net.Conn, 1)
	e.mu.Lock()
	if len(e.pending) >= 256 {
		e.mu.Unlock()
		_ = publicConn.Close()
		return
	}
	e.pending[requestID] = &pendingStream{session.id, record.ID, dataCh}
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		delete(e.pending, requestID)
		e.mu.Unlock()
		select {
		case conn := <-dataCh:
			_ = conn.Close()
		default:
		}
	}()
	if err := session.send(protocol.Message{Type: protocol.TypeOpenStream, RequestID: requestID, TunnelID: record.ID, LocalPort: record.LocalPort, PublicPort: record.PublicPort}); err != nil {
		_ = publicConn.Close()
		e.logConnection(publicConn.RemoteAddr(), record, started, "failed", err.Error())
		return
	}
	select {
	case dataConn := <-dataCh:
		uploaded, downloaded := proxy(publicConn, dataConn)
		if e.store != nil {
			_ = e.store.AddTraffic(uploaded, downloaded)
		}
		e.logConnection(publicConn.RemoteAddr(), record, started, "success", "")
	case <-time.After(15 * time.Second):
		_ = publicConn.Close()
		e.logConnection(publicConn.RemoteAddr(), record, started, "failed", "client data connection timeout")
	}
}

func (e *ServerEngine) nextPublicPort() (int, error) {
	min := e.cfg.PortMin
	max := e.cfg.PortMax
	if min <= 0 {
		min = 20000
	}
	if max < min {
		max = 60000
	}
	for port := min; port <= max; port++ {
		e.mu.Lock()
		_, exists := e.listeners[port]
		e.mu.Unlock()
		if exists {
			continue
		}
		ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", e.cfg.BindAddress, port))
		if err == nil {
			_ = ln.Close()
			return port, nil
		}
	}
	return 0, errors.New("no available public port")
}

func (e *ServerEngine) client(id string) *clientSession {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.clients[id]
}

func (s *clientSession) send(msg protocol.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return s.enc.Encode(msg)
}

func (e *ServerEngine) logConnection(addr net.Addr, record storage.TunnelRecord, started time.Time, status, reason string) {
	if e.store == nil {
		return
	}
	host := ""
	if addr != nil {
		host, _, _ = net.SplitHostPort(addr.String())
	}
	id, _ := crypto.RandomToken(10)
	_ = e.store.AddConnection(storage.ConnectionLog{
		ID:          id,
		IPv4:        host,
		ConnectedAt: started,
		Duration:    time.Since(started).String(),
		TunnelID:    record.ID,
		LocalPort:   record.LocalPort,
		PublicPort:  record.PublicPort,
		Status:      status,
		ErrorReason: reason,
	})
}

// Close releases this engine's resources for bounded tests and orderly shutdown.
func (e *ServerEngine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.control != nil {
		_ = e.control.Close()
	}
	for _, ln := range e.listeners {
		_ = ln.Close()
	}
	for _, client := range e.clients {
		_ = client.conn.Close()
	}
}
