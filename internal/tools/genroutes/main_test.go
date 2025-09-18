package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGenerateRoutes(t *testing.T) {
	// Test the version flag
	t.Run("version flag", func(t *testing.T) {
		cmd := exec.Command("go", "run", ".", "-version")
		cmd.Dir = "."
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to get version: %v\n%s", err, out)
		}
	})

	// Test the main functionality with a test modules directory
	t.Run("generate routes", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir := t.TempDir()
		testModulesDir := filepath.Join(tempDir, "testmodules")

		// Create a simple test module
		testModuleDir := filepath.Join(testModulesDir, "testmodule")
		if err := os.MkdirAll(testModuleDir, 0755); err != nil {
			t.Fatalf("Failed to create test module directory: %v", err)
		}

		// Create a simple routes file
		routesFile := filepath.Join(testModuleDir, "routes.go")
		if err := os.WriteFile(routesFile, []byte(`package testmodule

import "net/http"

func RegisterRoutes() []Route {
	return []Route{
		{
			Path:    "/test",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			Methods: []string{"GET"},
		},
	}
}`), 0644); err != nil {
			t.Fatalf("Failed to create test routes file: %v", err)
		}

		// Run the generator with the test directory
		cmd := exec.Command("go", "run", ".", "-modules", testModulesDir)
		cmd.Dir = "."
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to generate routes: %v\n%s", err, out)
		}

		// Verify the output file was created
		outputFile := filepath.Join(testModulesDir, "zz_routes_imports.go")
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Fatalf("Expected output file %s was not created", outputFile)
		}
	})
}
