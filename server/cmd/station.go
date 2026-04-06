package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"

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
}

var stationJSONFlag bool

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

func init() {
	stationCmd.PersistentFlags().BoolVar(&stationJSONFlag, "json", false, "Output as JSON")
	stationCmd.AddCommand(stationListCmd)
	rootCmd.AddCommand(stationCmd)
}

func runStationList(cmd *cobra.Command, args []string) error {
	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	resp, err := client.Get("/api/v1/stations")
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
