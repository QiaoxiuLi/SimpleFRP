package frpwrap

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/crypto"
	"github.com/simplefrp/simplefrp/internal/protocol"
	"github.com/simplefrp/simplefrp/internal/storage"
	"github.com/simplefrp/simplefrp/internal/sysutil"
)

type persistedTunnel struct {
	Tunnel   config.Tunnel
	ClientID string
	Total    int64
}
type runtimeState struct {
	NextID  int
	Clients map[string]string
	Tunnels []persistedTunnel
}
type tunnelBinding struct {
	record   config.Tunnel
	clientID string
	total    atomic.Int64
	last     int64
	speed    float64
}
type clientSession struct {
	conn net.Conn
	id   string
	enc  *json.Encoder
	mu   sync.Mutex
}
type pendingStream struct {
	clientID string
	tunnelID int
	channel  chan net.Conn
}
type ServerEngine struct {
	cfg             config.ServerConfig
	store           *storage.Store
	mu              sync.Mutex
	mutation        sync.Mutex
	clients         map[string]*clientSession
	keys            map[string]string
	bindings        map[int]*tunnelBinding
	listeners       map[int]net.Listener
	controls        []net.Listener
	pending         map[string]*pendingStream
	acks            map[string]chan bool
	replay          map[string]time.Time
	conns           map[net.Conn]int
	nextID          int
	done            chan struct{}
	once            sync.Once
	workers         sync.WaitGroup
	workerMu        sync.Mutex
	closing         bool
	slots           chan struct{}
	RebindDashboard func(int) error
	SaveConfig      func(config.ServerConfig) error
}

