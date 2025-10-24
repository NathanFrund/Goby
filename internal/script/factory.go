package script

import (
	"fmt"
)

// Factory implements the EngineFactory interface
type Factory struct {
	supportedLanguages []ScriptLanguage
}

// NewFactory creates a new engine factory
func NewFactory() *Factory {
	return &Factory{
		supportedLanguages: []ScriptLanguage{
			LanguageTengo,
			// LanguageZygomys will be added in a later task
		},
	}
}

// CreateEngine returns an engine for the specified language
func (f *Factory) CreateEngine(language ScriptLanguage) (LanguageEngine, error) {
	switch language {
	case LanguageTengo:
		return NewTengoEngine(), nil
	case LanguageZygomys:
		return nil, fmt.Errorf("zygomys engine not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported script language: %s", language)
	}
}

// SupportedLanguages returns all supported script languages
func (f *Factory) SupportedLanguages() []ScriptLanguage {
	// Return a copy to prevent modification
	languages := make([]ScriptLanguage, len(f.supportedLanguages))
	copy(languages, f.supportedLanguages)
	return languages
}