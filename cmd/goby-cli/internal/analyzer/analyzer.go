package analyzer

import (
	"fmt"
)

// Analyzer provides the main interface for service registry analysis
type Analyzer struct {
	aggregator *ServiceAggregator
}

// New creates a new analyzer instance
func New() *Analyzer {
	return &Analyzer{
		aggregator: NewServiceAggregator(),
	}
}

// AnalyzeProject analyzes a project and returns service information
func (a *Analyzer) AnalyzeProject(rootPath string) (*AnalysisResult, error) {
	services, err := a.aggregator.AnalyzeProject(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze project: %w", err)
	}

	summary := a.aggregator.GetProjectSummary(services)
	issues := a.aggregator.ValidateServices(services)
	byCategory := a.aggregator.GetServicesByCategory(services)

	return &AnalysisResult{
		Services:   services,
		Summary:    summary,
		Issues:     issues,
		Categories: byCategory,
	}, nil
}

// FindService finds a specific service by key
func (a *Analyzer) FindService(rootPath, serviceKey string) (*ServiceMetadata, error) {
	services, err := a.aggregator.AnalyzeProject(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze project: %w", err)
	}

	service := a.aggregator.GetServiceByKey(services, serviceKey)
	if service == nil {
		return nil, fmt.Errorf("service not found: %s", serviceKey)
	}

	return service, nil
}

// FilterServices filters services based on criteria
func (a *Analyzer) FilterServices(rootPath string, filter ServiceFilter) ([]ServiceMetadata, error) {
	services, err := a.aggregator.AnalyzeProject(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze project: %w", err)
	}

	return a.aggregator.FilterServices(services, filter), nil
}

// AnalysisResult contains the complete analysis of a project's services
type AnalysisResult struct {
	Services   []ServiceMetadata
	Summary    ProjectSummary
	Issues     []ValidationIssue
	Categories map[string][]ServiceMetadata
}
