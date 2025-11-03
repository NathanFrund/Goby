package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goby-cli",
	Short: "Goby CLI tool",
	Long: `Goby CLI is a command-line interface for the Goby framework.

Available commands:
  list-services    Discover and list registered services in the Goby registry
  
Use "goby [command] --help" for more information about a specific command.`,
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
