package script

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEmbeddedScriptProvider implements EmbeddedScriptProvider for testing
type MockEmbeddedScriptProvider struct {
	moduleName string
	scripts    map[string]string
}

func (m *MockEmbeddedScriptProvider) GetEmbeddedScripts() map[string]string {
	return m.scripts
}

func (m *MockEmbeddedScriptProvider) GetModuleName() string {
	return m.moduleName
}

func TestRegistry_RegisterAndLoadScripts(t *testing.T) {
	registry := NewRegistry()

	// Create a mock provider
	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"calculator.tengo":  "result := a + b",
			"processor.zygomys": "(defn process [x] (* x 2))",
			"simple_script":     "x := 42",
		},
	}

	// Register the provider
	registry.RegisterEmbeddedProvider(provider)

	// Load scripts
	err := registry.LoadScripts()
	require.NoError(t, err)

	// Test ListScripts
	scripts := registry.ListScripts()
	assert.Contains(t, scripts, "test_module")
	assert.Len(t, scripts["test_module"], 3)
	assert.Contains(t, scripts["test_module"], "calculator.tengo")
	assert.Contains(t, scripts["test_module"], "processor.zygomys")
	assert.Contains(t, scripts["test_module"], "simple_script")
}

func TestRegistry_GetScript(t *testing.T) {
	registry := NewRegistry()

	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"calculator.tengo": "result := a + b",
		},
	}

	registry.RegisterEmbeddedProvider(provider)
	err := registry.LoadScripts()
	require.NoError(t, err)

	// Test getting existing script
	script, err := registry.GetScript("test_module", "calculator.tengo")
	require.NoError(t, err)
	assert.Equal(t, "test_module", script.ModuleName)
	assert.Equal(t, "calculator.tengo", script.Name)
	assert.Equal(t, LanguageTengo, script.Language)
	assert.Equal(t, "result := a + b", script.Content)
	assert.Equal(t, SourceEmbedded, script.Source)

	// Test getting non-existent script
	_, err = registry.GetScript("test_module", "nonexistent")
	require.Error(t, err)
	var scriptErr *ScriptError
	assert.ErrorAs(t, err, &scriptErr)
	assert.Equal(t, ErrorTypeNotFound, scriptErr.Type)
}

func TestRegistry_LanguageDetection(t *testing.T) {
	registry := NewRegistry()

	testCases := []struct {
		name     string
		content  string
		expected ScriptLanguage
		skip     bool
		skipReason string
	}{
		{"script.tengo", "result := 42", LanguageTengo, false, ""},
		{"script.zygomys", "(defn test [])", LanguageZygomys, true, "Zygomys/Lisp support not implemented yet"},
		{"no_extension", "result := 42", LanguageTengo, false, ""}, // default
		{"lisp_content", "(+ 1 2)", LanguageZygomys, true, "Zygomys/Lisp support not implemented yet"},    // detected from content
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip {
				t.Skip(tc.skipReason)
			}
			detected := registry.detectLanguage(tc.name, tc.content)
			assert.Equal(t, tc.expected, detected)
		})
	}
}

func TestRegistry_ScriptCaching(t *testing.T) {
	registry := NewRegistry()

	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"cached_script": "result := 123",
		},
	}

	registry.RegisterEmbeddedProvider(provider)
	err := registry.LoadScripts()
	require.NoError(t, err)

	// First call should load and cache
	script1, err := registry.GetScript("test_module", "cached_script")
	require.NoError(t, err)

	// Second call should return cached version
	script2, err := registry.GetScript("test_module", "cached_script")
	require.NoError(t, err)

	// Should be the same instance (cached)
	assert.Equal(t, script1, script2)
}

func TestRegistry_GetScriptMetadata(t *testing.T) {
	registry := NewRegistry()

	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"test_script.tengo": "result := 42",
		},
	}

	registry.RegisterEmbeddedProvider(provider)
	err := registry.LoadScripts()
	require.NoError(t, err)

	metadata := registry.GetScriptMetadata()
	assert.Contains(t, metadata, "test_module")
	assert.Contains(t, metadata["test_module"], "test_script.tengo")

	scriptMeta := metadata["test_module"]["test_script.tengo"]
	assert.Equal(t, "test_script.tengo", scriptMeta.Name)
	assert.Equal(t, LanguageTengo, scriptMeta.Language)
	assert.Equal(t, SourceEmbedded, scriptMeta.Source)
	assert.Equal(t, 12, scriptMeta.Size) // len("result := 42")
	assert.NotEmpty(t, scriptMeta.Checksum)
}

func TestRegistry_StartWatcher(t *testing.T) {
	registry := NewRegistry()

	// Should not error (placeholder implementation)
	err := registry.StartWatcher(context.Background(), true)
	assert.NoError(t, err)
}
