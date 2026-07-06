package frpwrap

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	pending   map[string]chan net.Conn
}

type clientSession struct {
	id  string
	enc *json.Encoder
	mu  sync.Mutex
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
		pending:   map[string]chan net.Conn{},
	}
}

func (e *ServerEngine) ListenAndServe() error {
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", e.cfg.BindAddress, e.cfg.ControlPort))
	if err != nil {
		return err
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go e.handle(conn)
	}
}

func (e *ServerEngine) handle(conn net.Conn) {
	reader := bufio.NewReader(conn)
	var msg protocol.Message
	if err := json.NewDecoder(reader).Decode(&msg); err != nil {
		_ = json.NewEncoder(conn).Encode(protocol.Response{OK: false, Code: "UNKNOWN_ERROR", Message: "Unable to read client request."})
		_ = conn.Close()
		return
	}
	if !e.authorized(msg) {
		_ = json.NewEncoder(conn).Encode(protocol.Response{OK: false, Code: "AUTH_FAILED", Message: "Authentication failed."})
		_ = conn.Close()
		return
	}
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
	session := &clientSession{id: msg.ClientID, enc: json.NewEncoder(conn)}
	e.mu.Lock()
	e.clients[msg.ClientID] = session
	e.mu.Unlock()
	_ = session.send(protocol.Message{Type: protocol.TypeHeartbeat})
	for _, t := range msg.Tunnels {
		_ = e.ensureTunnel(session, storage.TunnelRecord{ID: t.TunnelID, LocalPort: t.LocalPort, PublicPort: t.PublicPort, Status: "success"})
	}
	_, _ = io.Copy(io.Discard, conn)
	e.mu.Lock()
	if e.clients[msg.ClientID] == session {
		delete(e.clients, msg.ClientID)
	}
	e.mu.Unlock()
	_ = conn.Close()
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
		if err := e.checkPublicPort(publicPort); err != nil {
			_ = json.NewEncoder(conn).Encode(protocol.Response{OK: false, Code: "SERVER_PORT_IN_USE", Message: err.Error()})
			return
		}
		if e.store != nil {
			_ = e.store.SetTunnel(storage.TunnelRecord{ID: msg.TunnelID, LocalPort: msg.LocalPort, PublicPort: publicPort, Status: "pending"})
		}
		_ = json.NewEncoder(conn).Encode(protocol.Response{OK: true, TunnelID: msg.TunnelID, LocalPort: msg.LocalPort, PublicPort: publicPort, Message: "Tunnel reserved successfully."})
		return
	}
	record := storage.TunnelRecord{ID: msg.TunnelID, LocalPort: msg.LocalPort, PublicPort: publicPort, Status: "success"}
	if err := e.ensureTunnel(session, record); err != nil {
		_ = json.NewEncoder(conn).Encode(protocol.Response{OK: false, Code: "SERVER_PORT_IN_USE", Message: err.Error()})
		return
	}
	_ = json.NewEncoder(conn).Encode(protocol.Response{OK: true, TunnelID: msg.TunnelID, LocalPort: msg.LocalPort, PublicPort: publicPort, Message: "Tunnel created successfully."})
}

func (e *ServerEngine) checkPublicPort(port int) error {
	e.mu.Lock()
	_, exists := e.listeners[port]
	e.mu.Unlock()
	if exists {
		return errors.New("public port is already in use")
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", e.cfg.BindAddress, port))
	if err != nil {
		return err
	}
	return ln.Close()
}

func (e *ServerEngine) handleDeleteTunnel(conn net.Conn, msg protocol.Message) {
	defer conn.Close()
	e.mu.Lock()
	ln := e.listeners[msg.PublicPort]
	delete(e.listeners, msg.PublicPort)
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
	ch := e.pending[msg.RequestID]
	delete(e.pending, msg.RequestID)
	e.mu.Unlock()
	if ch == nil {
		_ = conn.Close()
		return
	}
	ch <- conn
}

func (e *ServerEngine) ensureTunnel(session *clientSession, record storage.TunnelRecord) error {
	if record.PublicPort <= 0 || record.LocalPort <= 0 {
		return errors.New("invalid tunnel ports")
	}
	e.mu.Lock()
	if _, ok := e.listeners[record.PublicPort]; ok {
		e.mu.Unlock()
		if e.store != nil {
			_ = e.store.SetTunnel(record)
		}
		return nil
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", e.cfg.BindAddress, record.PublicPort))
	if err != nil {
		e.mu.Unlock()
		return err
	}
	e.listeners[record.PublicPort] = ln
	e.mu.Unlock()
	if e.store != nil {
		_ = e.store.SetTunnel(record)
	}
	go e.acceptTunnel(session, ln, record)
	return nil
}

func (e *ServerEngine) acceptTunnel(session *clientSession, ln net.Listener, record storage.TunnelRecord) {
	for {
		publicConn, err := ln.Accept()
		if err != nil {
			return
		}
		go e.bridgePublicConn(session, publicConn, record)
	}
}

func (e *ServerEngine) bridgePublicConn(session *clientSession, publicConn net.Conn, record storage.TunnelRecord) {
	started := time.Now()
	requestID, _ := crypto.RandomToken(18)
	dataCh := make(chan net.Conn, 1)
	e.mu.Lock()
	e.pending[requestID] = dataCh
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		delete(e.pending, requestID)
		e.mu.Unlock()
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

func proxy(a, b net.Conn) (int64, int64) {
	var wg sync.WaitGroup
	var uploaded int64
	var downloaded int64
	wg.Add(2)
	go func() {
		defer wg.Done()
		uploaded, _ = io.Copy(b, a)
		_ = b.Close()
		_ = a.Close()
	}()
	go func() {
		defer wg.Done()
		downloaded, _ = io.Copy(a, b)
		_ = a.Close()
		_ = b.Close()
	}()
	wg.Wait()
	return uploaded, downloaded
}
