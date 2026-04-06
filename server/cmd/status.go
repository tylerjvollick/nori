package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/tylerjvollick/nori/internal/cli"
)

// statusHealthResponse mirrors the GET /health JSON response.
type statusHealthResponse struct {
	Status string `json:"status"`
}

// statusMeSpace mirrors the space objects nested in the /auth/me response.
type statusMeSpace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// statusMeResponse mirrors the fields we care about from GET /auth/me.
type statusMeResponse struct {
	ID               string          `json:"id"`
	Email            string          `json:"email"`
	FirstName        *string         `json:"firstName,omitempty"`
	LastName         *string         `json:"lastName,omitempty"`
	Role             *string         `json:"role"`
	ActiveSpaceID    *string         `json:"activeSpaceId,omitempty"`
	AccessibleSpaces []statusMeSpace `json:"accessibleSpaces"`
}

// statusResult is the combined output for --json mode.
type statusResult struct {
	Server       string `json:"server"`
	ServerStatus string `json:"serverStatus"`
	AuthMethod   string `json:"authMethod"`
	UserEmail    string `json:"userEmail,omitempty"`
	UserID       string `json:"userId,omitempty"`
	SpaceID      string `json:"spaceId,omitempty"`
	SpaceName    string `json:"spaceName,omitempty"`
}

var statusJSONFlag bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check connection to the Nori server",
	Long:  "Verify CLI configuration by checking server health and authentication. Shows server URL, health status, authenticated user, and active space.",
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSONFlag, "json", false, "Output as JSON")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	result := statusResult{
		Server:     creds.ServerURL,
		AuthMethod: creds.AuthMethod(),
	}

	// Step 1: Hit the health endpoint (public, no auth needed).
	healthResp, err := client.Get("/health")
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	var health statusHealthResponse
	if err := cli.ReadJSON(healthResp, &health); err != nil {
		return fmt.Errorf("failed to parse health response: %w", err)
	}
	result.ServerStatus = health.Status

	// Step 2: Hit /auth/me to verify authentication and get user info.
	meResp, err := client.Get("/auth/me")
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := cli.Handle401(meResp); err != nil {
		return err
	}

	if meResp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if err := cli.ReadJSON(meResp, &errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("server error: %s", errResp.Error)
		}
		return fmt.Errorf("server returned status %d", meResp.StatusCode)
	}

	var me statusMeResponse
	if err := cli.ReadJSON(meResp, &me); err != nil {
		return fmt.Errorf("failed to parse user info: %w", err)
	}

	result.UserEmail = me.Email
	result.UserID = me.ID

	// Resolve active space name from the accessible spaces list.
	spaceID := resolveSpaceID(creds)
	if spaceID != "" {
		result.SpaceID = spaceID
		for _, s := range me.AccessibleSpaces {
			if s.ID == spaceID {
				result.SpaceName = s.Name
				break
			}
		}
	} else if me.ActiveSpaceID != nil {
		result.SpaceID = *me.ActiveSpaceID
		for _, s := range me.AccessibleSpaces {
			if s.ID == *me.ActiveSpaceID {
				result.SpaceName = s.Name
				break
			}
		}
	}

	if statusJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	// Human-readable output.
	fmt.Printf("Server:   %s\n", result.Server)
	fmt.Printf("Health:   %s\n", result.ServerStatus)
	fmt.Printf("Auth:     %s\n", result.AuthMethod)
	fmt.Printf("User:     %s (%s)\n", result.UserEmail, result.UserID)
	if result.SpaceName != "" {
		fmt.Printf("Space:    %s (%s)\n", result.SpaceName, result.SpaceID)
	} else if result.SpaceID != "" {
		fmt.Printf("Space:    %s\n", result.SpaceID)
	} else {
		fmt.Println("Space:    (none)")
	}

	return nil
}
