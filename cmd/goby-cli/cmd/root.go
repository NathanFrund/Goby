package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goby",
	Short: "Goby CLI tool",
	Long: `Goby CLI is a command-line interface for the Goby framework.
Use it to create new modules, manage dependencies, and more.`,
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
