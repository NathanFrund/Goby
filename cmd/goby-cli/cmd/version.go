package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "0.1.0" // This should be set at build time using -ldflags

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Goby CLI",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Goby CLI v%s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
