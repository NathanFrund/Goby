package pubsub

import (
	"os"
	"strconv"
)

// LoadTracingConfigFromEnv loads tracing configuration from environment variables
func LoadTracingConfigFromEnv() TracingConfig {
	config := DefaultTracingConfig()

	// Check if tracing is enabled
	if enabledStr := os.Getenv("PUBSUB_TRACING_ENABLED"); enabledStr != "" {
		if enabled, err := strconv.ParseBool(enabledStr); err == nil {
			config.Enabled = enabled
		}
	}

	// Service name
	if serviceName := os.Getenv("PUBSUB_TRACING_SERVICE_NAME"); serviceName != "" {
		config.ServiceName = serviceName
	}

	// Zipkin URL
	if zipkinURL := os.Getenv("PUBSUB_TRACING_ZIPKIN_URL"); zipkinURL != "" {
		config.ZipkinURL = zipkinURL
	}

	return config
}
