package script

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFactory_NewFactory(t *testing.T) {
	factory := NewFactory()
	assert.NotNil(t, factory)

	// Should support at least Tengo
	languages := factory.SupportedLanguages()
	assert.Contains(t, languages, LanguageTengo)
	assert.NotEmpty(t, languages)
}

func TestFactory_CreateEngine_Tengo(t *testing.T) {
	factory := NewFactory()

	engine, err := factory.CreateEngine(LanguageTengo)
	require.NoError(t, err)
	assert.NotNil(t, engine)

	// Verify it's actually a TengoEngine
	_, ok := engine.(*TengoEngine)
	assert.True(t, ok, "Expected TengoEngine instance")
}

func TestFactory_CreateEngine_Zygomys_NotImplemented(t *testing.T) {
	factory := NewFactory()

	engine, err := factory.CreateEngine(LanguageZygomys)
	assert.Error(t, err)
	assert.Nil(t, engine)
	assert.Contains(t, err.Error(), "zygomys engine not yet implemented")
}

func TestFactory_CreateEngine_UnsupportedLanguage(t *testing.T) {
	factory := NewFactory()

	engine, err := factory.CreateEngine(ScriptLanguage("unsupported"))
	assert.Error(t, err)
	assert.Nil(t, engine)
	assert.Contains(t, err.Error(), "unsupported script language")
}

func TestFactory_SupportedLanguages(t *testing.T) {
	factory := NewFactory()

	languages := factory.SupportedLanguages()
	assert.NotEmpty(t, languages)
	assert.Contains(t, languages, LanguageTengo)

	// Verify we get a copy (modification shouldn't affect original)
	originalLen := len(languages)
	_ = append(languages, ScriptLanguage("test")) // Modify the returned slice

	newLanguages := factory.SupportedLanguages()
	assert.Equal(t, originalLen, len(newLanguages), "SupportedLanguages should return a copy")
}

func TestFactory_SupportedLanguages_Consistency(t *testing.T) {
	factory := NewFactory()

	// Multiple calls should return the same languages
	languages1 := factory.SupportedLanguages()
	languages2 := factory.SupportedLanguages()

	assert.Equal(t, languages1, languages2)
}
