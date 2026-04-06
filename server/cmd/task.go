package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/tylerjvollick/nori/internal/cli"
)

// taskActionResponse mirrors the relevant fields from dtos.TaskResponse for display.
type taskActionResponse struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Status    string  `json:"status"`
	StationID *string `json:"stationId,omitempty"`
	Priority  int     `json:"priority"`
}

var taskJSONFlag bool

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
	Long:  "Task lifecycle commands: claim, complete, pause, resume, and skip tasks.",
}

var taskClaimCmd = &cobra.Command{
	Use:   "claim <id>",
	Short: "Claim a task",
	Long:  "Assign yourself to a task and set it to active. The task must be in open status and unassigned.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskClaim,
}

var taskCompleteCmd = &cobra.Command{
	Use:   "complete <id>",
	Short: "Complete a task",
	Long:  "Mark a task as done. Only the assigned user can complete it, and it must be in active status.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskComplete,
}

var taskPauseCmd = &cobra.Command{
	Use:   "pause <id>",
	Short: "Pause a task",
	Long:  "Pause an active task. Only the assigned user can pause it.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskPause,
}

var taskResumeCmd = &cobra.Command{
	Use:   "resume <id>",
	Short: "Resume a paused task",
	Long:  "Resume a paused task back to active status. Only the assigned user can resume it.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskResume,
}

var taskSkipCmd = &cobra.Command{
	Use:   "skip <id>",
	Short: "Skip a task",
	Long:  "Mark a task as skipped. Skipping triggers downstream readiness checks. Only the assigned user can skip it (if assigned).",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskSkip,
}

func init() {
	taskCmd.PersistentFlags().BoolVar(&taskJSONFlag, "json", false, "Output as JSON")
	taskCmd.AddCommand(taskClaimCmd)
	taskCmd.AddCommand(taskCompleteCmd)
	taskCmd.AddCommand(taskPauseCmd)
	taskCmd.AddCommand(taskResumeCmd)
	taskCmd.AddCommand(taskSkipCmd)
	rootCmd.AddCommand(taskCmd)
}

func runTaskClaim(cmd *cobra.Command, args []string) error {
	return runTaskAction(args[0], "claim")
}

func runTaskComplete(cmd *cobra.Command, args []string) error {
	return runTaskAction(args[0], "complete")
}

func runTaskPause(cmd *cobra.Command, args []string) error {
	return runTaskAction(args[0], "pause")
}

func runTaskResume(cmd *cobra.Command, args []string) error {
	return runTaskAction(args[0], "resume")
}

func runTaskSkip(cmd *cobra.Command, args []string) error {
	return runTaskAction(args[0], "skip")
}

// runTaskAction performs a POST to /api/v1/tasks/:id/<action> and prints the result.
func runTaskAction(taskID, action string) error {
	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	path := fmt.Sprintf("/api/v1/tasks/%s/%s", taskID, action)
	resp, err := client.Post(path, nil)
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

	var task taskActionResponse
	if err := cli.ReadJSON(resp, &task); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if taskJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(task)
	}

	fmt.Printf("Task %s: %s (status: %s)\n", task.ID, task.Title, task.Status)
	return nil
}
