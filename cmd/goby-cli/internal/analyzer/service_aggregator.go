package analyzer

import (
	"fmt"
	"sort"
	"strings"
)

// ServiceMetadata represents comprehensive information about a registered service
type ServiceMetadata struct {
	Key           string
	Type          string
	Module        string
	Description   string
	FilePath      string
	LineNumber    int
	IsRegistered  bool
	SetLocation   string
	SetLineNumber int
	Category      string
}

// ServiceAggregator combines registry key and set information into comprehensive service metadata
type ServiceAggregator struct {
	parser  *RegistryParser
	matcher *SetKeyMatcher
}

// NewServiceAggregator creates a new service aggregator
func NewServiceAggregator() *ServiceAggregator {
	return &ServiceAggregator{
		parser: NewRegistryParser(),
	}
}

// AnalyzeProject analyzes a project directory and returns comprehensive service information
func (a *ServiceAggregator) AnalyzeProject(rootPath string) ([]ServiceMetadata, error) {
	// Parse all files for registry patterns
	keys, sets, err := a.parser.ParseDirectory(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory: %w", err)
	}

	// Create matcher and match services
	a.matcher = NewSetKeyMatcher(keys, sets)
	matchedServices := a.matcher.MatchServices()

	// Convert to service metadata
	var services []ServiceMetadata
	for _, matched := range matchedServices {
		metadata := a.createServiceMetadata(matched)
		services = append(services, metadata)
	}

	// Sort services by key for consistent output
	sort.Slice(services, func(i, j int) bool {
		return services[i].Key < services[j].Key
	})

	return services, nil
}

// createServiceMetadata creates comprehensive service metadata from matched service
func (a *ServiceAggregator) createServiceMetadata(matched MatchedService) ServiceMetadata {
	metadata := ServiceMetadata{
		Key:          matched.Key.KeyString,
		Type:         matched.Key.Type,
		Module:       matched.Module,
		Description:  a.cleanDescription(matched.Key.Description),
		FilePath:     matched.Key.FilePath,
		LineNumber:   matched.Key.LineNumber,
		IsRegistered: matched.IsMatched,
		Category:     a.categorizeService(matched.Key.KeyString, matched.Module),
	}

	// Add Set call information if available
	if matched.SetCall != nil {
		metadata.SetLocation = matched.SetCall.FilePath
		metadata.SetLineNumber = matched.SetCall.LineNumber
	}

	return metadata
}

// cleanDescription cleans and formats service descriptions
func (a *ServiceAggregator) cleanDescription(description string) string {
	if description == "" {
		return "No description available"
	}

	// Remove common prefixes and clean up
	cleaned := strings.TrimSpace(description)

	// Remove variable name prefixes like "KeyGameEngine is the..."
	if strings.Contains(cleaned, " is the ") {
		parts := strings.SplitN(cleaned, " is the ", 2)
		if len(parts) == 2 {
			cleaned = "The " + parts[1]
		}
	}

	// Remove "type-safe key for accessing" patterns
	cleaned = strings.ReplaceAll(cleaned, "type-safe key for accessing the ", "")
	cleaned = strings.ReplaceAll(cleaned, "type-safe key for accessing ", "")
	cleaned = strings.ReplaceAll(cleaned, " service from the registry", "")
	cleaned = strings.ReplaceAll(cleaned, " service.", "")

	// Capitalize first letter
	if len(cleaned) > 0 {
		cleaned = strings.ToUpper(string(cleaned[0])) + cleaned[1:]
	}

	return cleaned
}

// categorizeService determines the category of a service based on its key and module
func (a *ServiceAggregator) categorizeService(key, module string) string {
	keyLower := strings.ToLower(key)
	moduleLower := strings.ToLower(module)

	// Core services
	if strings.HasPrefix(keyLower, "core.") || moduleLower == "core" || strings.Contains(moduleLower, "core/") {
		return "core"
	}

	// Test services
	if strings.HasPrefix(keyLower, "test.") || strings.Contains(moduleLower, "test") {
		return "test"
	}

	// Command services
	if strings.Contains(moduleLower, "cmd/") {
		return "command"
	}

	// Module services
	if strings.Contains(moduleLower, "modules/") || (!strings.Contains(moduleLower, "core") && !strings.Contains(moduleLower, "cmd")) {
		return "module"
	}

	return "other"
}

