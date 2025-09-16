package testutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/logging"
)

// ConfigForTests loads the .env.test file and returns a valid config.Provider.
// This is the definitive way to get configuration for integration tests.
func ConfigForTests(t *testing.T) config.Provider {
	t.Helper()

	// 1. Find project root by looking for go.mod to reliably locate .env.test
	path, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
			break
		}
		if path == filepath.Dir(path) {
			t.Fatalf("could not find project root with go.mod")
		}
		path = filepath.Dir(path)
	}

	// 2. Manually read the .env.test file.
	env, err := godotenv.Read(filepath.Join(path, ".env.test"))
	if err != nil {
		t.Fatalf("failed to load .env.test file: %v", err)
	}

	// 3. Use t.Setenv to set the environment variables for this test.
	// This is the idiomatic and safest way to handle test environments.
	for key, value := range env {
		t.Setenv(key, value)
	}

	logging.New()

	// 4. Now that the environment is set, create the config.
	return config.New()
}
