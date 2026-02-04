package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server manages HTTP server with all routes and middleware
type Server struct {
	http    *http.Server
	codex   *CodexClient
	sessmgr *SessionManager
	metrics *MetricsCollector
}

// NewServer creates a new server with all routes mounted
func NewServer(cfg *Config) (*Server, error) {
	codex := NewCodexClient(cfg.Path, cfg.Timeout)
	sessmgr := NewSessionManager(1 * time.Hour)
	metrics := NewMetricsCollector()

	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", handleHealth(cfg))

	// Prometheus metrics
	mux.Handle("/metrics", promhttp.Handler())

	// Ollama-compatible API routes
	mux.HandleFunc("/api/tags", HandleOllamaTags)
	mux.HandleFunc("/api/generate", HandleOllamaGenerate(codex))
	mux.HandleFunc("/api/chat", HandleOllamaChat(codex))

	// OpenAI-compatible API routes
	mux.HandleFunc("/v1/models", HandleOpenAIModels)
	mux.HandleFunc("/v1/chat/completions", HandleOpenAIChat(codex))

	// Anthropic-compatible API routes
	anthropic := NewAnthropicHandler(codex, metrics)
	mux.HandleFunc("/v1/messages", anthropic.HandleMessages)

	// WebSocket session endpoint
	wsHandler := NewWSHandler(sessmgr, codex)
	mux.HandleFunc("/ws/session", wsHandler.HandleSession)

	// Session management endpoints
	mux.HandleFunc("/v1/sessions", handleSessions(sessmgr))

	// Apply JWT auth middleware
	handler := JWTMiddleware(cfg.JWTSecret)(mux)

	srv := &Server{
		http: &http.Server{
			Addr:         cfg.Address,
			Handler:      handler,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: cfg.Timeout + 10*time.Second,
			IdleTimeout:  120 * time.Second,
		},
		codex:   codex,
		sessmgr: sessmgr,
		metrics: metrics,
	}

	return srv, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	return s.http.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.sessmgr.Close()
	return s.http.Shutdown(ctx)
}

// handleHealth returns server health status
func handleHealth(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, &APIError{
				Code:    ErrInvalidRequest,
				Message: "method not allowed",
				Status:  http.StatusMethodNotAllowed,
			})
			return
		}

		resp := map[string]string{
			"status": "ok",
			"claude": cfg.Path,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// handleSessions returns list of sessions
func handleSessions(sessmgr *SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, &APIError{
				Code:    ErrInvalidRequest,
				Message: "method not allowed",
				Status:  http.StatusMethodNotAllowed,
			})
			return
		}

		ids := sessmgr.List()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sessions": ids,
			"count":    len(ids),
		})
	}
}
