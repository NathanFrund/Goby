package analyzer

import (
	"path/filepath"
	"strings"
)

// SetKeyMatcher matches registry.Set calls with their corresponding Key declarations
type SetKeyMatcher struct {
	keys []RegistryKeyInfo
	sets []RegistrySetInfo
}

// NewSetKeyMatcher creates a new matcher with parsed keys and sets
func NewSetKeyMatcher(keys []RegistryKeyInfo, sets []RegistrySetInfo) *SetKeyMatcher {
	return &SetKeyMatcher{
		keys: keys,
		sets: sets,
	}
}

// MatchedService represents a service with both key declaration and set call information
type MatchedService struct {
	Key       RegistryKeyInfo
	SetCall   *RegistrySetInfo
	Module    string
	IsMatched bool
}

// MatchServices matches Set calls with Key declarations
func (m *SetKeyMatcher) MatchServices() []MatchedService {
	var matched []MatchedService

	// Create a map of variable names to keys for quick lookup
	keysByVar := make(map[string]RegistryKeyInfo)
	for _, key := range m.keys {
		keysByVar[key.VarName] = key
	}

	// Track which keys have been matched
	matchedKeys := make(map[string]bool)

	// Match each Set call with its corresponding Key
	for _, set := range m.sets {
		if key, exists := keysByVar[set.KeyVar]; exists {
			module := m.extractModuleFromPath(key.FilePath)
			matched = append(matched, MatchedService{
				Key:       key,
				SetCall:   &set,
				Module:    module,
				IsMatched: true,
			})
			matchedKeys[key.VarName] = true
		}
	}

	// Add unmatched keys (declared but not used in Set calls)
	for _, key := range m.keys {
		if !matchedKeys[key.VarName] {
			module := m.extractModuleFromPath(key.FilePath)
			matched = append(matched, MatchedService{
				Key:       key,
				SetCall:   nil,
				Module:    module,
				IsMatched: false,
			})
		}
	}

	return matched
}

// extractModuleFromPath extracts module name from file path
func (m *SetKeyMatcher) extractModuleFromPath(filePath string) string {
	// Normalize path separators
	normalizedPath := filepath.ToSlash(filePath)

	// Extract module from path patterns
	if strings.Contains(normalizedPath, "internal/modules/") {
		parts := strings.Split(normalizedPath, "internal/modules/")
		if len(parts) > 1 {
			moduleParts := strings.Split(parts[1], "/")
			if len(moduleParts) > 0 {
				return moduleParts[0]
			}
		}
	}

	// Check for cmd directory
	if strings.Contains(normalizedPath, "cmd/") {
		parts := strings.Split(normalizedPath, "cmd/")
		if len(parts) > 1 {
			cmdParts := strings.Split(parts[1], "/")
			if len(cmdParts) > 0 {
				return "cmd/" + cmdParts[0]
			}
		}
	}

	// Check for internal core services
	if strings.Contains(normalizedPath, "internal/") {
		parts := strings.Split(normalizedPath, "internal/")
		if len(parts) > 1 {
			internalParts := strings.Split(parts[1], "/")
			if len(internalParts) > 0 {
				return "core/" + internalParts[0]
			}
		}
	}

	return "unknown"
}

// GetUnmatchedSets returns Set calls that don't have corresponding Key declarations
func (m *SetKeyMatcher) GetUnmatchedSets() []RegistrySetInfo {
	keysByVar := make(map[string]bool)
	for _, key := range m.keys {
		keysByVar[key.VarName] = true
	}

	var unmatched []RegistrySetInfo
	for _, set := range m.sets {
		if !keysByVar[set.KeyVar] {
			unmatched = append(unmatched, set)
		}
	}

	return unmatched
}

// GetServicesByModule groups services by module
func (m *SetKeyMatcher) GetServicesByModule() map[string][]MatchedService {
	services := m.MatchServices()
	byModule := make(map[string][]MatchedService)

	for _, service := range services {
		module := service.Module
		byModule[module] = append(byModule[module], service)
	}

	return byModule
}
