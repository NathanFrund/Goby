package script

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockConfig implements config.Provider for testing
type MockConfig struct{}

func (m *MockConfig) GetServerAddr() string                 { return ":8080" }
func (m *MockConfig) GetDBURL() string                      { return "" }
func (m *MockConfig) GetDBNs() string                       { return "" }
func (m *MockConfig) GetDBDb() string                       { return "" }
func (m *MockConfig) GetDBUser() string                     { return "" }
func (m *MockConfig) GetDBPass() string                     { return "" }
func (m *MockConfig) GetEmailProvider() string              { return "mock" }
func (m *MockConfig) GetEmailAPIKey() string                { return "" }
func (m *MockConfig) GetEmailSender() string                { return "test@example.com" }
func (m *MockConfig) GetAppBaseURL() string                  { return "http://localhost:8080" }
func (m *MockConfig) GetSessionSecret() string              { return "test-secret" }
func (m *MockConfig) GetDBQueryTimeout() time.Duration      { return 5 * time.Second }
func (m *MockConfig) GetDBExecuteTimeout() time.Duration    { return 10 * time.Second }
func (m *MockConfig) GetStorageBackend() string             { return "mem" }
func (m *MockConfig) GetStoragePath() string                { return "/tmp" }
func (m *MockConfig) GetMaxFileSize() int64                 { return 1024 * 1024 }
func (m *MockConfig) GetAllowedMimeTypes() []string         { return []string{"text/plain"} }
func (m *MockConfig) GetModuleConfig(moduleName string) (interface{}, bool) { return nil, false }

func TestEngine_Initialize(t *testing.T) {
	cfg := &MockConfig{}
	engine := NewEngine(Dependencies{Config: cfg})

	err := engine.Initialize(context.Background())
	assert.NoError(t, err)
}

func TestEngine_RegisterEmbeddedProvider(t *testing.T) {
	cfg := &MockConfig{}
	engine := NewEngine(Dependencies{Config: cfg})

	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"test_script": "result := 42",
		},
	}

	engine.RegisterEmbeddedProvider(provider)
	err := engine.Initialize(context.Background())
	require.NoError(t, err)

	// Test that we can get the script
	script, err := engine.GetScript("test_module", "test_script")
	require.NoError(t, err)
	assert.Equal(t, "test_module", script.ModuleName)
	assert.Equal(t, "test_script", script.Name)
	assert.Equal(t, "result := 42", script.Content)
}

func TestEngine_Execute(t *testing.T) {
	cfg := &MockConfig{}
	engine := NewEngine(Dependencies{Config: cfg})

	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"calculator": "result := a + b",
		},
	}

	engine.RegisterEmbeddedProvider(provider)
	err := engine.Initialize(context.Background())
	require.NoError(t, err)

	// Execute the script
	req := ExecutionRequest{
		ModuleName: "test_module",
		ScriptName: "calculator",
		Input: &ScriptInput{
			Context: map[string]interface{}{
				"a": 10,
				"b": 20,
			},
		},
	}

	output, err := engine.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 30, output.Result)
	assert.True(t, output.Metrics.Success)
}

func TestEngine_ExtractDefaultScripts(t *testing.T) {
	cfg := &MockConfig{}
	engine := NewEngine(Dependencies{Config: cfg})

	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"script1": "result := 1",
			"script2": "result := 2",
		},
	}

	engine.RegisterEmbeddedProvider(provider)
	err := engine.Initialize(context.Background())
	require.NoError(t, err)

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "script_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Extract scripts
	err = engine.ExtractDefaultScripts(tempDir)
	require.NoError(t, err)

	// Verify files were created
	script1Path := filepath.Join(tempDir, "test_module", "script1.tengo")
	script2Path := filepath.Join(tempDir, "test_module", "script2.tengo")

	assert.FileExists(t, script1Path)
	assert.FileExists(t, script2Path)

	// Verify content
	content1, err := os.ReadFile(script1Path)
	require.NoError(t, err)
	assert.Equal(t, "result := 1", string(content1))

	content2, err := os.ReadFile(script2Path)
	require.NoError(t, err)
	assert.Equal(t, "result := 2", string(content2))
}

func TestEngine_GetSupportedLanguages(t *testing.T) {
	cfg := &MockConfig{}
	engine := NewEngine(Dependencies{Config: cfg})

	languages := engine.GetSupportedLanguages()
	assert.Contains(t, languages, LanguageTengo)
	// Zygomys will be added in a later task
}

func TestEngine_Shutdown(t *testing.T) {
	cfg := &MockConfig{}
	engine := NewEngine(Dependencies{Config: cfg})

	err := engine.Shutdown(context.Background())
	assert.NoError(t, err)
}