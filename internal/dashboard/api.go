package dashboard

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func (s *Server) writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, map[string]any{
		"clients":        0,
		"active_tunnels": len(s.store.Tunnels()),
		"traffic":        s.store.Traffic("30d"),
	})
}

func (s *Server) traffic(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, s.store.Traffic(r.URL.Query().Get("range")))
}

func (s *Server) connections(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 100
	}
	s.writeJSON(w, s.store.Connections(limit))
}

func (s *Server) tunnels(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, s.store.Tunnels())
}
