package script

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_ExtractDefaultScripts_Detailed(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "script_extraction_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create engine with mock config
	cfg := &MockConfig{}
	engine := NewEngine(Dependencies{Config: cfg})

	// Register embedded script provider
	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"calculator":    "result := a + b",
			"processor":     "result := input * 2",
			"simple_script": "x := 42",
		},
	}

	engine.RegisterEmbeddedProvider(provider)

	// Initialize engine to load scripts
	err = engine.Initialize(context.Background())
	require.NoError(t, err)

	// Extract scripts
	extractDir := filepath.Join(tempDir, "extracted_scripts")
	err = engine.ExtractDefaultScripts(extractDir)
	require.NoError(t, err)

	// Verify directory structure was created
	moduleDir := filepath.Join(extractDir, "test_module")
	assert.DirExists(t, moduleDir)

	// Verify script files were created with correct extensions
	calculatorPath := filepath.Join(moduleDir, "calculator.tengo")
	processorPath := filepath.Join(moduleDir, "processor.tengo")
	simplePath := filepath.Join(moduleDir, "simple_script.tengo")

	assert.FileExists(t, calculatorPath)
	assert.FileExists(t, processorPath)
	assert.FileExists(t, simplePath)

	// Verify file contents
	calculatorContent, err := os.ReadFile(calculatorPath)
	require.NoError(t, err)
	assert.Equal(t, "result := a + b", string(calculatorContent))

	processorContent, err := os.ReadFile(processorPath)
	require.NoError(t, err)
	assert.Equal(t, "result := input * 2", string(processorContent))

	simpleContent, err := os.ReadFile(simplePath)
	require.NoError(t, err)
	assert.Equal(t, "x := 42", string(simpleContent))
}

func TestEngine_ExtractDefaultScripts_SkipExisting(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "script_extraction_skip_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create engine with mock config
	cfg := &MockConfig{}
	engine := NewEngine(Dependencies{Config: cfg})

	// Register embedded script provider
	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"existing_script": "result := 1",
		},
	}

	engine.RegisterEmbeddedProvider(provider)

	// Initialize engine to load scripts
	err = engine.Initialize(context.Background())
	require.NoError(t, err)

	// Create extraction directory and pre-existing file
	extractDir := filepath.Join(tempDir, "extracted_scripts")
	moduleDir := filepath.Join(extractDir, "test_module")
	err = os.MkdirAll(moduleDir, 0755)
	require.NoError(t, err)

	existingFilePath := filepath.Join(moduleDir, "existing_script.tengo")
	err = os.WriteFile(existingFilePath, []byte("result := 999"), 0644)
	require.NoError(t, err)

	// Extract scripts
	err = engine.ExtractDefaultScripts(extractDir)
	require.NoError(t, err)

	// Verify existing file was not overwritten
	content, err := os.ReadFile(existingFilePath)
	require.NoError(t, err)
	assert.Equal(t, "result := 999", string(content)) // Should still be the original content
}
