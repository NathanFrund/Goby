package config

import (
	"log"
	"net"
	"os"
	"strings"
)

// Provider defines the interface for accessing configuration values.
// This allows for dependency injection and easier testing.
type Provider interface {
	GetServerAddr() string
	GetDBUrl() string
	GetDBNs() string
	GetDBDb() string
	GetDBUser() string
	GetDBPass() string
	GetEmailProvider() string
	GetEmailAPIKey() string
	GetEmailSender() string
	GetAppBaseURL() string
	GetSessionSecret() string
}

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
func New() Provider {
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
		// It's better for the application's entry point (main.go) to handle this.
		log.Println("WARNING: One or more required database environment variables are not set (SURREAL_URL, SURREAL_NS, SURREAL_DB).")
	}

	if cfg.SessionSecret == "" {
		// Same as above, let the caller decide if this is a fatal error.
		log.Println("WARNING: Required environment variable SESSION_SECRET is not set.")
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

// GetServerAddr returns the server address.
func (c *Config) GetServerAddr() string {
	return c.ServerAddr
}

// GetDBUrl returns the database URL.
func (c *Config) GetDBUrl() string {
	return c.DBUrl
}

// GetDBNs returns the database namespace.
func (c *Config) GetDBNs() string {
	return c.DBNs
}

// GetDBDb returns the database name.
func (c *Config) GetDBDb() string {
	return c.DBDb
}

// GetDBUser returns the database user.
func (c *Config) GetDBUser() string {
	return c.DBUser
}

// GetDBPass returns the database password.
func (c *Config) GetDBPass() string {
	return c.DBPass
}

// GetEmailProvider returns the email provider.
func (c *Config) GetEmailProvider() string {
	return c.EmailProvider
}

// GetEmailAPIKey returns the email API key.
func (c *Config) GetEmailAPIKey() string {
	return c.EmailAPIKey
}

// GetEmailSender returns the email sender address.
func (c *Config) GetEmailSender() string {
	return c.EmailSender
}

// GetAppBaseURL returns the application's base URL.
func (c *Config) GetAppBaseURL() string {
	return c.AppBaseURL
}

// GetSessionSecret returns the session secret key.
func (c *Config) GetSessionSecret() string {
	return c.SessionSecret
}
