package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tylerjvollick/nori/internal/cli"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
	Long:  "View and modify Nori CLI configuration stored in ~/.config/nori/credentials.",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value. Supported keys:
  api-key    Set the API key for authentication (e.g., nori_...)
  server-url Set the server URL
  space      Set the default space ID`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  "Display the current CLI configuration including auth method, server URL, and user info. Secrets are not revealed.",
	RunE:  runConfigShow,
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	switch key {
	case "api-key":
		return setAPIKey(value)
	case "server-url":
		return setServerURL(value)
	case "space":
		return setSpace(value)
	default:
		return fmt.Errorf("unknown config key %q — supported keys: api-key, server-url, space", key)
	}
}

func setAPIKey(apiKey string) error {
	if !strings.HasPrefix(apiKey, "nori_") {
		return fmt.Errorf("invalid API key — API keys must start with 'nori_'")
	}

	// Load existing credentials (raw, without validation) or create new ones
	creds, err := cli.LoadCredentialsRaw()
	if err != nil {
		creds = &cli.Credentials{}
	}

	creds.APIKey = apiKey

	if err := cli.SaveCredentials(creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Println("API key saved.")
	return nil
}

func setServerURL(serverURL string) error {
	serverURL = normalizeURL(serverURL)

	creds, err := cli.LoadCredentialsRaw()
	if err != nil {
		creds = &cli.Credentials{}
	}

	creds.ServerURL = serverURL

	if err := cli.SaveCredentials(creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Printf("Server URL set to %s\n", serverURL)
	return nil
}

func setSpace(spaceID string) error {
	creds, err := cli.LoadCredentialsRaw()
	if err != nil {
		creds = &cli.Credentials{}
	}

	creds.SpaceID = spaceID

	if err := cli.SaveCredentials(creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Printf("Default space set to %s\n", spaceID)
	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	creds, err := cli.LoadCredentials()
	if err != nil {
		return fmt.Errorf("no configuration found — run 'nori login' or 'nori config set' first")
	}

	fmt.Printf("Server URL:  %s\n", creds.ServerURL)
	fmt.Printf("Auth method: %s\n", creds.AuthMethod())

	if creds.UserEmail != "" {
		fmt.Printf("User:        %s\n", creds.UserEmail)
	}

	if creds.SpaceID != "" {
		fmt.Printf("Space:       %s\n", creds.SpaceID)
	}

	if creds.APIKey != "" {
		fmt.Printf("API key:     %s...%s\n", creds.APIKey[:8], creds.APIKey[len(creds.APIKey)-4:])
	}

	if creds.AccessToken != "" {
		// Show just that a JWT is present, never the full token
		fmt.Printf("JWT:         %s...%s\n", creds.AccessToken[:8], creds.AccessToken[len(creds.AccessToken)-4:])
	}

	return nil
}
