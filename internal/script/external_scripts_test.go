package script

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_LoadExternalScripts(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "script_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory for test
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create scripts directory structure
	err = os.MkdirAll("scripts/test_module", 0755)
	require.NoError(t, err)

	// Create external script files
	testScripts := map[string]string{
		"scripts/test_module/calculator.tengo":  "result := a + b + 10",
		"scripts/test_module/processor.zygomys": "(defn process [x] (* x 3))",
		"scripts/test_module/simple_script":     "x := 100",
	}

	for path, content := range testScripts {
		err = os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create registry and load external scripts
	registry := NewRegistry()
	err = registry.LoadExternalScripts()
	require.NoError(t, err)

	// Test that scripts were loaded
	scripts := registry.ListScripts()
	assert.Contains(t, scripts, "test_module")
	assert.Len(t, scripts["test_module"], 3)
	assert.Contains(t, scripts["test_module"], "calculator")
	assert.Contains(t, scripts["test_module"], "processor")
	assert.Contains(t, scripts["test_module"], "simple_script")

	// Test getting specific scripts
	calculator, err := registry.GetScript("test_module", "calculator")
	require.NoError(t, err)
	assert.Equal(t, "test_module", calculator.ModuleName)
	assert.Equal(t, "calculator", calculator.Name)
	assert.Equal(t, LanguageTengo, calculator.Language)
	assert.Equal(t, "result := a + b + 10", calculator.Content)
	assert.Equal(t, SourceExternal, calculator.Source)

	processor, err := registry.GetScript("test_module", "processor")
	require.NoError(t, err)
	assert.Equal(t, LanguageZygomys, processor.Language)
	assert.Equal(t, "(defn process [x] (* x 3))", processor.Content)
}

func TestRegistry_ExternalScriptPriority(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "script_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory for test
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create scripts directory
	err = os.MkdirAll("scripts/test_module", 0755)
	require.NoError(t, err)

	// Create external script that will override embedded
	err = os.WriteFile("scripts/test_module/test_script.tengo", []byte("result := 999"), 0644)
	require.NoError(t, err)

	// Create registry with embedded script
	registry := NewRegistry()

	// Register embedded script provider
	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"test_script": "result := 42",
		},
	}
	registry.RegisterEmbeddedProvider(provider)

	// Load all scripts
	err = registry.LoadScripts()
	require.NoError(t, err)

	// External script should override embedded
	script, err := registry.GetScript("test_module", "test_script")
	require.NoError(t, err)
	assert.Equal(t, "result := 999", script.Content)
	assert.Equal(t, SourceExternal, script.Source)
	assert.Equal(t, LanguageTengo, script.Language)
}

func TestRegistry_ReloadScript(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "script_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory for test
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create scripts directory
	err = os.MkdirAll("scripts/test_module", 0755)
	require.NoError(t, err)

	// Create initial external script
	scriptPath := "scripts/test_module/dynamic_script.tengo"
	err = os.WriteFile(scriptPath, []byte("result := 1"), 0644)
	require.NoError(t, err)

	// Create registry and load scripts
	registry := NewRegistry()
	err = registry.LoadExternalScripts()
	require.NoError(t, err)

	// Get initial script
	script, err := registry.GetScript("test_module", "dynamic_script")
	require.NoError(t, err)
	assert.Equal(t, "result := 1", script.Content)

	// Update the external script file
	err = os.WriteFile(scriptPath, []byte("result := 2"), 0644)
	require.NoError(t, err)

	// Reload the script
	err = registry.ReloadScript("test_module", "dynamic_script")
	require.NoError(t, err)

	// Get updated script
	script, err = registry.GetScript("test_module", "dynamic_script")
	require.NoError(t, err)
	assert.Equal(t, "result := 2", script.Content)
}

func TestRegistry_CrossLanguageReplacement(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "script_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory for test
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create scripts directory
	err = os.MkdirAll("scripts/test_module", 0755)
	require.NoError(t, err)

	// Create external Zygomys script to replace Tengo embedded script
	err = os.WriteFile("scripts/test_module/calculator.zygomys", []byte("(+ a b)"), 0644)
	require.NoError(t, err)

	// Create registry with embedded Tengo script
	registry := NewRegistry()

	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"calculator": "result := a + b",
		},
	}
	registry.RegisterEmbeddedProvider(provider)

	// Load all scripts
	err = registry.LoadScripts()
	require.NoError(t, err)

	// External Zygomys script should replace embedded Tengo script
	script, err := registry.GetScript("test_module", "calculator")
	require.NoError(t, err)
	assert.Equal(t, "(+ a b)", script.Content)
	assert.Equal(t, SourceExternal, script.Source)
	assert.Equal(t, LanguageZygomys, script.Language)
	assert.Equal(t, LanguageTengo, script.OriginalLanguage) // Original was Tengo
}

func TestRegistry_NoExternalScriptsDirectory(t *testing.T) {
	// Create temporary directory without scripts folder
	tempDir, err := os.MkdirTemp("", "script_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory for test
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create registry and try to load external scripts
	registry := NewRegistry()
	err = registry.LoadExternalScripts()

	// Should not error when scripts directory doesn't exist
	assert.NoError(t, err)

	scripts := registry.ListScripts()
	assert.Empty(t, scripts)
}
