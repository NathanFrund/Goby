package fullchat

import (
	"os"
)

// Config holds the configuration for the full-chat module.
type Config struct {
	SurrealNS string
	SurrealDB string
}

// NewConfig creates a new Config instance with values from environment variables.
func NewConfig() *Config {
	return &Config{
		SurrealNS: os.Getenv("FULLCHAT_SURREAL_NS"),
		SurrealDB: os.Getenv("FULLCHAT_SURREAL_DB"),
	}
}
