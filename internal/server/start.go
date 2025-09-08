package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Start runs the HTTP server.
func (s *Server) Start(addr string) {
	go func() {
		if err := s.E.Start(addr); err != nil && err != http.ErrServerClosed {
			s.E.Logger.Fatalf("shutting down the server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server with a timeout.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s.DB.Close(ctx)
	if err := s.E.Shutdown(ctx); err != nil {
		s.E.Logger.Fatal(err)
	}
}
