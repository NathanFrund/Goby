package cmd

import (
	"fmt"
	"os"

	"github.com/nfrund/goby/cmd/goby-cli/internal/topics"
	"github.com/nfrund/goby/internal/topicmgr"
	"github.com/spf13/cobra"
)

var (
	getOutputFormat string
)

// topicsGetCmd represents the topics get command
var topicsGetCmd = &cobra.Command{
	Use:   "get <topic-name>",
	Short: "Get detailed information about a specific topic",
	Long: `Get detailed information about a specific topic registered in the Goby framework.
This command displays comprehensive details including name, scope, module, description,
pattern, example, and metadata for the specified topic.

The command initializes the framework to register all module topics and then looks up
the requested topic by name. If the topic is not found, an appropriate error message
is displayed.

Examples:
  # Basic usage
  goby-cli topics get user.created                    # Show details for user.created topic
  goby-cli topics get chat.message.sent --format json # Show topic details in JSON format
  
  # Error handling
  goby-cli topics get nonexistent.topic              # Shows "topic not found" error

Output formats:
  table - Human-readable detailed format (default)
  json  - Machine-readable JSON format with all metadata`,
	Args: cobra.ExactArgs(1),
	Run:  topicsGetHandler,
}

func topicsGetHandler(cmd *cobra.Command, args []string) {
	topicName := args[0]

	// Initialize topics system
	if err := topics.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize topics: %v\n", err)
		os.Exit(1)
	}

	manager := topicmgr.Default()

	// Look up the topic by name
	topic, found := manager.Get(topicName)
	if !found {
		fmt.Fprintf(os.Stderr, "Error: Topic '%s' not found\n", topicName)
		fmt.Fprintf(os.Stderr, "\nUse 'goby-cli topics list' to see all available topics.\n")
		os.Exit(1)
	}

	// Display topic details based on format
	if err := topics.DisplayTopicDetails(topic, getOutputFormat); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to display topic details: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	topicsCmd.AddCommand(topicsGetCmd)

	// Add flags for output formatting
	topicsGetCmd.Flags().StringVarP(&getOutputFormat, "format", "f", "table", "Output format (table, json)")
}