func NewServerEngine(cfg config.ServerConfig, stores ...*storage.Store) *ServerEngine {
	e := &ServerEngine{cfg: cfg, clients: map[string]*clientSession{}, keys: map[string]string{}, bindings: map[int]*tunnelBinding{}, listeners: map[int]net.Listener{}, pending: map[string]*pendingStream{}, acks: map[string]chan bool{}, replay: map[string]time.Time{}, conns: map[net.Conn]int{}, done: make(chan struct{}), slots: make(chan struct{}, 1024), SaveConfig: config.SaveServer}
	if len(stores) > 0 {
		e.store = stores[0]
	}
	return e
}
func (e *ServerEngine) Start() error {
	pair, err := tls.X509KeyPair([]byte(e.cfg.Certificate), []byte(e.cfg.PrivateKey))
	if err != nil {
		return err
	}
	var saved runtimeState
	if err = e.store.ReadRuntime(&saved); err != nil {
		return err
	}
	e.nextID = saved.NextID
	if saved.Clients != nil {
		e.keys = saved.Clients
	}
	for _, p := range saved.Tunnels {
		b := &tunnelBinding{record: p.Tunnel, clientID: p.ClientID}
		b.total.Store(p.Total)
		b.last = p.Total
		e.bindings[p.Tunnel.ID] = b
	}
	ln, err := tls.Listen("tcp", net.JoinHostPort(e.cfg.BindAddress, strconv.Itoa(e.cfg.ControlPort)), &tls.Config{Certificates: []tls.Certificate{pair}, MinVersion: tls.VersionTLS13})
	if err != nil {
		return fmt.Errorf("heartbeat port unavailable: %w", err)
	}
	e.controls = append(e.controls, ln)
	if len(e.bindings) == 0 {
		if _, err = e.listenPublic(e.cfg.FirstPublicPort); err != nil {
			e.Close()
			return err
		}
	} else {
		for _, b := range e.bindings {
			if _, err = e.listenPublic(b.record.PublicPort); err != nil {
				e.Close()
				return err
			}
		}
	}
	e.spawn(func() { e.acceptControl(ln) })
	e.spawn(e.sample)
	return nil
}
func (e *ServerEngine) spawn(fn func()) bool {
	e.workerMu.Lock()
	defer e.workerMu.Unlock()
	if e.closing {
		return false
	}
	e.workers.Add(1)
	go func() { defer e.workers.Done(); fn() }()
	return true
}
func (e *ServerEngine) ListenAndServe() error {
	if err := e.Start(); err != nil {
		return err
	}
	<-e.done
	return nil
}
func (e *ServerEngine) acceptControl(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		select {
		case e.slots <- struct{}{}:
		default:
			conn.Close()
			continue
		}
		e.mu.Lock()
		e.conns[conn] = -1
		e.mu.Unlock()
		if !e.spawn(func() {
			defer func() { <-e.slots; e.mu.Lock(); delete(e.conns, conn); e.mu.Unlock() }()
			e.handle(conn)
		}) {
			conn.Close()
			<-e.slots
			e.mu.Lock()
			delete(e.conns, conn)
			e.mu.Unlock()
		}
	}
}
func (e *ServerEngine) handle(conn net.Conn) {
	conn.SetDeadline(time.Now().Add(12 * time.Second))
	reader := bufio.NewReaderSize(conn, 64*1024)
	line, err := reader.ReadSlice('\n')
	var msg protocol.Message
	if err != nil || json.Unmarshal(line, &msg) != nil {
		conn.Close()
		return
	}
	conn = &bufferedConn{Conn: conn, reader: reader}
	if !e.authorized(msg) {
		json.NewEncoder(conn).Encode(protocol.Response{Code: "AUTH_FAILED", Message: "Authentication failed"})
		conn.Close()
		return
	}
	if msg.Type == protocol.TypeAuth {
		conn.SetDeadline(time.Time{})
		e.handleControl(conn, msg)
		return
	}
	if msg.Type == protocol.TypeDataConn {
		conn.SetDeadline(time.Time{})
		e.handleData(conn, msg)
		return
	}
	defer conn.Close()
	response := e.request(msg)
	json.NewEncoder(conn).Encode(response)
}
func (e *ServerEngine) authorized(msg protocol.Message) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	key := e.keys[msg.ClientID]
	if msg.Admin {
		key = e.cfg.AdminKey
	} else if msg.Type == protocol.TypePair {
		key = e.cfg.InviteKey
	}
	if !verifyMessage(key, msg) {
		return false
	}
	now := time.Now()
	for nonce, at := range e.replay {
		if now.Sub(at) > 2*time.Minute {
			delete(e.replay, nonce)
		}
	}
	if _, exists := e.replay[msg.Nonce]; exists || len(e.replay) >= 8192 {
		return false
	}
	e.replay[msg.Nonce] = now
	return true
}
func (e *ServerEngine) request(msg protocol.Message) protocol.Response {
	if msg.Type == protocol.TypeStatus {
		status := e.Status(msg.ClientID, msg.Admin)
		return protocol.Response{OK: true, Status: &status}
	}
	e.mutation.Lock()
	defer e.mutation.Unlock()
	select {
	case <-e.done:
		return protocol.Response{Code: "STOPPING", Message: "Server is stopping"}
	default:
	}
	var resp protocol.Response
	var err error
	switch msg.Type {
	case protocol.TypeUnpair:
		if msg.Admin {
			err = errors.New("unpair is a client lifecycle request")
			break
		}
		err = e.unpair(msg.ClientID)
	case protocol.TypePair:
		if msg.Admin {
			err = errors.New("invalid invitation request")
			break
		}
		resp, err = e.pair(msg.LocalPort)
	case protocol.TypeCreateTunnel:
		resp, err = e.create(msg.ClientID, msg.LocalPort)
		if err == nil {
			e.mu.Lock()
			session := e.clients[msg.ClientID]
			e.mu.Unlock()
			if session == nil {
				err = errors.New("client is offline; start it before adding a tunnel")
			} else {
				err = e.configure(session, resp.Tunnels)
			}
			if err != nil {
				e.mu.Lock()
				delete(e.bindings, resp.TunnelID)
				if ln := e.listeners[resp.PublicPort]; ln != nil {
					ln.Close()
					delete(e.listeners, resp.PublicPort)
				}
				_ = e.persistLocked()
				e.mu.Unlock()
			}
		}
	case protocol.TypeUpdate, protocol.TypeDeleteTunnel:
		resp, err = e.changeTunnel(msg)
	case protocol.TypePort:
		if !msg.Admin {
			err = errors.New("server administrator permission required")
			break
		}
		err = e.changePort(msg.Kind, msg.Port)
		resp.OK = err == nil
	default:
		err = errors.New("unsupported v2 request")
	}
	if err != nil {
		return protocol.Response{Code: "OPERATION_FAILED", Message: err.Error()}
	}
	return resp
}
func (e *ServerEngine) unpair(clientID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	key, exists := e.keys[clientID]
	if !exists {
		return errors.New("client is not paired")
	}
	removed := map[int]*tunnelBinding{}
	for id, b := range e.bindings {
		if b.clientID == clientID {
			removed[id] = b
			delete(e.bindings, id)
		}
	}
	delete(e.keys, clientID)
	if err := e.persistLocked(); err != nil {
		e.keys[clientID] = key
		for id, b := range removed {
			e.bindings[id] = b
		}
		return err
	}
	for _, b := range removed {
		if ln := e.listeners[b.record.PublicPort]; ln != nil {
			ln.Close()
			delete(e.listeners, b.record.PublicPort)
		}
	}
	for conn, id := range e.conns {
		if removed[id] != nil {
			conn.Close()
		}
	}
	if session := e.clients[clientID]; session != nil {
		session.conn.Close()
		delete(e.clients, clientID)
	}
	if len(e.bindings) == 0 && e.listeners[e.cfg.FirstPublicPort] == nil {
		_, _ = e.listenPublic(e.cfg.FirstPublicPort)
	}
	return nil
}
func (e *ServerEngine) pair(local int) (protocol.Response, error) {
	if local < 1 || local > 65535 {
		return protocol.Response{}, errors.New("invalid local port")
	}
	e.mu.Lock()
	if len(e.keys) >= 64 {
		e.mu.Unlock()
		return protocol.Response{}, errors.New("client limit reached")
	}
	e.mu.Unlock()
	id, err := crypto.RandomToken(16)
	if err != nil {
		return protocol.Response{}, err
	}
	key, err := crypto.RandomToken(32)
	if err != nil {
		return protocol.Response{}, err
	}
	e.mu.Lock()
	e.keys[id] = key
	e.mu.Unlock()
	resp, err := e.create(id, local)
	if err != nil {
		e.mu.Lock()
		delete(e.keys, id)
		e.mu.Unlock()
		return resp, err
	}
	resp.ClientID = id
	resp.ClientKey = key
	return resp, nil
}
func (e *ServerEngine) listenPublic(port int) (net.Listener, error) {
	if !sysutil.IsPortAvailable(port) {
		return nil, errors.New("public port is in use; configuration unchanged")
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(e.cfg.BindAddress, strconv.Itoa(port)))
	if err != nil {
		return nil, fmt.Errorf("public port unavailable: %w", err)
	}
	e.listeners[port] = ln
	e.spawn(func() { e.acceptPublic(ln, port) })
	return ln, nil
}
func (e *ServerEngine) create(clientID string, local int) (protocol.Response, error) {
	if local < 1 || local > 65535 {
		return protocol.Response{}, errors.New("invalid local target port")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if clientID == "" || e.keys[clientID] == "" {
		return protocol.Response{}, errors.New("unknown client")
	}
	if len(e.bindings) >= 256 {
		return protocol.Response{}, errors.New("tunnel limit reached")
	}
	port := 0
	if len(e.bindings) == 0 && e.listeners[e.cfg.FirstPublicPort] != nil {
		port = e.cfg.FirstPublicPort
	} else {
		for i := 0; i < 256; i++ {
			candidate, err := sysutil.RandomAvailablePort()
			if err != nil {
				return protocol.Response{}, err
			}
			if candidate == e.cfg.ControlPort || candidate == e.cfg.DashboardPort || e.listeners[candidate] != nil {
				continue
			}
			if _, err = e.listenPublic(candidate); err == nil {
				port = candidate
				break
			}
		}
	}
	if port == 0 {
		return protocol.Response{}, errors.New("no public port available")
	}
	record := config.Tunnel{ID: e.nextID, LocalPort: local, PublicPort: port, Status: "waiting"}
	b := &tunnelBinding{record: record, clientID: clientID}
	e.bindings[record.ID] = b
	e.nextID++
	if err := e.persistLocked(); err != nil {
		delete(e.bindings, record.ID)
		e.nextID--
		if len(e.bindings) > 0 {
			e.listeners[port].Close()
			delete(e.listeners, port)
		}
		return protocol.Response{}, err
	}
	return protocol.Response{OK: true, TunnelID: record.ID, LocalPort: local, PublicPort: port, Tunnels: e.tunnelsLocked(clientID)}, nil
}
func (e *ServerEngine) tunnelsLocked(clientID string) []config.Tunnel {
	result := []config.Tunnel{}
	for _, b := range e.bindings {
		if b.clientID == clientID {
			result = append(result, b.record)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
func (e *ServerEngine) persistLocked() error {
	state := runtimeState{NextID: e.nextID, Clients: e.keys, Tunnels: []persistedTunnel{}}
	for _, b := range e.bindings {
		state.Tunnels = append(state.Tunnels, persistedTunnel{b.record, b.clientID, b.total.Load()})
	}
	return e.store.SaveRuntime(state)
}
func (s *clientSession) send(msg protocol.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return s.enc.Encode(msg)
}
func (e *ServerEngine) handleControl(conn net.Conn, msg protocol.Message) {
	defer conn.Close()
	session := &clientSession{conn: conn, id: msg.ClientID, enc: json.NewEncoder(conn)}
	e.mu.Lock()
	old := e.clients[msg.ClientID]
	e.clients[msg.ClientID] = session
	tunnels := e.tunnelsLocked(msg.ClientID)
	e.mu.Unlock()
	if old != nil {
		old.conn.Close()
	}
	defer func() {
		e.mu.Lock()
		if e.clients[msg.ClientID] == session {
			delete(e.clients, msg.ClientID)
		}
		e.mu.Unlock()
	}()
	if err := session.send(protocol.Message{Type: protocol.TypeConfigure, Tunnels: tunnels, Kind: "initial"}); err != nil {
		return
	}
	reader := bufio.NewReaderSize(conn, 64*1024)
	for {
		conn.SetReadDeadline(time.Now().Add(65 * time.Second))
		var reply protocol.Message
		line, err := reader.ReadSlice('\n')
		if err != nil || json.Unmarshal(line, &reply) != nil {
			return
		}
		switch reply.Type {
		case protocol.TypeHeartbeat:
			if session.send(protocol.Message{Type: protocol.TypeHeartbeat}) != nil {
				return
			}
		case protocol.TypeAck:
			e.mu.Lock()
			ch := e.acks[reply.RequestID]
			e.mu.Unlock()
			if ch != nil {
				select {
				case ch <- reply.OK:
				default:
				}
			}
		default:
			return
		}
	}
}
func (e *ServerEngine) configure(session *clientSession, tunnels []config.Tunnel) error {
	if session == nil {
		return errors.New("client is offline; configuration unchanged")
	}
	id, err := crypto.RandomToken(16)
	if err != nil {
		return err
	}
	ch := make(chan bool, 1)
	e.mu.Lock()
	e.acks[id] = ch
	e.mu.Unlock()
	defer func() { e.mu.Lock(); delete(e.acks, id); e.mu.Unlock() }()
	if err = session.send(protocol.Message{Type: protocol.TypeConfigure, RequestID: id, Tunnels: tunnels}); err != nil {
		return err
	}
	select {
	case ok := <-ch:
		if !ok {
			return errors.New("client could not save configuration")
		}
		return nil
	case <-e.done:
		return errors.New("server stopped")
	case <-time.After(8 * time.Second):
		return errors.New("client confirmation timed out; configuration unchanged")
	}
}
func (e *ServerEngine) changeTunnel(msg protocol.Message) (protocol.Response, error) {
	e.mu.Lock()
	b := e.bindings[msg.TunnelID]
	if b == nil || (!msg.Admin && b.clientID != msg.ClientID) {
		e.mu.Unlock()
		return protocol.Response{}, errors.New("tunnel not found or not owned by this client")
	}
	old := b.record
	candidate := old
	session := e.clients[b.clientID]
	previous := e.tunnelsLocked(b.clientID)
	e.mu.Unlock()
	if session == nil && !(msg.Admin && msg.Type == protocol.TypeDeleteTunnel) {
		return protocol.Response{}, errors.New("client is offline; configuration unchanged")
	}
	if msg.Type == protocol.TypeUpdate {
		if msg.Kind == "local" {
			if msg.Port < 1 || msg.Port > 65535 {
				return protocol.Response{}, errors.New("local port must be 1-65535")
			}
			candidate.LocalPort = msg.Port
		} else if msg.Kind == "public" {
			if err := sysutil.ValidatePort(msg.Port); err != nil {
				return protocol.Response{}, err
			}
			candidate.PublicPort = msg.Port
		} else {
			return protocol.Response{}, errors.New("use local or public")
		}
	}
	newPort := candidate.PublicPort != old.PublicPort
	if newPort {
		e.mu.Lock()
		_, used := e.listeners[candidate.PublicPort]
		if !used {
			_, err := e.listenPublic(candidate.PublicPort)
			used = err != nil
		}
		e.mu.Unlock()
		if used {
			return protocol.Response{}, errors.New("public port is in use; configuration unchanged")
		}
	}
	committed := false
	defer func() {
		if newPort && !committed {
			e.mu.Lock()
			if ln := e.listeners[candidate.PublicPort]; ln != nil {
				ln.Close()
				delete(e.listeners, candidate.PublicPort)
			}
			e.mu.Unlock()
		}
	}()
	next := []config.Tunnel{}
	for _, t := range previous {
		if t.ID == old.ID {
			if msg.Type != protocol.TypeDeleteTunnel {
				next = append(next, candidate)
			}
		} else {
			next = append(next, t)
		}
	}
	if session != nil {
		if err := e.configure(session, next); err != nil {
			_ = e.configure(session, previous)
			return protocol.Response{}, err
		}
	}
	e.mu.Lock()
	if msg.Type == protocol.TypeDeleteTunnel {
		delete(e.bindings, old.ID)
	} else {
		b.record = candidate
	}
	err := e.persistLocked()
	if err != nil {
		e.bindings[old.ID] = b
		b.record = old
	}
	e.mu.Unlock()
	if err != nil {
		if session != nil {
			_ = e.configure(session, previous)
		}
		return protocol.Response{}, err
	}
	e.mu.Lock()
	if newPort || msg.Type == protocol.TypeDeleteTunnel {
		if ln := e.listeners[old.PublicPort]; ln != nil {
			ln.Close()
			delete(e.listeners, old.PublicPort)
		}
	}
	if msg.Type == protocol.TypeDeleteTunnel {
		for conn, id := range e.conns {
			if id == old.ID {
				conn.Close()
			}
		}
	}
	e.mu.Unlock()
	committed = true
	return protocol.Response{OK: true, TunnelID: old.ID, LocalPort: candidate.LocalPort, PublicPort: candidate.PublicPort, Tunnels: next}, nil
}
func (e *ServerEngine) changePort(kind string, port int) error {
	if err := sysutil.ValidatePort(port); err != nil {
		return err
	}
	e.mu.Lock()
	old := e.cfg
	next := old
	if kind == "dashboard" {
		if port == old.DashboardPort {
			e.mu.Unlock()
			return nil
		}
		next.DashboardPort = port
	} else if kind == "heartbeat" {
		if port == old.ControlPort {
			e.mu.Unlock()
			return nil
		}
		next.ControlPort = port
	} else {
		e.mu.Unlock()
		return errors.New("use dashboard or heartbeat")
	}
	if port == old.ControlPort || port == old.DashboardPort || e.listeners[port] != nil {
		e.mu.Unlock()
		return errors.New("port belongs to SimpleFRP; configuration unchanged")
	}
	e.mu.Unlock()
	if kind == "dashboard" {
		if e.RebindDashboard == nil {
			return errors.New("dashboard is not running")
		}
		if err := e.RebindDashboard(port); err != nil {
			return err
		}
		if err := e.SaveConfig(next); err != nil {
			_ = e.RebindDashboard(old.DashboardPort)
			return err
		}
		e.mu.Lock()
		e.cfg = next
		e.mu.Unlock()
		return nil
	}
	pair, err := tls.X509KeyPair([]byte(old.Certificate), []byte(old.PrivateKey))
	if err != nil {
		return err
	}
	if !sysutil.IsPortAvailable(port) {
		return errors.New("heartbeat port is in use; configuration unchanged")
	}
	ln, err := tls.Listen("tcp", net.JoinHostPort(old.BindAddress, strconv.Itoa(port)), &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{pair}})
	if err != nil {
		return errors.New("heartbeat port is in use; configuration unchanged")
	}
	if err = e.SaveConfig(next); err != nil {
		ln.Close()
		return err
	}
	e.mu.Lock()
	e.cfg = next
	previous := append([]net.Listener(nil), e.controls...)
	e.controls = append(e.controls, ln)
	sessions := []*clientSession{}
	for _, s := range e.clients {
		sessions = append(sessions, s)
	}
	e.mu.Unlock()
	e.spawn(func() { e.acceptControl(ln) })
	for _, s := range sessions {
		_ = s.send(protocol.Message{Type: protocol.TypeMoved, Address: net.JoinHostPort(next.PublicIP, strconv.Itoa(port))})
	}
	e.spawn(func() {
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-e.done:
		}
		for _, oldLn := range previous {
			oldLn.Close()
		}
		e.mu.Lock()
		kept := e.controls[:0]
		for _, candidate := range e.controls {
			remove := false
			for _, oldLn := range previous {
				if candidate == oldLn {
					remove = true
					break
				}
			}
			if !remove {
				kept = append(kept, candidate)
			}
		}
		e.controls = kept
		e.mu.Unlock()
	})
	return nil
}
func (e *ServerEngine) handleData(conn net.Conn, msg protocol.Message) {
	e.mu.Lock()
	pending := e.pending[msg.RequestID]
	if pending == nil || pending.clientID != msg.ClientID || pending.tunnelID != msg.TunnelID {
		e.mu.Unlock()
		conn.Close()
		return
	}
	delete(e.pending, msg.RequestID)
	e.conns[conn] = msg.TunnelID
	e.mu.Unlock()
	pending.channel <- conn
}
func (e *ServerEngine) acceptPublic(ln net.Listener, port int) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		e.mu.Lock()
		var binding *tunnelBinding
		for _, b := range e.bindings {
			if b.record.PublicPort == port {
				binding = b
				break
			}
		}
		var session *clientSession
		if binding != nil {
			session = e.clients[binding.clientID]
		}
		if session == nil || len(e.pending) >= 256 || len(e.conns) >= 4096 {
			e.mu.Unlock()
			conn.Close()
			continue
		}
		record := binding.record
		e.conns[conn] = record.ID
		e.mu.Unlock()
		if !e.spawn(func() { e.bridge(session, conn, binding, record) }) {
			conn.Close()
			e.mu.Lock()
			delete(e.conns, conn)
			e.mu.Unlock()
		}
	}
}

