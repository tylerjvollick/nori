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

// jobItem mirrors the relevant fields from dtos.TaskResponse for job display.
type jobItem struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Status   string  `json:"status"`
	Priority int     `json:"priority"`
	DueDate  *string `json:"dueDate,omitempty"`
}

// jobListResponse mirrors the paginated response from GET /api/v1/tasks.
type jobListResponse struct {
	Items  []jobItem `json:"items"`
	Total  int64     `json:"total"`
	Offset int       `json:"offset"`
	Limit  int       `json:"limit"`
}

// jobDetail is a richer view of a job for the show command.
type jobDetail struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Status      string      `json:"status"`
	Type        string      `json:"type"`
	Priority    int         `json:"priority"`
	DueDate     *string     `json:"dueDate,omitempty"`
	Description *string     `json:"description,omitempty"`
	CreatedAt   string      `json:"createdAt"`
	Children    []taskChild `json:"children"`
}

// taskChild represents a child task within a job's task tree.
type taskChild struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Type     string `json:"type"`
	Priority int    `json:"priority"`
}

var (
	jobJSONFlag   bool
	jobStatusFlag string
)

var jobCmd = &cobra.Command{
	Use:   "job",
	Short: "Manage jobs",
	Long:  "Job commands: list and inspect jobs (root tasks of type 'job').",
}

var jobListCmd = &cobra.Command{
	Use:   "list",
	Short: "List jobs",
	Long:  "List root tasks of type 'job', optionally filtered by status.",
	RunE:  runJobList,
}

var jobShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show job details with task tree",
	Long:  "Display a job with its full task tree, statuses, and dependencies.",
	Args:  cobra.ExactArgs(1),
	RunE:  runJobShow,
}

func init() {
	jobCmd.PersistentFlags().BoolVar(&jobJSONFlag, "json", false, "Output as JSON")
	jobListCmd.Flags().StringVar(&jobStatusFlag, "status", "", "Filter by status (open, active, paused, done, skipped, cancelled)")
	jobCmd.AddCommand(jobListCmd)
	jobCmd.AddCommand(jobShowCmd)
	rootCmd.AddCommand(jobCmd)
}

func runJobList(cmd *cobra.Command, args []string) error {
	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	path := "/api/v1/tasks?type=job"
	if jobStatusFlag != "" {
		path += "&status=" + jobStatusFlag
	}

	resp, err := client.Get(path)
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

	var list jobListResponse
	if err := cli.ReadJSON(resp, &list); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if jobJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(list.Items)
	}

	if len(list.Items) == 0 {
		fmt.Println("No jobs found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tSTATUS\tPRIORITY")
	for _, j := range list.Items {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", j.ID, j.Title, j.Status, j.Priority)
	}
	return w.Flush()
}

func runJobShow(cmd *cobra.Command, args []string) error {
	jobID := args[0]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	// 1. Fetch the job itself.
	jobPath := fmt.Sprintf("/api/v1/tasks/%s", jobID)
	resp, err := client.Get(jobPath)
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

	var job jobDetail
	if err := cli.ReadJSON(resp, &job); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// 2. Fetch children (task tree).
	childrenPath := fmt.Sprintf("/api/v1/tasks?parentId=%s&limit=200", jobID)
	childResp, err := client.Get(childrenPath)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if childResp.StatusCode == http.StatusOK {
		var childList struct {
			Items []taskChild `json:"items"`
			Total int64       `json:"total"`
		}
		if err := cli.ReadJSON(childResp, &childList); err == nil {
			job.Children = childList.Items
		}
	} else {
		childResp.Body.Close()
	}

	if jobJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(job)
	}

	// Human-readable output.
	fmt.Printf("Job %s: %s\n", job.ID, job.Title)
	fmt.Printf("  Status:   %s\n", job.Status)
	fmt.Printf("  Priority: %d\n", job.Priority)
	if job.DueDate != nil {
		fmt.Printf("  Due:      %s\n", *job.DueDate)
	}
	if job.Description != nil {
		fmt.Printf("  Description: %s\n", *job.Description)
	}

	if len(job.Children) == 0 {
		fmt.Println("  No child tasks.")
		return nil
	}

	fmt.Printf("\n  Tasks (%d):\n", len(job.Children))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  ID\tTITLE\tSTATUS\tTYPE")
	for _, c := range job.Children {
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", c.ID, c.Title, c.Status, c.Type)
	}
	return w.Flush()
}
