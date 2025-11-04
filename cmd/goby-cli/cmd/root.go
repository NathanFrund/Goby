package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goby-cli",
	Short: "Goby CLI tool for the Goby framework",
	Long: `Goby CLI is a unified command-line interface for the Goby framework.
It provides tools for discovering services, managing topics, scaffolding modules,
and other development tasks.

Available commands:
  list-services    Discover and list registered services in the Goby registry
  new-module       Scaffold a new application module with boilerplate code
  topics           Manage and explore Goby framework topics (list, get, validate)
  version          Print the version number of Goby CLI

Examples:
  # Service discovery
  goby-cli list-services                    # List all services
  goby-cli list-services --format json     # List services in JSON format
  
  # Topic management
  goby-cli topics list                      # List all topics
  goby-cli topics get chat.message.sent    # Get topic details
  goby-cli topics validate my.topic.name   # Validate topic name
  
  # Module scaffolding
  goby-cli new-module --name inventory      # Create new module
  
  # General
  goby-cli version                          # Show version information

Use "goby-cli [command] --help" for more information about a specific command.`,
}

// Execute executes the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you can define your flags and configuration settings
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
}
