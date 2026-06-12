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

// readyTask mirrors the relevant fields from dtos.TaskResponse for display.
type readyTask struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	StationID *string `json:"stationId,omitempty"`
	Priority  int     `json:"priority"`
}

// readyResponse mirrors the API response from GET /api/v1/tasks/ready.
type readyResponse struct {
	Items []readyTask `json:"items"`
	Total int         `json:"total"`
}

var readyJSONFlag bool

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "Show tasks ready for work",
	Long:  "List tasks that are ready to be worked on — unblocked, sorted by priority. Calls the Nori server's ready-work API.",
	RunE:  runReady,
}

func init() {
	readyCmd.Flags().BoolVar(&readyJSONFlag, "json", false, "Output as JSON")
	rootCmd.AddCommand(readyCmd)
}

func runReady(cmd *cobra.Command, args []string) error {
	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	resp, err := client.Get(client.SpacePath("/tasks/ready"))
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Handle 401
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

	var ready readyResponse
	if err := cli.ReadJSON(resp, &ready); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if readyJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(ready.Items)
	}

	if len(ready.Items) == 0 {
		fmt.Println("No tasks ready.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tSTATION\tPRIORITY")
	for _, t := range ready.Items {
		station := "-"
		if t.StationID != nil {
			station = *t.StationID
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", t.ID, t.Title, station, t.Priority)
	}
	return w.Flush()
}