type meteredConn struct {
	net.Conn
	total *atomic.Int64
}

func (c *meteredConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	c.total.Add(int64(n))
	return n, err
}
func (c *meteredConn) CloseWrite() error {
	if v, ok := c.Conn.(interface{ CloseWrite() error }); ok {
		return v.CloseWrite()
	}
	return c.Conn.Close()
}
func (e *ServerEngine) bridge(session *clientSession, public net.Conn, b *tunnelBinding, record config.Tunnel) {
	defer public.Close()
	id, err := crypto.RandomToken(16)
	if err != nil {
		return
	}
	ch := make(chan net.Conn, 1)
	e.mu.Lock()
	e.pending[id] = &pendingStream{session.id, record.ID, ch}
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		delete(e.pending, id)
		delete(e.conns, public)
		e.mu.Unlock()
		select {
		case c := <-ch:
			c.Close()
		default:
		}
	}()
	if session.send(protocol.Message{Type: protocol.TypeOpenStream, RequestID: id, TunnelID: record.ID, LocalPort: record.LocalPort, PublicPort: record.PublicPort}) != nil {
		return
	}
	timer := time.NewTimer(12 * time.Second)
	defer timer.Stop()
	select {
	case data := <-ch:
		defer func() { e.mu.Lock(); delete(e.conns, data); e.mu.Unlock() }()
		proxy(&meteredConn{public, &b.total}, &meteredConn{data, &b.total})
	case <-timer.C:
	case <-e.done:
	}
}
func (e *ServerEngine) Status(clientID string, admin bool) protocol.Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := protocol.Status{HeartbeatPort: e.cfg.ControlPort, Online: true, Tunnels: []protocol.TunnelStatus{}}
	if admin {
		s.DashboardPort = e.cfg.DashboardPort
	}
	for _, b := range e.bindings {
		if !admin && b.clientID != clientID {
			continue
		}
		state := "offline"
		speed := b.speed
		if e.clients[b.clientID] != nil {
			state = "connected"
		} else {
			speed = 0
		}
		s.Tunnels = append(s.Tunnels, protocol.TunnelStatus{ID: b.record.ID, LocalPort: b.record.LocalPort, PublicPort: b.record.PublicPort, Status: state, BytesPerSecond: speed, TotalBytes: b.total.Load()})
	}
	sort.Slice(s.Tunnels, func(i, j int) bool { return s.Tunnels[i].ID < s.Tunnels[j].ID })
	return s
}
func (e *ServerEngine) sample() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	previous := time.Now()
	for {
		select {
		case now := <-ticker.C:
			e.mu.Lock()
			for _, b := range e.bindings {
				total := b.total.Load()
				b.speed = float64(total-b.last) / now.Sub(previous).Seconds()
				b.last = total
			}
			_ = e.persistLocked()
			e.mu.Unlock()
			previous = now
		case <-e.done:
			return
		}
	}
}
func (e *ServerEngine) Close() {
	e.once.Do(func() {
		e.mutation.Lock()
		e.workerMu.Lock()
		e.closing = true
		e.workerMu.Unlock()
		close(e.done)
		e.mu.Lock()
		for _, ln := range e.controls {
			ln.Close()
		}
		for _, ln := range e.listeners {
			ln.Close()
		}
		for conn := range e.conns {
			conn.Close()
		}
		for _, s := range e.clients {
			s.conn.Close()
		}
		e.mu.Unlock()
		e.mutation.Unlock()
		e.workers.Wait()
		e.mu.Lock()
		_ = e.persistLocked()
		e.mu.Unlock()
	})
}
