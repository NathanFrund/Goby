package cmd

import (
	"github.com/spf13/cobra"
)

// topicsCmd represents the topics command
var topicsCmd = &cobra.Command{
	Use:   "topics",
	Short: "Manage and explore Goby framework topics",
	Long: `The topics command provides tools for discovering, inspecting, and validating
topics in the Goby framework. Topics are used for event-driven communication
between modules and framework components.

Available subcommands:
  list      List all registered topics with optional filtering
  get       Get detailed information about a specific topic
  validate  Validate a topic name and definition

Examples:
  # List all topics
  goby-cli topics list
  
  # List topics for a specific module
  goby-cli topics list --module=chat
  
  # List framework-level topics only
  goby-cli topics list --scope=framework
  
  # Get detailed information about a topic
  goby-cli topics get chat.message.sent
  
  # Validate a topic name
  goby-cli topics validate chat.message.sent

Use "goby-cli topics [command] --help" for more information about a specific command.`,
}

func init() {
	rootCmd.AddCommand(topicsCmd)
}
