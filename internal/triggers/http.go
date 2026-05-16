package triggers

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type HTTPServer struct {
	srv    *http.Server
	engine *Engine
	mu     sync.Mutex
}

func NewHTTPServer(addr string, engine *Engine) *HTTPServer {
	mux := http.NewServeMux()

	s := &HTTPServer{
		engine: engine,
		srv: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}

	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/", s.triggerHandler)

	return s
}

func (s *HTTPServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *HTTPServer) triggerHandler(w http.ResponseWriter, r *http.Request) {
	if s.engine == nil {
		http.Error(w, "trigger engine unavailable", http.StatusServiceUnavailable)
		return
	}
	s.engine.HandleHTTP(r.Context(), w, r)
}

func (s *HTTPServer) Start() error {
	if s == nil || s.srv == nil {
		return fmt.Errorf("http server is nil")
	}
	err := s.srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	if s == nil || s.srv == nil {
		return nil
	}
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.srv.Shutdown(shutdownCtx)
}