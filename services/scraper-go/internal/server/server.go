package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"trackfolio/scraper-go/internal/pipeline"
)

type Server struct {
	pl  *pipeline.Pipeline
	log *slog.Logger
}

func New(pl *pipeline.Pipeline, log *slog.Logger) *Server {
	return &Server{pl: pl, log: log}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /trigger", s.handleTrigger)
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /metrics", s.handleMetrics)
	return mux
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleTrigger(w http.ResponseWriter, r *http.Request) {
	accepted := s.pl.TriggerAsync(r.Context())
	if !accepted {
		writeJSON(w, http.StatusConflict, map[string]string{"status": "already_running"})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	writeJSON(w, http.StatusOK, s.pl.Snapshot(ctx))
}
