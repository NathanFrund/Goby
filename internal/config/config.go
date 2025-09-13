package config

import (
	"log"
	"net"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application.
type Config struct {
	ServerAddr    string
	DBUrl         string
	DBNs          string
	DBDb          string
	DBUser        string
	DBPass        string
	EmailProvider string
	EmailAPIKey   string
	EmailSender   string
	AppBaseURL    string
	SessionSecret string
}

// New loads configuration from environment variables.
func New() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	cfg := &Config{
		ServerAddr:    os.Getenv("SERVER_ADDR"),
		DBUrl:         os.Getenv("SURREAL_URL"),
		DBUser:        os.Getenv("SURREAL_USER"),
		DBPass:        os.Getenv("SURREAL_PASS"),
		DBNs:          os.Getenv("SURREAL_NS"),
		DBDb:          os.Getenv("SURREAL_DB"),
		EmailProvider: os.Getenv("EMAIL_PROVIDER"),
		EmailAPIKey:   os.Getenv("EMAIL_API_KEY"),
		EmailSender:   os.Getenv("EMAIL_SENDER"),
		AppBaseURL:    os.Getenv("APP_BASE_URL"),
		SessionSecret: os.Getenv("SESSION_SECRET"),
	}

	if cfg.ServerAddr == "" {
		cfg.ServerAddr = ":8080"
	}

	if cfg.DBUrl == "" || cfg.DBNs == "" || cfg.DBDb == "" {
		log.Fatal("Required environment variables SURREAL_URL, SURREAL_NS, or SURREAL_DB are not set.")
	}

	if cfg.SessionSecret == "" {
		log.Fatal("Required environment variable SESSION_SECRET is not set.")
	}

	// Set sensible defaults for development
	if cfg.EmailProvider == "" {
		cfg.EmailProvider = "log" // Default to logging emails to the console
		cfg.EmailSender = "noreply@localhost"
	}

	// Default the base URL for local development if not set.
	if cfg.AppBaseURL == "" {
		// Derive from ServerAddr to avoid hardcoding.
		host, port, err := net.SplitHostPort(cfg.ServerAddr)
		if err != nil {
			// Handle case where address is just a port like ":8080"
			if strings.HasPrefix(cfg.ServerAddr, ":") {
				host = "localhost"
				port = strings.TrimPrefix(cfg.ServerAddr, ":")
			}
		}
		if host == "" {
			host = "localhost"
		}
		cfg.AppBaseURL = "http://" + net.JoinHostPort(host, port)
	}

	return cfg
}
