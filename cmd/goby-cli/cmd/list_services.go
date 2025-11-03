package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/nfrund/goby/cmd/goby-cli/internal/analyzer"
	"github.com/spf13/cobra"
)

var (
	outputFormat    string
	serviceCategory string
	moduleFilter    string
	registeredOnly  bool
	declaredOnly    bool
)

// listServicesCmd represents the list-services command
var listServicesCmd = &cobra.Command{
	Use:   "list-services [service-name]",
	Short: "List all registered services in the Goby registry",
	Long: `Displays all services currently registered in the Goby application registry.
This command helps developers discover what dependencies are available for their modules to use.

The command uses static analysis to discover services by parsing Go source files for 
registry.Key declarations and registry.Set calls. This approach is safe and read-only.

Examples:
  # Basic usage
  goby-cli list-services                           # List all services in table format
  goby-cli list-services --format json            # List all services in JSON format
  
  # Filtering options
  goby-cli list-services --category core          # Show only core services
  goby-cli list-services --category module        # Show only module services
  goby-cli list-services --module wargame         # Show services from wargame module
  goby-cli list-services --registered-only        # Show only registered services
  goby-cli list-services --declared-only          # Show only declared but unregistered services
  
  # Detailed service information
  goby-cli list-services core.database.Connection # Show detailed info for specific service
  goby-cli list-services wargame.Engine --format json # Service details in JSON format
  
  # Combined filtering
  goby-cli list-services --category core --format json # Core services in JSON format

Note: This command must be run from the root directory of a Go project with a go.mod file.`,
	Args: cobra.MaximumNArgs(1), // Allow 0 or 1 argument (optional service name)
	Run:  listServicesHandler,
}

func listServicesHandler(cmd *cobra.Command, args []string) {
	// Get current working directory as project root
	projectRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get current directory: %v\n", err)
		os.Exit(1)
	}

	// Validate that we're in a Go project directory
	if err := validateGoProject(projectRoot); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Initialize analyzer
	serviceAnalyzer := analyzer.New()

	// Handle specific service lookup
	if len(args) > 0 {
		serviceName := args[0]
		service, err := serviceAnalyzer.FindService(projectRoot, serviceName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		displayServiceDetails(service, outputFormat)
		return
	}

	// Analyze project for all services
	result, err := serviceAnalyzer.AnalyzeProject(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to analyze project: %v\n", err)
		os.Exit(1)
	}

	// Apply filters if specified
	services := result.Services
	if serviceCategory != "" || moduleFilter != "" || registeredOnly || declaredOnly {
		filter := analyzer.ServiceFilter{
			Category: serviceCategory,
			Module:   moduleFilter,
		}

		// Handle registration status filters
		if registeredOnly {
			registered := true
			filter.Registered = &registered
		} else if declaredOnly {
			declared := false
			filter.Registered = &declared
		}

		filteredServices, err := serviceAnalyzer.FilterServices(projectRoot, filter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to filter services: %v\n", err)
			os.Exit(1)
		}
		services = filteredServices
	}

	// Handle empty results
	if len(services) == 0 {
		message := "No services found"
		filters := []string{}

		if serviceCategory != "" {
			filters = append(filters, fmt.Sprintf("category '%s'", serviceCategory))
		}
		if moduleFilter != "" {
			filters = append(filters, fmt.Sprintf("module '%s'", moduleFilter))
		}
		if registeredOnly {
			filters = append(filters, "registered services only")
		}
		if declaredOnly {
			filters = append(filters, "declared but unregistered services only")
		}

		if len(filters) > 0 {
			message += " matching: " + strings.Join(filters, ", ")
		} else {
			message += " in the project"
		}

		fmt.Println(message)
		return
	}

	// Display services based on format
	switch outputFormat {
	case "json":
		displayServicesJSON(services)
	case "table":
		displayServicesTable(services)
	default:
		fmt.Fprintf(os.Stderr, "Error: Unsupported output format '%s'. Use 'table' or 'json'\n", outputFormat)
		os.Exit(1)
	}

	// Show summary
	if outputFormat == "table" {
		displaySummary(result.Summary)
	}
}

func init() {
	rootCmd.AddCommand(listServicesCmd)

	// Add flags for output formatting and filtering
	listServicesCmd.Flags().StringVarP(&outputFormat, "format", "f", "table", "Output format (table, json)")
	listServicesCmd.Flags().StringVarP(&serviceCategory, "category", "c", "", "Filter services by category (core, module, test, command)")
	listServicesCmd.Flags().StringVarP(&moduleFilter, "module", "m", "", "Filter services by module name")
	listServicesCmd.Flags().BoolVar(&registeredOnly, "registered-only", false, "Show only registered services")
	listServicesCmd.Flags().BoolVar(&declaredOnly, "declared-only", false, "Show only declared but unregistered services")

	// Make registered-only and declared-only mutually exclusive
	listServicesCmd.MarkFlagsMutuallyExclusive("registered-only", "declared-only")
}

// displayServicesTable displays services in a formatted table
func displayServicesTable(services []analyzer.ServiceMetadata) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Print header
	fmt.Fprintln(w, "KEY\tTYPE\tMODULE\tSTATUS\tDESCRIPTION")
	fmt.Fprintln(w, "---\t----\t------\t------\t-----------")

	// Print services
	for _, service := range services {
		status := "declared"
		if service.IsRegistered {
			status = "registered"
		}

		// Truncate long descriptions for table display
		description := service.Description
		if len(description) > 50 {
			description = description[:47] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			service.Key,
			service.Type,
			service.Module,
			status,
			description,
		)
	}
}

