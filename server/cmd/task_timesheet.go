package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/tylerjvollick/nori/internal/cli"
)

// --- Response types ---

type timeEntryResponse struct {
	ID           string  `json:"id"`
	TaskID      string  `json:"taskId"`
	SpaceID     string  `json:"spaceId"`
	LoggedByID  string  `json:"loggedById"`
	StartedAt   string  `json:"startedAt"`
	EndedAt     *string `json:"endedAt,omitempty"`
	ElapsedSecs int     `json:"elapsedSecs"`
	IsPaused    bool    `json:"isPaused"`
	Notes       *string `json:"notes,omitempty"`
}

type timeEntryListResponse struct {
	Items             []timeEntryResponse `json:"items"`
	TotalDurationSecs int                 `json:"totalDurationSecs"`
}

type unloggedTaskResponse struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type unloggedListResponse struct {
	Items []unloggedTaskResponse `json:"items"`
}

// --- Flags ---

var (
	timesheetAddStart string
	timesheetAddEnd   string
	timesheetAddNotes string

	timesheetEditStart string
	timesheetEditEnd   string
	timesheetEditNotes string
)

// --- Commands ---

var taskTimesheetCmd = &cobra.Command{
	Use:   "timesheet <taskId>",
	Short: "List time entries for a task",
	Long:  "Display a table of all time entries recorded against a task.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTimesheetList,
}

var timesheetAddCmd = &cobra.Command{
	Use:   "add <taskId>",
	Short: "Add a manual time entry",
	Long:  "Create a manual time entry with explicit start and end times. Use when you forgot to start the timer.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTimesheetAdd,
}

var timesheetEditCmd = &cobra.Command{
	Use:   "edit <taskId> <entryId>",
	Short: "Edit a time entry",
	Long:  "Update the start, end, or notes of an existing time entry.",
	Args:  cobra.ExactArgs(2),
	RunE:  runTimesheetEdit,
}

var timesheetDeleteCmd = &cobra.Command{
	Use:   "delete <taskId> <entryId>",
	Short: "Delete a time entry",
	Long:  "Remove a time entry from a task.",
	Args:  cobra.ExactArgs(2),
	RunE:  runTimesheetDelete,
}

var taskUnloggedCmd = &cobra.Command{
	Use:   "unlogged",
	Short: "List completed tasks with no time entries",
	Long:  "Show completed tasks that have no time entries recorded. Useful for manager review and catching missed time logging.",
	Args:  cobra.NoArgs,
	RunE:  runUnlogged,
}

func init() {
	timesheetAddCmd.Flags().StringVar(&timesheetAddStart, "start", "", "Start time (e.g., '2026-04-25 14:00')")
	timesheetAddCmd.Flags().StringVar(&timesheetAddEnd, "end", "", "End time (e.g., '2026-04-25 15:00')")
	timesheetAddCmd.Flags().StringVar(&timesheetAddNotes, "notes", "", "Optional notes")
	_ = timesheetAddCmd.MarkFlagRequired("start")
	_ = timesheetAddCmd.MarkFlagRequired("end")

	timesheetEditCmd.Flags().StringVar(&timesheetEditStart, "start", "", "New start time")
	timesheetEditCmd.Flags().StringVar(&timesheetEditEnd, "end", "", "New end time")
	timesheetEditCmd.Flags().StringVar(&timesheetEditNotes, "notes", "", "New notes")

	taskTimesheetCmd.AddCommand(timesheetAddCmd)
	taskTimesheetCmd.AddCommand(timesheetEditCmd)
	taskTimesheetCmd.AddCommand(timesheetDeleteCmd)
	taskCmd.AddCommand(taskTimesheetCmd)
	taskCmd.AddCommand(taskUnloggedCmd)
}

// --- Implementations ---

