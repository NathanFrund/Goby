package logging

import (
	"log/slog"
	"os"
)

// New initializes a new slog logger and sets it as the default.
// It reads the LOG_FORMAT environment variable to determine the output format.
// Defaults to "text" for development, can be set to "json" for production.
func New() {
	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "" {
		logFormat = "text" // Default to text for development
	}

	var handler slog.Handler
	switch logFormat {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug, // Or read from env
		})
	default:
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelDebug, // Or read from env
			AddSource: true,            // Adds source file and line number
		})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
