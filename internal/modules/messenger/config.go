package messenger

import (
	"os"
)

// Config holds the configuration for the messenger module
type Config struct {
	SurrealNS string
	SurrealDB string
}

// GetConfig loads and returns the configuration for the messenger module
func GetConfig() (*Config, error) {
	return &Config{
		SurrealNS: os.Getenv("MESSENGER_SURREAL_NS"),
		SurrealDB: os.Getenv("MESSENGER_SURREAL_DB"),
	}, nil
}