func runTimesheetList(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}
	client := newClientWithSpace(creds)

	path := client.SpacePath(fmt.Sprintf("/tasks/%s/time-entries", taskID))
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

	var list timeEntryListResponse
	if err := cli.ReadJSON(resp, &list); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if taskJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	}

	if len(list.Items) == 0 {
		fmt.Println("No time entries recorded for this task.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "#\tSTART\tEND\tDURATION\tNOTES\n")
	for i, entry := range list.Items {
		start := formatEntryTime(entry.StartedAt)
		end := "--"
		if entry.EndedAt != nil {
			end = formatEntryTime(*entry.EndedAt)
		} else {
			end = "(running)"
		}
		dur := formatDurationCLI(entry.ElapsedSecs)
		if entry.EndedAt == nil && !entry.IsPaused {
			// Running entry — add live delta.
			t, parseErr := time.Parse(time.RFC3339, entry.StartedAt)
			if parseErr == nil {
				live := entry.ElapsedSecs + int(time.Since(t).Seconds())
				dur = formatDurationCLI(live) + " (live)"
			}
		} else if entry.IsPaused {
			dur += " (paused)"
		}
		notes := ""
		if entry.Notes != nil {
			notes = *entry.Notes
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", i+1, start, end, dur, notes)
	}
	w.Flush()

	fmt.Printf("\nTotal: %s\n", formatDurationCLI(list.TotalDurationSecs))
	return nil
}

func runTimesheetAdd(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	startTime, err := parseFlexibleTime(timesheetAddStart)
	if err != nil {
		return fmt.Errorf("invalid --start time: %w", err)
	}
	endTime, err := parseFlexibleTime(timesheetAddEnd)
	if err != nil {
		return fmt.Errorf("invalid --end time: %w", err)
	}

	body := map[string]interface{}{
		"startedAt": startTime.Format(time.RFC3339),
		"endedAt":   endTime.Format(time.RFC3339),
	}
	if timesheetAddNotes != "" {
		body["notes"] = timesheetAddNotes
	}

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}
	client := newClientWithSpace(creds)

	path := client.SpacePath(fmt.Sprintf("/tasks/%s/time-entries", taskID))
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

	var entry timeEntryResponse
	if err := cli.ReadJSON(resp, &entry); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if taskJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(entry)
	}

	dur := formatDurationCLI(entry.ElapsedSecs)
	fmt.Printf("Added time entry: %s – %s (%s)\n", formatEntryTime(entry.StartedAt), formatEntryTime(*entry.EndedAt), dur)
	return nil
}

func runTimesheetEdit(cmd *cobra.Command, args []string) error {
	taskID := args[0]
	entryID := args[1]

	body := map[string]interface{}{}

	if timesheetEditStart != "" {
		t, err := parseFlexibleTime(timesheetEditStart)
		if err != nil {
			return fmt.Errorf("invalid --start time: %w", err)
		}
		body["startedAt"] = t.Format(time.RFC3339)
	}
	if timesheetEditEnd != "" {
		t, err := parseFlexibleTime(timesheetEditEnd)
		if err != nil {
			return fmt.Errorf("invalid --end time: %w", err)
		}
		body["endedAt"] = t.Format(time.RFC3339)
	}
	if timesheetEditNotes != "" {
		body["notes"] = timesheetEditNotes
	}

	if len(body) == 0 {
		return fmt.Errorf("no fields to update (use --start, --end, or --notes)")
	}

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}
	client := newClientWithSpace(creds)

	path := client.SpacePath(fmt.Sprintf("/tasks/%s/time-entries/%s", taskID, entryID))
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

	var entry timeEntryResponse
	if err := cli.ReadJSON(resp, &entry); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if taskJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(entry)
	}

	fmt.Printf("Updated time entry %s\n", entry.ID)
	return nil
}

func runTimesheetDelete(cmd *cobra.Command, args []string) error {
	taskID := args[0]
	entryID := args[1]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}
	client := newClientWithSpace(creds)

	path := client.SpacePath(fmt.Sprintf("/tasks/%s/time-entries/%s", taskID, entryID))
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

	fmt.Printf("Deleted time entry %s\n", entryID)
	return nil
}

func runUnlogged(cmd *cobra.Command, args []string) error {
	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}
	client := newClientWithSpace(creds)

	path := client.SpacePath("/tasks/unlogged")
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

	var list unloggedListResponse
	if err := cli.ReadJSON(resp, &list); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if taskJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	}

	if len(list.Items) == 0 {
		fmt.Println("All completed tasks have time entries.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tTITLE\tSTATUS\n")
	for _, task := range list.Items {
		fmt.Fprintf(w, "%s\t%s\t%s\n", task.ID, task.Title, task.Status)
	}
	w.Flush()

	fmt.Printf("\n%d completed tasks without time entries.\n", len(list.Items))
	return nil
}

// --- Helpers ---

// formatDurationCLI formats seconds as "Xh Ym" or "Ym" or "Xs".
func formatDurationCLI(seconds int) string {
	if seconds <= 0 {
		return "--"
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%ds", seconds)
}

// formatEntryTime formats an RFC3339 timestamp for tabular display.
func formatEntryTime(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Local().Format("Jan 02 15:04")
}

// parseFlexibleTime parses a time string in common formats.
func parseFlexibleTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"Jan 02 15:04",
		"01/02 15:04",
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, s, time.Now().Location()); err == nil {
			// For formats without year, set to current year.
			if t.Year() == 0 {
				t = t.AddDate(time.Now().Year(), 0, 0)
			}
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized time format %q (try '2026-04-25 14:00')", s)
}
