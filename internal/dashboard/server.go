package dashboard

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/storage"
)

//go:embed static/*
var staticFS embed.FS

type Server struct {
	cfg   config.ServerConfig
	store *storage.Store
	hub   *Hub
}

func New(cfg config.ServerConfig, store *storage.Store) *Server {
	return &Server{cfg: cfg, store: store, hub: NewHub()}
}

func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	s.routes(mux)
	host := s.cfg.DashboardBindAddress
	if host == "" {
		host = "127.0.0.1"
	}
	addr := fmt.Sprintf("%s:%d", host, s.cfg.DashboardPort)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) routes(mux *http.ServeMux) {
	static, _ := fs.Sub(staticFS, "static")
	mux.Handle("/", http.FileServer(http.FS(static)))
	mux.HandleFunc("/api/status", s.status)
	mux.HandleFunc("/api/traffic", s.traffic)
	mux.HandleFunc("/api/connections", s.connections)
	mux.HandleFunc("/api/tunnels", s.tunnels)
	mux.HandleFunc("/events", s.events)
}
