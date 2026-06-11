package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"

	"github.com/tylerjvollick/nori/internal/cli"
)

// stationItem mirrors the station response from GET /api/v1/stations.
type stationItem struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Description  *string `json:"description,omitempty"`
	DisplayOrder int     `json:"displayOrder"`
	WIPLimit     int     `json:"wipLimit"`
	WIPCount     int     `json:"wipCount"`
	BufferSize   int     `json:"bufferSize"`
	IsActive     bool    `json:"isActive"`
	CostsHour    *string `json:"costsHour"`
}

var stationJSONFlag bool

// Flags for station create
var (
	stationCreateWIPLimit    int
	stationCreateDescription string
)

// Flags for station update
var (
	stationUpdateCostsHour      string
	stationUpdateClearCostsHour bool
)

var stationCmd = &cobra.Command{
	Use:   "station",
	Short: "Manage stations",
	Long:  "Station commands: list and inspect shop floor stations.",
}

var stationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List stations with WIP status",
	Long:  "List all stations in the current space with their WIP count and limits.",
	RunE:  runStationList,
}

var stationCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new station",
	Long:  "Create a new station in the current space (admin only). The station is appended to the end of the display order.",
	Args:  cobra.ExactArgs(1),
	RunE:  runStationCreate,
}

var stationUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a station",
	Long: `Update a station in the current space (admin only).

Use --costs-hour to set the station's hourly cost rate (e.g. --costs-hour 45.00).
Use --clear-costs-hour to remove the rate so costing falls back to the
space's default labor rate.`,
	Args: cobra.ExactArgs(1),
	RunE: runStationUpdate,
}

func init() {
	stationCmd.PersistentFlags().BoolVar(&stationJSONFlag, "json", false, "Output as JSON")
	stationCreateCmd.Flags().IntVar(&stationCreateWIPLimit, "wip-limit", 1, "WIP limit for the station")
	stationCreateCmd.Flags().StringVar(&stationCreateDescription, "description", "", "Station description")
	stationUpdateCmd.Flags().StringVar(&stationUpdateCostsHour, "costs-hour", "", "Hourly cost rate for the station (e.g. 45.00)")
	stationUpdateCmd.Flags().BoolVar(&stationUpdateClearCostsHour, "clear-costs-hour", false, "Clear the hourly cost rate (falls back to the space default labor rate)")
	stationUpdateCmd.MarkFlagsMutuallyExclusive("costs-hour", "clear-costs-hour")
	stationCmd.AddCommand(stationListCmd)
	stationCmd.AddCommand(stationCreateCmd)
	stationCmd.AddCommand(stationUpdateCmd)
	rootCmd.AddCommand(stationCmd)
}

func runStationList(cmd *cobra.Command, args []string) error {
	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	resp, err := client.Get(client.SpacePath("/stations"))
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

	var stations []stationItem
	if err := cli.ReadJSON(resp, &stations); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if stationJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(stations)
	}

	if len(stations) == 0 {
		fmt.Println("No stations found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tWIP\tLIMIT\tBUFFER\tACTIVE")
	for _, s := range stations {
		active := "yes"
		if !s.IsActive {
			active = "no"
		}
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%s\n", s.Name, s.WIPCount, s.WIPLimit, s.BufferSize, active)
	}
	return w.Flush()
}

// runStationCreate implements `nori station create <name>`.
func runStationCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	body := map[string]interface{}{
		"name":     name,
		"wipLimit": stationCreateWIPLimit,
	}
	if stationCreateDescription != "" {
		body["description"] = stationCreateDescription
	}

	resp, err := client.Post(client.SpacePath("/stations"), body)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := cli.Handle401(resp); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("server error: %s", errResp.Error)
		}
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var station stationItem
	if err := cli.ReadJSON(resp, &station); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if stationJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(station)
	}

	fmt.Printf("Created station: %s (WIP limit: %d)\n", station.Name, station.WIPLimit)
	return nil
}

// runStationUpdate implements `nori station update <id>`.
func runStationUpdate(cmd *cobra.Command, args []string) error {
	id := args[0]

	body := map[string]interface{}{}
	if cmd.Flags().Changed("costs-hour") {
		rate, err := decimal.NewFromString(stationUpdateCostsHour)
		if err != nil {
			return fmt.Errorf("invalid --costs-hour value %q: must be a decimal number", stationUpdateCostsHour)
		}
		if rate.IsNegative() {
			return fmt.Errorf("--costs-hour must not be negative")
		}
		body["costsHour"] = rate.String()
	}
	if stationUpdateClearCostsHour {
		body["costsHour"] = nil
	}

	if len(body) == 0 {
		return fmt.Errorf("nothing to update: pass --costs-hour or --clear-costs-hour")
	}

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	resp, err := client.Put(client.SpacePath("/stations/"+id), body)
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

	var station stationItem
	if err := cli.ReadJSON(resp, &station); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if stationJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(station)
	}

	if station.CostsHour != nil {
		fmt.Printf("Updated station: %s (costs/hour: %s)\n", station.Name, *station.CostsHour)
	} else {
		fmt.Printf("Updated station: %s (costs/hour: space default)\n", station.Name)
	}
	return nil
}
