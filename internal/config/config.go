package config

import (
	"log"
	"net"
	"os"
	"strconv"
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
	GetStorageBackend() string
	GetStoragePath() string
	GetMaxUploadSize() int64
	GetAllowedMimeTypes() []string
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
	StorageBackend   string
	StoragePath      string
	MaxUploadSizeMB  int64
	AllowedMimeTypes string
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
		StorageBackend:   os.Getenv("STORAGE_BACKEND"),
		StoragePath:      os.Getenv("STORAGE_PATH"),
		MaxUploadSizeMB:  getInt64Env("STORAGE_MAX_UPLOAD_MB", 5),
		AllowedMimeTypes: os.Getenv("STORAGE_ALLOWED_MIME_TYPES"),
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

	// Set sensible defaults for storage
	if cfg.StorageBackend == "" {
		cfg.StorageBackend = "os" // 'os' for OsFs, 'mem' for MemMapFs
	}

	if cfg.StoragePath == "" {
		cfg.StoragePath = "tmp/uploads" // Default local storage path
	}

	return cfg
}

// getInt64Env is a helper to parse an int64 from env with a default.
func getInt64Env(key string, fallback int64) int64 {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return int64(i)
		}
	}
	return fallback
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

// GetStorageBackend returns the configured storage backend ('os' or 'mem').
func (c *Config) GetStorageBackend() string {
	return c.StorageBackend
}

// GetStoragePath returns the root path for the file storage.
func (c *Config) GetStoragePath() string {
	return c.StoragePath
}

// GetMaxUploadSize returns the maximum file upload size in bytes.
func (c *Config) GetMaxUploadSize() int64 {
	return c.MaxUploadSizeMB * 1024 * 1024
}

// GetAllowedMimeTypes returns a list of allowed MIME types for uploads.
func (c *Config) GetAllowedMimeTypes() []string {
	if c.AllowedMimeTypes == "" {
		// Return a default list or an empty list to allow all types
		return []string{"image/jpeg", "image/png", "application/pdf"}
	}
	return strings.Split(c.AllowedMimeTypes, ",")
}

// GetModuleConfig retrieves the configuration for a specific module.
// Returns the config and a boolean indicating if it was found.
func (c *Config) GetModuleConfig(moduleName string) (interface{}, bool) {
	configMutex.RLock()
	defer configMutex.RUnlock()

	cfg, exists := c.moduleConfigs[moduleName]
	return cfg, exists
}
