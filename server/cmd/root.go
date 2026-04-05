package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "nori",
	Short: "Nori — AI-native manufacturing operations tool",
	Long:  "Nori is a self-hosted, AI-native manufacturing operations tool built for small shops.",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
