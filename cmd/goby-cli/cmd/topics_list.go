package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/nfrund/goby/cmd/goby-cli/internal/topics"
	"github.com/nfrund/goby/internal/topicmgr"
	"github.com/spf13/cobra"
)

var (
	listOutputFormat string
	listModuleFilter string
	listScopeFilter  string
)

// topicsListCmd represents the topics list command
var topicsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered topics",
	Long: `List all topics currently registered in the Goby framework.
This command helps developers discover what topics are available for event-driven communication.

The command initializes the framework to register all module topics and displays them
in either table or JSON format with optional filtering capabilities.

Examples:
  # Basic usage
  goby-cli topics list                           # List all topics in table format
  goby-cli topics list --format json            # List all topics in JSON format
  
  # Filtering options
  goby-cli topics list --module chat            # Show only topics from chat module
  goby-cli topics list --scope framework        # Show only framework-level topics
  goby-cli topics list --scope module           # Show only module-level topics
  
  # Combined filtering
  goby-cli topics list --module chat --format json # Chat module topics in JSON format

Output formats:
  table - Human-readable table format (default)
  json  - Machine-readable JSON format with metadata`,
	Run: topicsListHandler,
}

func topicsListHandler(cmd *cobra.Command, args []string) {
	// Initialize topics system
	if err := topics.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize topics: %v\n", err)
		os.Exit(1)
	}

	manager := topicmgr.Default()
	var topicList []topicmgr.Topic

	// Apply filtering based on flags
	if listModuleFilter != "" && listScopeFilter != "" {
		// Both module and scope filters
		scope := parseScope(listScopeFilter)
		if scope == "" {
			fmt.Fprintf(os.Stderr, "Error: Invalid scope '%s'. Valid scopes: framework, module\n", listScopeFilter)
			os.Exit(1)
		}

		moduleTopics := manager.ListByModule(listModuleFilter)
		for _, topic := range moduleTopics {
			if topic.Scope() == scope {
				topicList = append(topicList, topic)
			}
		}

		if listOutputFormat == "table" {
			fmt.Printf("Topics for module '%s' with scope '%s':\n\n", listModuleFilter, listScopeFilter)
		}
	} else if listModuleFilter != "" {
		// Module filter only
		topicList = manager.ListByModule(listModuleFilter)
		if listOutputFormat == "table" {
			fmt.Printf("Topics for module '%s':\n\n", listModuleFilter)
		}
	} else if listScopeFilter != "" {
		// Scope filter only
		scope := parseScope(listScopeFilter)
		if scope == "" {
			fmt.Fprintf(os.Stderr, "Error: Invalid scope '%s'. Valid scopes: framework, module\n", listScopeFilter)
			os.Exit(1)
		}
		topicList = manager.ListByScope(scope)
		if listOutputFormat == "table" {
			fmt.Printf("Topics for scope '%s':\n\n", listScopeFilter)
		}
	} else {
		// No filters - list all topics
		topicList = manager.List()
	}

	// Handle empty results
	if len(topicList) == 0 {
		message := "No topics found"
		filters := []string{}

		if listModuleFilter != "" {
			filters = append(filters, fmt.Sprintf("module '%s'", listModuleFilter))
		}
		if listScopeFilter != "" {
			filters = append(filters, fmt.Sprintf("scope '%s'", listScopeFilter))
		}

		if len(filters) > 0 {
			message += " matching: " + strings.Join(filters, ", ")
		}

		fmt.Println(message)
		return
	}

	// Display topics based on format
	switch listOutputFormat {
	case "json":
		if err := topics.DisplayTopicsJSON(topicList); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to encode JSON: %v\n", err)
			os.Exit(1)
		}
	case "table":
		topics.DisplayTopicsTable(topicList)
	default:
		fmt.Fprintf(os.Stderr, "Error: Unsupported output format '%s'. Use 'table' or 'json'\n", listOutputFormat)
		os.Exit(1)
	}
}

// parseScope converts string scope to topicmgr.TopicScope
func parseScope(scopeStr string) topicmgr.TopicScope {
	switch strings.ToLower(scopeStr) {
	case "framework":
		return topicmgr.ScopeFramework
	case "module":
		return topicmgr.ScopeModule
	default:
		return ""
	}
}

func init() {
	topicsCmd.AddCommand(topicsListCmd)

	// Add flags for output formatting and filtering
	topicsListCmd.Flags().StringVarP(&listOutputFormat, "format", "f", "table", "Output format (table, json)")
	topicsListCmd.Flags().StringVarP(&listModuleFilter, "module", "m", "", "Filter topics by module name")
	topicsListCmd.Flags().StringVarP(&listScopeFilter, "scope", "s", "", "Filter topics by scope (framework, module)")
}
