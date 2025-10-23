package script

import "time"

// DefaultSecurityLimits provides safe default constraints for script execution
var DefaultSecurityLimits = SecurityLimits{
	MaxExecutionTime: 5 * time.Second,
	MaxMemoryBytes:   10 * 1024 * 1024, // 10MB
	AllowedPackages: []string{
		"fmt",
		"strings",
		"math",
		"rand",
	},
	ExposedFunctions: map[string]interface{}{
		// Default functions will be populated by the engine implementation
	},
}

// GetDefaultSecurityLimits returns a copy of the default security limits
func GetDefaultSecurityLimits() SecurityLimits {
	// Create a copy to prevent modification of the default
	limits := DefaultSecurityLimits

	// Deep copy the slices and maps
	limits.AllowedPackages = make([]string, len(DefaultSecurityLimits.AllowedPackages))
	copy(limits.AllowedPackages, DefaultSecurityLimits.AllowedPackages)

	limits.ExposedFunctions = make(map[string]interface{})
	for k, v := range DefaultSecurityLimits.ExposedFunctions {
		limits.ExposedFunctions[k] = v
	}

	return limits
}
