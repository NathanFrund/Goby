package server

import (
	"os"
	"os/signal"
	"syscall"
)

// waitForShutdown blocks until an interrupt or terminate signal is received.
func waitForShutdown() {
	// Wait for interrupt signal to gracefully shut down the server.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
}