// displayServicesJSON displays services in JSON format
func displayServicesJSON(services []analyzer.ServiceMetadata) {
	output := struct {
		Services []analyzer.ServiceMetadata `json:"services"`
		Count    int                        `json:"count"`
	}{
		Services: services,
		Count:    len(services),
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to encode JSON: %v\n", err)
		os.Exit(1)
	}
}

// displayServiceDetails displays detailed information for a specific service
func displayServiceDetails(service *analyzer.ServiceMetadata, format string) {
	if format == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(service); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to encode JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Table format for detailed view
	fmt.Printf("Service Details: %s\n", service.Key)
	fmt.Println(strings.Repeat("=", len(service.Key)+17))
	fmt.Printf("Key:         %s\n", service.Key)
	fmt.Printf("Type:        %s\n", service.Type)
	fmt.Printf("Module:      %s\n", service.Module)
	fmt.Printf("Category:    %s\n", service.Category)
	fmt.Printf("Status:      %s\n", getStatusString(service.IsRegistered))
	fmt.Printf("Description: %s\n", service.Description)

	fmt.Printf("\nFile Locations:\n")
	fmt.Printf("  Declaration: %s:%d\n", getRelativePath(service.FilePath), service.LineNumber)

	if service.IsRegistered && service.SetLocation != "" {
		fmt.Printf("  Registration: %s:%d\n", getRelativePath(service.SetLocation), service.SetLineNumber)
	}

	// Add availability information
	displayAvailabilityInfo(service)
}

// displaySummary displays a summary of the analysis results
func displaySummary(summary analyzer.ProjectSummary) {
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Total services: %d\n", summary.TotalServices)
	fmt.Printf("  Registered:     %d\n", summary.RegisteredServices)
	fmt.Printf("  Declared only:  %d\n", summary.TotalServices-summary.RegisteredServices)

	if len(summary.Categories) > 0 {
		fmt.Printf("\nBy Category:\n")
		for category, count := range summary.Categories {
			fmt.Printf("  %s: %d\n", category, count)
		}
	}
}

// getStatusString returns a human-readable status string
func getStatusString(isRegistered bool) string {
	if isRegistered {
		return "registered"
	}
	return "declared only"
}

// getRelativePath converts absolute paths to relative paths for cleaner display
func getRelativePath(fullPath string) string {
	if cwd, err := os.Getwd(); err == nil {
		if relPath, err := filepath.Rel(cwd, fullPath); err == nil {
			return relPath
		}
	}
	return fullPath
}

// displayAvailabilityInfo shows simple availability information
func displayAvailabilityInfo(service *analyzer.ServiceMetadata) {
	fmt.Printf("\nAvailability:\n")

	if service.IsRegistered {
		fmt.Printf("  ✓ This service is registered and available for modules to request\n")
		fmt.Printf("  ✓ Modules can access this service through their dependency struct\n")

		// Show the correct type for Dependencies struct
		fieldName := getDependencyFieldName(service.Type)
		dependencyType := getDependencyType(service.Key, service.Type)
		fmt.Printf("  Add to Dependencies struct: %s %s\n", fieldName, dependencyType)
	} else {
		fmt.Printf("  ⚠ This service is declared but not registered\n")
		fmt.Printf("  ⚠ Check the registration code to make it available\n")
		fmt.Printf("  Once registered, modules can request it through their dependency struct\n")
	}
}

// validateGoProject checks if the current directory appears to be a Go project
func validateGoProject(projectRoot string) error {
	// Check for go.mod file
	goModPath := filepath.Join(projectRoot, "go.mod")
	if _, err := os.Stat(goModPath); err != nil {
		return fmt.Errorf("not a Go module directory (no go.mod found). Please run this command from the root of a Go project")
	}

	// Check for registry package (optional but helpful)
	registryPath := filepath.Join(projectRoot, "internal", "registry")
	if _, err := os.Stat(registryPath); err != nil {
		// This is just a warning, not an error
		fmt.Fprintf(os.Stderr, "Warning: No internal/registry directory found. This may not be a Goby project.\n")
	}

	return nil
}

// getDependencyFieldName generates the appropriate field name for a Dependencies struct
func getDependencyFieldName(serviceType string) string {
	// Remove pointer prefix
	typeName := strings.TrimPrefix(serviceType, "*")

	// Handle package-qualified types
	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		if len(parts) >= 2 {
			// For types like "script.ScriptEngine", use "ScriptEngine"
			typeName = parts[len(parts)-1]
		}
	}

	// Convert to appropriate field name (PascalCase)
	if len(typeName) > 0 {
		return strings.ToUpper(string(typeName[0])) + typeName[1:]
	}

	return "Service"
}

// getDependencyType converts a registry key and type to the correct Go type for Dependencies struct
func getDependencyType(registryKey, serviceType string) string {
	// Map common registry keys to their correct Go types
	switch registryKey {
	case "core.script.Engine":
		return "script.ScriptEngine"
	case "core.database.Connection":
		return "*database.Connection"
	case "core.presence.Service":
		return "*presence.Service"
	case "wargame.Engine":
		return "*Engine"
	default:
		// For unknown services, try to infer from the key
		if strings.HasPrefix(registryKey, "core.") {
			parts := strings.Split(registryKey, ".")
			if len(parts) >= 3 {
				// core.package.Type -> package.Type
				packageName := parts[1]
				typeName := parts[2]
				return fmt.Sprintf("%s.%s", packageName, typeName)
			}
		}

		// Fallback to the original service type
		return serviceType
	}
}
