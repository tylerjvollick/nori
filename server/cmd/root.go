package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/tylerjvollick/nori/internal/cli"
)

// spaceFlag holds the value of the --space global flag.
var spaceFlag string

// rootCmd is the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "nori",
	Short: "Nori — AI-native manufacturing operations tool",
	Long:  "Nori is a self-hosted, AI-native manufacturing operations tool built for small shops.",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&spaceFlag, "space", "", "Space ID to use (overrides config and NORI_SPACE env)")
}

// resolveSpaceID returns the effective space ID by checking: flag > env > credentials.
func resolveSpaceID(creds *cli.Credentials) string {
	if spaceFlag != "" {
		return spaceFlag
	}
	if envSpace := os.Getenv("NORI_SPACE"); envSpace != "" {
		return envSpace
	}
	return creds.SpaceID
}

// newClientWithSpace creates a CLI client from credentials with the space ID
// resolved from the --space flag, NORI_SPACE env var, or stored credentials.
func newClientWithSpace(creds *cli.Credentials) *cli.Client {
	client := cli.NewClient(creds)
	client.SpaceID = resolveSpaceID(creds)
	return client
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
