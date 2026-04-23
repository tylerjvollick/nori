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

// ── Flags ───────────────────────────────────────────────────────────

var (
	taskUpdateTitle       string
	taskUpdateDescription string
	taskUpdateStation     string
	taskUpdatePriority    int
	taskUpdatePrioritySet bool

	taskShowJSONFlag bool

	taskDepJSONFlag bool
)

// ── Commands ────────────────────────────────────────────────────────

var taskShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show task details",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskShow,
}

var taskUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a task",
	Long:  "Update a task's title, description, station, or priority.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskUpdate,
}

var taskDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskDelete,
}

var taskTreeCmd = &cobra.Command{
	Use:   "tree <id>",
	Short: "Show task tree",
	Long:  "Display the full task tree rooted at the given task.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskTree,
}

// ── Dep subcommands ─────────────────────────────────────────────────

var taskDepCmd = &cobra.Command{
	Use:   "dep",
	Short: "Manage task dependencies",
}

var taskDepListCmd = &cobra.Command{
	Use:   "list <task-id>",
	Short: "List dependencies for a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskDepList,
}

var taskDepAddCmd = &cobra.Command{
	Use:   "add <blocker-id> <target-id>",
	Short: "Add dependency: blocker blocks target",
	Long:  "Create a dependency where target-id cannot start until blocker-id is complete.",
	Args:  cobra.ExactArgs(2),
	RunE:  runTaskDepAdd,
}

var taskDepRemoveCmd = &cobra.Command{
	Use:   "remove <task-id> <dep-id>",
	Short: "Remove a dependency by its UUID",
	Args:  cobra.ExactArgs(2),
	RunE:  runTaskDepRemove,
}

func init() {
	taskShowCmd.Flags().BoolVar(&taskShowJSONFlag, "json", false, "Output as JSON")

	taskUpdateCmd.Flags().StringVar(&taskUpdateTitle, "title", "", "New title")
	taskUpdateCmd.Flags().StringVar(&taskUpdateDescription, "description", "", "New description")
	taskUpdateCmd.Flags().StringVar(&taskUpdateStation, "station", "", "Station name")
	taskUpdateCmd.Flags().IntVar(&taskUpdatePriority, "priority", 0, "Priority (0-4)")

	taskTreeCmd.Flags().BoolVar(&taskShowJSONFlag, "json", false, "Output as JSON")

	taskDepCmd.PersistentFlags().BoolVar(&taskDepJSONFlag, "json", false, "Output as JSON")

	taskDepCmd.AddCommand(taskDepListCmd)
	taskDepCmd.AddCommand(taskDepAddCmd)
	taskDepCmd.AddCommand(taskDepRemoveCmd)

	taskCmd.AddCommand(taskShowCmd)
	taskCmd.AddCommand(taskUpdateCmd)
	taskCmd.AddCommand(taskDeleteCmd)
	taskCmd.AddCommand(taskTreeCmd)
	taskCmd.AddCommand(taskDepCmd)
}

// ── Implementations ─────────────────────────────────────────────────

