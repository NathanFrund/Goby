package config

import (
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// ModuleConfigLoader is a function that loads configuration for a specific module.
type ModuleConfigLoader func() interface{}

var (
	moduleConfigLoaders = make(map[string]ModuleConfigLoader)
	_                   = make(map[string]interface{}) // moduleConfigs is used in Config struct
	configMutex         sync.RWMutex
)

// RegisterModuleConfig registers a configuration loader for a module.
// This should be called during package initialization.
func RegisterModuleConfig(moduleName string, loader ModuleConfigLoader) {
	configMutex.Lock()
	defer configMutex.Unlock()
	moduleConfigLoaders[moduleName] = loader
}

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
	GetDBQueryTimeout() time.Duration
	GetDBExecuteTimeout() time.Duration
	// GetModuleConfig retrieves the configuration for a specific module.
	// Returns the config and a boolean indicating if it was found.
	GetModuleConfig(moduleName string) (interface{}, bool)
}

// Config holds all configuration for the application.
type Config struct {
	ServerAddr       string
	DBUrl            string
	DBNs             string
	DBDb             string
	DBUser           string
	DBPass           string
	DBQueryTimeout   time.Duration
	DBExecuteTimeout time.Duration
	EmailProvider    string
	EmailAPIKey      string
	EmailSender      string
	AppBaseURL       string
	SessionSecret    string
	// ModuleConfigs holds configuration for registered modules.
	moduleConfigs map[string]interface{}
}

// New loads configuration from environment variables.
func New() Provider {
	queryTimeout, err := time.ParseDuration(os.Getenv("DB_QUERY_TIMEOUT"))
	if err != nil {
		queryTimeout = 5 * time.Second // Default value
	}

	executeTimeout, err := time.ParseDuration(os.Getenv("DB_EXECUTE_TIMEOUT"))
	if err != nil {
		executeTimeout = 10 * time.Second // Default value
	}

	cfg := &Config{
		ServerAddr:       os.Getenv("SERVER_ADDR"),
		DBUrl:            os.Getenv("SURREAL_URL"),
		DBUser:           os.Getenv("SURREAL_USER"),
		DBPass:           os.Getenv("SURREAL_PASS"),
		DBNs:             os.Getenv("SURREAL_NS"),
		DBDb:             os.Getenv("SURREAL_DB"),
		DBQueryTimeout:   queryTimeout,
		DBExecuteTimeout: executeTimeout,
		EmailProvider:    os.Getenv("EMAIL_PROVIDER"),
		EmailAPIKey:      os.Getenv("EMAIL_API_KEY"),
		EmailSender:      os.Getenv("EMAIL_SENDER"),
		AppBaseURL:       os.Getenv("APP_BASE_URL"),
		SessionSecret:    os.Getenv("SESSION_SECRET"),
		moduleConfigs:    make(map[string]interface{}),
	}

	// Load all registered module configurations
	configMutex.RLock()
	defer configMutex.RUnlock()

	for moduleName, loader := range moduleConfigLoaders {
		cfg.moduleConfigs[moduleName] = loader()
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

// GetDBQueryTimeout returns the default timeout for database read queries.
func (c *Config) GetDBQueryTimeout() time.Duration {
	return c.DBQueryTimeout
}

// GetDBExecuteTimeout returns the default timeout for database write operations.
func (c *Config) GetDBExecuteTimeout() time.Duration {
	return c.DBExecuteTimeout
}

// GetModuleConfig retrieves the configuration for a specific module.
// Returns the config and a boolean indicating if it was found.
func (c *Config) GetModuleConfig(moduleName string) (interface{}, bool) {
	configMutex.RLock()
	defer configMutex.RUnlock()

	cfg, exists := c.moduleConfigs[moduleName]
	return cfg, exists
}
