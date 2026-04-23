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

// taskAddResponse mirrors the response from POST /api/v1/tasks/:id/children.
type taskAddResponse struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Status   string  `json:"status"`
	Type     string  `json:"type"`
	ParentID *string `json:"parentId,omitempty"`
}

// taskNoteResponse mirrors the response from POST /api/v1/tasks/:id/notes.
type taskNoteResponse struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	DeviationNotes *string `json:"deviationNotes,omitempty"`
}

// activeTaskResponse is used to find the user's current active task.
type activeTaskResponse struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Status   string  `json:"status"`
	ParentID *string `json:"parentId,omitempty"`
}

// activeTaskListResponse wraps the list response for active task lookup.
type activeTaskListResponse struct {
	Items []activeTaskResponse `json:"items"`
	Total int64                `json:"total"`
}

var (
	taskJSONFlag    bool
	taskAddStation  string
	taskAddAfter    string
	taskAddType     string
	taskAddParentID string
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
	Long:  "Task lifecycle commands: start, complete, pause, resume, skip, add, and note.",
}

var taskStartCmd = &cobra.Command{
	Use:   "start <id>",
	Short: "Start a task",
	Long:  "Start an open task, transitioning it to active status.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskStart,
}

var taskCompleteCmd = &cobra.Command{
	Use:   "complete <id>",
	Short: "Complete a task",
	Long:  "Mark a task as done. The task must be in active status.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskComplete,
}

var taskPauseCmd = &cobra.Command{
	Use:   "pause <id>",
	Short: "Pause a task",
	Long:  "Pause an active task.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskPause,
}

var taskResumeCmd = &cobra.Command{
	Use:   "resume <id>",
	Short: "Resume a paused task",
	Long:  "Resume a paused task back to active status.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskResume,
}

var taskSkipCmd = &cobra.Command{
	Use:   "skip <id>",
	Short: "Skip a task",
	Long:  "Mark a task as skipped. Skipping triggers downstream readiness checks.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskSkip,
}

var taskAddCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Add ad-hoc task to current job",
	Long:  "Add a new child task to the current job or a specified parent task. Supports first-time capture mode where operators add steps during execution.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskAdd,
}

var taskNoteCmd = &cobra.Command{
	Use:   "note <text>",
	Short: "Add deviation note to current task",
	Long:  "Add a deviation note to your current active task. Notes are appended if the task already has notes.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskNote,
}

func init() {
	taskCmd.PersistentFlags().BoolVar(&taskJSONFlag, "json", false, "Output as JSON")

	taskAddCmd.Flags().StringVar(&taskAddStation, "station", "", "Station name for the new task")
	taskAddCmd.Flags().StringVar(&taskAddAfter, "after", "", "Task ID to add as a dependency (new task runs after this one)")
	taskAddCmd.Flags().StringVar(&taskAddType, "type", "task", "Task type: task, job, recipe")
	taskAddCmd.Flags().StringVar(&taskAddParentID, "parent", "", "Parent task ID (defaults to current job)")

	taskCmd.AddCommand(taskStartCmd)
	taskCmd.AddCommand(taskCompleteCmd)
	taskCmd.AddCommand(taskPauseCmd)
	taskCmd.AddCommand(taskResumeCmd)
	taskCmd.AddCommand(taskSkipCmd)
	taskCmd.AddCommand(taskAddCmd)
	taskCmd.AddCommand(taskNoteCmd)
	rootCmd.AddCommand(taskCmd)
}

func runTaskStart(cmd *cobra.Command, args []string) error {
	return runTaskAction(args[0], "start")
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

// findActiveTask finds the user's current active task.
func findActiveTask(client *cli.Client, creds *cli.Credentials) (*activeTaskResponse, error) {
	path := fmt.Sprintf("/api/v1/tasks?status=active&assigneeId=%s&limit=1", creds.UserID)
	resp, err := client.Get(path)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := cli.Handle401(resp); err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("server error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var list activeTaskListResponse
	if err := cli.ReadJSON(resp, &list); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(list.Items) == 0 {
		return nil, fmt.Errorf("no active task found")
	}

	return &list.Items[0], nil
}

// resolveParentID determines the parent task ID for `task add`.
// Priority: --parent flag > infer from current active task's parent (job).
func resolveParentID(client *cli.Client, creds *cli.Credentials) (string, error) {
	if taskAddParentID != "" {
		return taskAddParentID, nil
	}

	// Find the user's current active task and use its parent (the job).
	active, err := findActiveTask(client, creds)
	if err != nil {
		return "", err
	}

	if active.ParentID != nil {
		return *active.ParentID, nil
	}

	// Active task has no parent — it's a root task (job) itself.
	return active.ID, nil
}

// resolveStationID looks up a station by name and returns its UUID.
func resolveStationID(client *cli.Client, name string) (string, error) {
	resp, err := client.Get("/api/v1/stations")
	if err != nil {
		return "", fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := cli.Handle401(resp); err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return "", fmt.Errorf("failed to list stations")
	}

	var stations []stationItem
	if err := cli.ReadJSON(resp, &stations); err != nil {
		return "", fmt.Errorf("failed to parse stations: %w", err)
	}

	for _, s := range stations {
		if s.Name == name {
			return s.ID, nil
		}
	}

	return "", fmt.Errorf("station %q not found", name)
}

// runTaskAdd implements `nori task add <title>`.
func runTaskAdd(cmd *cobra.Command, args []string) error {
	title := args[0]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	// Resolve the parent task (job).
	parentID, err := resolveParentID(client, creds)
	if err != nil {
		return err
	}

	// Build the request body.
	body := map[string]interface{}{
		"title": title,
		"type":  taskAddType,
	}

	// Resolve station name to ID if specified.
	if taskAddStation != "" {
		stationID, err := resolveStationID(client, taskAddStation)
		if err != nil {
			return err
		}
		body["stationId"] = stationID
	}

	// Add dependency if specified.
	if taskAddAfter != "" {
		body["afterId"] = taskAddAfter
	}

	// POST /api/v1/tasks/:parentId/children
	path := fmt.Sprintf("/api/v1/tasks/%s/children", parentID)
	resp, err := client.Post(path, body)
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

	var task taskAddResponse
	if err := cli.ReadJSON(resp, &task); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if taskJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(task)
	}

	fmt.Printf("Added %s: %s (parent: %s)\n", task.ID, task.Title, parentID)
	return nil
}

// runTaskNote implements `nori task note <text>`.
func runTaskNote(cmd *cobra.Command, args []string) error {
	text := args[0]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	// Find the user's current active task.
	active, err := findActiveTask(client, creds)
	if err != nil {
		return err
	}

	// POST /api/v1/tasks/:id/notes
	path := fmt.Sprintf("/api/v1/tasks/%s/notes", active.ID)
	resp, err := client.Post(path, map[string]string{
		"text": text,
	})
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

	var task taskNoteResponse
	if err := cli.ReadJSON(resp, &task); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if taskJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(task)
	}

	fmt.Printf("Note added to %s: %s\n", task.ID, task.Title)
	return nil
}
