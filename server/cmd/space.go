package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"

	"github.com/tylerjvollick/nori/internal/cli"
)

// spaceItem mirrors the space response from the spaces API.
type spaceItem struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Slug             string  `json:"slug"`
	IsDefault        bool    `json:"isDefault"`
	DefaultLaborRate *string `json:"defaultLaborRate"`
}

var spaceJSONFlag bool

// Flags for space update
var (
	spaceUpdateDefaultLaborRate      string
	spaceUpdateClearDefaultLaborRate bool
)

var spaceCmd = &cobra.Command{
	Use:   "space",
	Short: "Manage spaces",
	Long:  "Space commands: update settings for the current space.",
}

var spaceUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the current space",
	Long: `Update settings for the current space.

Use --default-labor-rate to set the space's default hourly labor rate
(e.g. --default-labor-rate 50.00). It is used as the fallback cost rate for
stations without their own costs/hour. Use --clear-default-labor-rate to
remove it.

The target space is resolved from --space, NORI_SPACE, or stored credentials.`,
	Args: cobra.NoArgs,
	RunE: runSpaceUpdate,
}

func init() {
	spaceCmd.PersistentFlags().BoolVar(&spaceJSONFlag, "json", false, "Output as JSON")
	spaceUpdateCmd.Flags().StringVar(&spaceUpdateDefaultLaborRate, "default-labor-rate", "", "Default hourly labor rate for the space (e.g. 50.00)")
	spaceUpdateCmd.Flags().BoolVar(&spaceUpdateClearDefaultLaborRate, "clear-default-labor-rate", false, "Clear the default labor rate")
	spaceUpdateCmd.MarkFlagsMutuallyExclusive("default-labor-rate", "clear-default-labor-rate")
	spaceCmd.AddCommand(spaceUpdateCmd)
	rootCmd.AddCommand(spaceCmd)
}

// runSpaceUpdate implements `nori space update`.
func runSpaceUpdate(cmd *cobra.Command, args []string) error {
	body := map[string]interface{}{}
	if cmd.Flags().Changed("default-labor-rate") {
		rate, err := decimal.NewFromString(spaceUpdateDefaultLaborRate)
		if err != nil {
			return fmt.Errorf("invalid --default-labor-rate value %q: must be a decimal number", spaceUpdateDefaultLaborRate)
		}
		if rate.IsNegative() {
			return fmt.Errorf("--default-labor-rate must not be negative")
		}
		body["defaultLaborRate"] = rate.String()
	}
	if spaceUpdateClearDefaultLaborRate {
		body["defaultLaborRate"] = nil
	}

	if len(body) == 0 {
		return fmt.Errorf("nothing to update: pass --default-labor-rate or --clear-default-labor-rate")
	}

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	spaceID := resolveSpaceID(creds)
	if spaceID == "" {
		return fmt.Errorf("no space selected: pass --space, set NORI_SPACE, or run `nori login`")
	}

	client := newClientWithSpace(creds)

	resp, err := client.Put("/api/spaces/"+spaceID, body)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := cli.Handle401(resp); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("server error: %s", errResp.Error)
		}
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var space spaceItem
	if err := cli.ReadJSON(resp, &space); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if spaceJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(space)
	}

	if space.DefaultLaborRate != nil {
		fmt.Printf("Updated space: %s (default labor rate: %s)\n", space.Name, *space.DefaultLaborRate)
	} else {
		fmt.Printf("Updated space: %s (default labor rate: not set)\n", space.Name)
	}
	return nil
}
