package api

import (
	"context"
	"log"
	"net/http"
	"time"
)

// Server wraps an HTTP server with graceful shutdown.
type Server struct {
	srv *http.Server
}

// NewServer creates a Server on the given address with the registered handler.
func NewServer(addr string, handler *Handler) *Server {
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	var h http.Handler = mux
	h = requestIDMiddleware(h)
	h = loggingMiddleware(h)
	h = recoveryMiddleware(h)

	return &Server{
		srv: &http.Server{
			Addr:         addr,
			Handler:      h,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

// Start begins listening. It blocks until the server stops.
func (s *Server) Start() error {
	log.Printf("server listening on %s", s.srv.Addr)
	err := s.srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Shutdown gracefully drains in-flight requests within the given timeout.
func (s *Server) Shutdown(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}
