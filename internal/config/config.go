package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application.
type Config struct {
	DBUrl  string
	DBNs   string
	DBDb   string
	DBUser string
	DBPass string
}

// New loads configuration from environment variables.
func New() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	cfg := &Config{
		DBUrl:  os.Getenv("SURREAL_URL"),
		DBUser: os.Getenv("SURREAL_USER"),
		DBPass: os.Getenv("SURREAL_PASS"),
		DBNs:   os.Getenv("SURREAL_NS"),
		DBDb:   os.Getenv("SURREAL_DB"),
	}

	if cfg.DBUrl == "" || cfg.DBNs == "" || cfg.DBDb == "" {
		log.Fatal("Required environment variables SURREAL_URL, SURREAL_NS, or SURREAL_DB are not set.")
	}

	return cfg
}
