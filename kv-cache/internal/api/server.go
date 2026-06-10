package api

import (
	"context"
	"net/http"
	"time"
)

// Server wraps an http.Server with middleware and graceful shutdown.
type Server struct {
	srv *http.Server
}

// NewServer creates an HTTP server with the handler's routes and middleware.
func NewServer(addr string, h *Handler) *Server {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	var handler http.Handler = mux
	handler = loggingMiddleware(handler)
	handler = requestIDMiddleware(handler)
	handler = recoveryMiddleware(handler)

	return &Server{
		srv: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

// Start begins listening. It blocks until the server is shut down.
func (s *Server) Start() error {
	return s.srv.ListenAndServe()
}

// Shutdown gracefully drains the server within the given timeout.
func (s *Server) Shutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.srv.Shutdown(ctx)
}
