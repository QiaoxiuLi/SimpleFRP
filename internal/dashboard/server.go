package dashboard

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/simplefrp/simplefrp/internal/protocol"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

//go:embed static/*
var staticFS embed.FS

type Server struct {
	host   string
	status func() protocol.Status
	mu     sync.Mutex
	http   *http.Server
}

func New(host string, status func() protocol.Status) *Server {
	if host == "" {
		host = "127.0.0.1"
	}
	return &Server{host: host, status: status}
}
func (s *Server) Start(port int) error {
	ln, err := net.Listen("tcp", net.JoinHostPort(s.host, strconv.Itoa(port)))
	if err != nil {
		return fmt.Errorf("dashboard port unavailable; configuration unchanged: %w", err)
	}
	mux := http.NewServeMux()
	static, _ := fs.Sub(staticFS, "static")
	mux.Handle("/", http.FileServer(http.FS(static)))
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(s.status())
	})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Read-only status page", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")
		mux.ServeHTTP(w, r)
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
	s.mu.Lock()
	old := s.http
	s.http = server
	s.mu.Unlock()
	go server.Serve(ln)
	if old != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		old.Shutdown(ctx)
	}
	return nil
}
func (s *Server) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.http != nil {
		s.http.Close()
	}
}