// GetServicesByCategory groups services by category
func (a *ServiceAggregator) GetServicesByCategory(services []ServiceMetadata) map[string][]ServiceMetadata {
	byCategory := make(map[string][]ServiceMetadata)

	for _, service := range services {
		category := service.Category
		byCategory[category] = append(byCategory[category], service)
	}

	return byCategory
}

// GetServiceByKey finds a specific service by its key
func (a *ServiceAggregator) GetServiceByKey(services []ServiceMetadata, key string) *ServiceMetadata {
	for _, service := range services {
		if service.Key == key {
			return &service
		}
	}
	return nil
}

// FilterServices filters services based on criteria
func (a *ServiceAggregator) FilterServices(services []ServiceMetadata, filter ServiceFilter) []ServiceMetadata {
	var filtered []ServiceMetadata

	for _, service := range services {
		if a.matchesFilter(service, filter) {
			filtered = append(filtered, service)
		}
	}

	return filtered
}

// ServiceFilter defines criteria for filtering services
type ServiceFilter struct {
	Category     string
	Module       string
	Registered   *bool // nil means don't filter, true/false for specific values
	KeyContains  string
	TypeContains string
}

// matchesFilter checks if a service matches the filter criteria
func (a *ServiceAggregator) matchesFilter(service ServiceMetadata, filter ServiceFilter) bool {
	if filter.Category != "" && service.Category != filter.Category {
		return false
	}

	if filter.Module != "" && !strings.Contains(strings.ToLower(service.Module), strings.ToLower(filter.Module)) {
		return false
	}

	if filter.Registered != nil && service.IsRegistered != *filter.Registered {
		return false
	}

	if filter.KeyContains != "" && !strings.Contains(strings.ToLower(service.Key), strings.ToLower(filter.KeyContains)) {
		return false
	}

	if filter.TypeContains != "" && !strings.Contains(strings.ToLower(service.Type), strings.ToLower(filter.TypeContains)) {
		return false
	}

	return true
}

// GetProjectSummary returns a summary of the project's services
func (a *ServiceAggregator) GetProjectSummary(services []ServiceMetadata) ProjectSummary {
	summary := ProjectSummary{
		TotalServices:      len(services),
		RegisteredServices: 0,
		Categories:         make(map[string]int),
		Modules:            make(map[string]int),
	}

	for _, service := range services {
		if service.IsRegistered {
			summary.RegisteredServices++
		}

		summary.Categories[service.Category]++
		summary.Modules[service.Module]++
	}

	return summary
}

// ProjectSummary provides an overview of services in a project
type ProjectSummary struct {
	TotalServices      int
	RegisteredServices int
	Categories         map[string]int
	Modules            map[string]int
}

// ValidateServices performs basic validation on discovered services
func (a *ServiceAggregator) ValidateServices(services []ServiceMetadata) []ValidationIssue {
	var issues []ValidationIssue

	// Check for duplicate keys
	keyCount := make(map[string]int)
	for _, service := range services {
		keyCount[service.Key]++
	}

	for key, count := range keyCount {
		if count > 1 {
			issues = append(issues, ValidationIssue{
				Type:       "duplicate_key",
				Message:    fmt.Sprintf("Duplicate service key found: %s", key),
				Severity:   "warning",
				ServiceKey: key,
			})
		}
	}

	// Check for declared but unregistered services
	for _, service := range services {
		if !service.IsRegistered {
			issues = append(issues, ValidationIssue{
				Type:       "unregistered_service",
				Message:    fmt.Sprintf("Service key declared but not registered: %s", service.Key),
				Severity:   "info",
				ServiceKey: service.Key,
				FilePath:   service.FilePath,
				LineNumber: service.LineNumber,
			})
		}
	}

	return issues
}

// ValidationIssue represents a validation problem found in the service registry
type ValidationIssue struct {
	Type       string
	Message    string
	Severity   string // "error", "warning", "info"
	ServiceKey string
	FilePath   string
	LineNumber int
}
