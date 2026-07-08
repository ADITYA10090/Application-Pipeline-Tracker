package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

type Server struct {
	engine *Engine
	queue  *Queue
	log    *slog.Logger
}

func NewServer(engine *Engine, queue *Queue, log *slog.Logger) *Server {
	return &Server{engine: engine, queue: queue, log: log}
}

func (s *Server) Routes() *http.ServeMux {
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

// handleTrigger kicks off a scrape run in the background and returns 202 so the
// caller (the Node gateway) is not blocked for the duration of the scrape.
func (s *Server) handleTrigger(w http.ResponseWriter, r *http.Request) {
	go s.engine.Run(context.Background())
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "scrape started"})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	depth, err := s.queue.Depth(r.Context())
	if err != nil {
		depth = -1
	}
	m := s.engine.metrics
	writeJSON(w, http.StatusOK, map[string]any{
		"jobs_fetched":    m.Fetched.Load(),
		"jobs_inserted":   m.Inserted.Load(),
		"duplicates":      m.Duplicates.Load(),
		"errors":          m.Errors.Load(),
		"queue_depth":     depth,
		"last_run_unix":   m.LastRunAt.Load(),
		"running":         s.engine.running.Load(),
		"configured_srcs": len(s.engine.cfg.Sources),
	})
}