// cliTaskDetail is a richer view of a task for the show command.
type cliTaskDetail struct {
	ID                string  `json:"id"`
	Title             string  `json:"title"`
	Type              string  `json:"type"`
	Status            string  `json:"status"`
	Priority          int     `json:"priority"`
	ParentID          *string `json:"parentId,omitempty"`
	StationID         *string `json:"stationId,omitempty"`
	Description       *string `json:"description,omitempty"`
	EstimatedTimeSecs *int    `json:"estimatedTimeSeconds,omitempty"`
	ActualTimeSecs    int     `json:"actualTimeSeconds"`
	DisplayOrder      int     `json:"displayOrder"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
}

func runTaskShow(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	path := client.SpacePath(fmt.Sprintf("/tasks/%s", taskID))
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

	var task cliTaskDetail
	if err := cli.ReadJSON(resp, &task); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if taskShowJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(task)
	}

	fmt.Printf("Task: %s [%s]\n", task.Title, task.ID)
	fmt.Printf("  Type: %s  Status: %s  Priority: P%d\n", task.Type, task.Status, task.Priority)
	if task.ParentID != nil {
		fmt.Printf("  Parent: %s\n", *task.ParentID)
	}
	if task.Description != nil {
		fmt.Printf("  Description: %s\n", *task.Description)
	}
	if task.EstimatedTimeSecs != nil {
		fmt.Printf("  Estimated: %ds\n", *task.EstimatedTimeSecs)
	}
	if task.ActualTimeSecs > 0 {
		fmt.Printf("  Actual: %ds\n", task.ActualTimeSecs)
	}

	return nil
}

func runTaskUpdate(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	body := map[string]interface{}{}

	if cmd.Flags().Changed("title") {
		body["title"] = taskUpdateTitle
	}
	if cmd.Flags().Changed("description") {
		body["description"] = taskUpdateDescription
	}
	if cmd.Flags().Changed("station") {
		stationID, err := resolveStationID(client, taskUpdateStation)
		if err != nil {
			return err
		}
		body["stationId"] = stationID
	}
	if cmd.Flags().Changed("priority") {
		body["priority"] = taskUpdatePriority
	}

	if len(body) == 0 {
		return fmt.Errorf("no fields to update — use --title, --description, --station, or --priority")
	}

	path := client.SpacePath(fmt.Sprintf("/tasks/%s", taskID))
	resp, err := client.Put(path, body)
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

	var task taskActionResponse
	if err := cli.ReadJSON(resp, &task); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if taskJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(task)
	}

	fmt.Printf("Updated %s: %s\n", task.ID, task.Title)
	return nil
}

func runTaskDelete(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	path := client.SpacePath(fmt.Sprintf("/tasks/%s", taskID))
	resp, err := client.Delete(path)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := cli.Handle401(resp); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("server error: %s", errResp.Error)
		}
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	resp.Body.Close()

	fmt.Printf("Deleted task %s\n", taskID)
	return nil
}

func runTaskTree(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	path := client.SpacePath(fmt.Sprintf("/tasks/%s/tree", taskID))
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

	var tree cliTaskTree
	if err := cli.ReadJSON(resp, &tree); err != nil {
		return fmt.Errorf("failed to parse task tree: %w", err)
	}

	if taskShowJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(tree)
	}

	allTasks := flattenTree(&tree)
	depMap := fetchAllDeps(client, allTasks)
	printTaskTree(&tree, "", true, depMap)

	return nil
}

// ── Dep implementations ─────────────────────────────────────────────

type cliDepEdge struct {
	ID         string `json:"id"`
	FromTaskID string `json:"fromTaskId"`
	ToTaskID   string `json:"toTaskId"`
	Type       string `json:"type"`
}

func runTaskDepList(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	path := client.SpacePath(fmt.Sprintf("/tasks/%s/deps", taskID))
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

	var deps struct {
		Blockers   []cliDepEdge `json:"blockers"`
		Dependents []cliDepEdge `json:"dependents"`
	}
	if err := cli.ReadJSON(resp, &deps); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if taskDepJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(deps)
	}

	if len(deps.Blockers) == 0 && len(deps.Dependents) == 0 {
		fmt.Printf("Task %s has no dependencies.\n", taskID)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if len(deps.Blockers) > 0 {
		fmt.Fprintf(w, "BLOCKED BY:\n")
		fmt.Fprintf(w, "  DEP ID\tBLOCKER TASK\tTYPE\n")
		for _, b := range deps.Blockers {
			fmt.Fprintf(w, "  %s\t%s\t%s\n", b.ID, b.ToTaskID, b.Type)
		}
	}
	if len(deps.Dependents) > 0 {
		fmt.Fprintf(w, "BLOCKS:\n")
		fmt.Fprintf(w, "  DEP ID\tDEPENDENT TASK\tTYPE\n")
		for _, d := range deps.Dependents {
			fmt.Fprintf(w, "  %s\t%s\t%s\n", d.ID, d.FromTaskID, d.Type)
		}
	}
	return w.Flush()
}

func runTaskDepAdd(cmd *cobra.Command, args []string) error {
	blockerID := args[0]
	targetID := args[1]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	// POST /api/v1/spaces/:spaceId/tasks/:blockerID/deps — blocker blocks target.
	body := map[string]interface{}{
		"targetTaskId": targetID,
		"type":         "blocks",
	}

	path := client.SpacePath(fmt.Sprintf("/tasks/%s/deps", blockerID))
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

	var dep cliDepEdge
	if err := cli.ReadJSON(resp, &dep); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if taskDepJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(dep)
	}

	fmt.Printf("Dependency added: %s blocks %s (dep: %s)\n", blockerID, targetID, dep.ID)
	return nil
}

func runTaskDepRemove(cmd *cobra.Command, args []string) error {
	taskID := args[0]
	depID := args[1]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	path := client.SpacePath(fmt.Sprintf("/tasks/%s/deps/%s", taskID, depID))
	resp, err := client.Delete(path)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := cli.Handle401(resp); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("server error: %s", errResp.Error)
		}
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	resp.Body.Close()

	fmt.Printf("Dependency %s removed\n", depID)
	return nil
}
