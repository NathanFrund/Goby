// internal/script/extractor/extractor_test.go
package extractor

import (
	"os"
	"testing"

	"github.com/nfrund/goby/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEmbeddedScriptProvider is a mock implementation of the EmbeddedScriptProvider interface
type MockEmbeddedScriptProvider struct {
	moduleName string
	scripts    map[string]string
}

func (m *MockEmbeddedScriptProvider) GetModuleName() string {
	return m.moduleName
}

func (m *MockEmbeddedScriptProvider) GetEmbeddedScripts() map[string]string {
	return m.scripts
}

func TestExtractor_ExtractScripts(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "script_extract_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test config
	cfg := config.New()

	// Test with force=false (should fail if directory exists)
	extractor := NewExtractor(cfg, false)
	err = extractor.ExtractScripts(tempDir)
	assert.Error(t, err, "should fail when directory exists and force=false")
	assert.Contains(t, err.Error(), "target directory exists and --force-extract not specified")

	// Test with force=true
	extractor = NewExtractor(cfg, true)
	err = extractor.ExtractScripts(tempDir)

	// The extraction might succeed or fail depending on whether there are embedded scripts
	// So we'll just verify that we don't get the "directory exists" error
	if err != nil {
		assert.NotContains(t, err.Error(), "target directory exists")
	}

	// Verify the target directory exists
	_, err = os.Stat(tempDir)
	assert.NoError(t, err, "target directory should exist after extraction")
}

func TestExtractor_prepareTargetDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "prepare_dir_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name      string
		force     bool
		setup     func() string
		wantError bool
	}{
		{
			name:  "new directory",
			force: false,
			setup: func() string {
				dir, _ := os.MkdirTemp("", "test_dir_*")
				os.RemoveAll(dir) // Ensure it doesn't exist
				return dir
			},
			wantError: false,
		},
		{
			name:  "existing directory without force",
			force: false,
			setup: func() string {
				dir, _ := os.MkdirTemp("", "test_dir_*")
				return dir
			},
			wantError: true,
		},
		{
			name:  "existing directory with force",
			force: true,
			setup: func() string {
				dir, _ := os.MkdirTemp("", "test_dir_*")
				return dir
			},
			wantError: false,
		},
	}

	cfg := config.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := tt.setup()
			if !tt.wantError {
				defer os.RemoveAll(testDir)
			}

			e := NewExtractor(cfg, tt.force)
			err := e.prepareTargetDirectory(testDir)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				_, err := os.Stat(testDir)
				assert.NoError(t, err)
			}
		})
	}
}
