package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/tylerjvollick/nori/internal/cli"
)

// recipeListItem mirrors relevant fields from dtos.RecipeResponse for slug lookup.
type recipeListItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// recipeListResponse mirrors the paginated response from GET /api/v1/recipes.
type recipeListResponse struct {
	Items []recipeListItem `json:"items"`
	Total int64            `json:"total"`
}

// pourResponse mirrors the relevant fields from dtos.TaskResponse returned by POST /api/v1/recipes/:id/pour.
type pourResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// taskListResponse mirrors the paginated response from GET /api/v1/tasks for counting children.
type taskListResponse struct {
	Total int64 `json:"total"`
}

var (
	pourVarFlags  []string
	pourOrderFlag string
	pourJSONFlag  bool
)

var recipeCmd = &cobra.Command{
	Use:   "recipe",
	Short: "Manage recipes",
	Long:  "Recipe commands: list, show, pour, and manage recipe versions.",
}

var recipePourCmd = &cobra.Command{
	Use:   "pour <slug>",
	Short: "Pour a recipe into a new job",
	Long:  "Resolve a recipe by slug and pour it into a task graph, creating a new job with child tasks.",
	Args:  cobra.ExactArgs(1),
	RunE:  runRecipePour,
}

func init() {
	recipePourCmd.Flags().StringArrayVar(&pourVarFlags, "var", nil, "Variable override in key=value format (repeatable)")
	recipePourCmd.Flags().StringVar(&pourOrderFlag, "order", "", "Order ID to link to the job")
	recipePourCmd.Flags().BoolVar(&pourJSONFlag, "json", false, "Output as JSON")
	recipeCmd.AddCommand(recipePourCmd)
	rootCmd.AddCommand(recipeCmd)
}

func runRecipePour(cmd *cobra.Command, args []string) error {
	slug := args[0]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := cli.NewClient(creds)

	// 1. Resolve slug to recipe ID via GET /api/v1/recipes?slug=<slug>
	recipeID, err := resolveRecipeSlug(client, slug)
	if err != nil {
		return err
	}

	// 2. Build the pour request body.
	pourBody := map[string]interface{}{}

	// Parse --var key=value flags into a vars map.
	if len(pourVarFlags) > 0 {
		vars := make(map[string]string)
		for _, v := range pourVarFlags {
			parts := strings.SplitN(v, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid --var format %q — expected key=value", v)
			}
			vars[parts[0]] = parts[1]
		}
		pourBody["vars"] = vars
	}

	// Parse --order flag.
	if pourOrderFlag != "" {
		orderID, err := uuid.Parse(pourOrderFlag)
		if err != nil {
			return fmt.Errorf("invalid --order UUID: %w", err)
		}
		pourBody["orderId"] = orderID.String()
	}

	// 3. POST /api/v1/recipes/:id/pour
	path := fmt.Sprintf("/api/v1/recipes/%s/pour", recipeID)
	resp, err := client.Post(path, pourBody)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := cli.Handle401(resp); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("pour failed: %s", errResp.Error)
		}
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var job pourResponse
	if err := cli.ReadJSON(resp, &job); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// 4. Get task count by querying children of the job.
	taskCount, err := countChildTasks(client, job.ID)
	if err != nil {
		// Non-fatal: just print without count.
		taskCount = -1
	}

	if pourJSONFlag {
		output := map[string]interface{}{
			"jobId": job.ID,
			"title": job.Title,
		}
		if taskCount >= 0 {
			output["taskCount"] = taskCount
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	if taskCount >= 0 {
		fmt.Printf("Job %s created: %s\n  %d tasks\n", job.ID, job.Title, taskCount)
	} else {
		fmt.Printf("Job %s created: %s\n", job.ID, job.Title)
	}

	return nil
}

// resolveRecipeSlug looks up a recipe by slug via the list endpoint and returns its ID.
func resolveRecipeSlug(client *cli.Client, slug string) (string, error) {
	path := fmt.Sprintf("/api/v1/recipes?slug=%s&limit=1", slug)
	resp, err := client.Get(path)
	if err != nil {
		return "", fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := cli.Handle401(resp); err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return "", fmt.Errorf("server error: %s", errResp.Error)
		}
		return "", fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var list recipeListResponse
	if err := cli.ReadJSON(resp, &list); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(list.Items) == 0 {
		return "", fmt.Errorf("recipe %q not found", slug)
	}

	return list.Items[0].ID, nil
}

// countChildTasks queries the tasks list endpoint to count children of a job.
func countChildTasks(client *cli.Client, parentID string) (int64, error) {
	path := fmt.Sprintf("/api/v1/tasks?parentId=%s&limit=1", parentID)
	resp, err := client.Get(path)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return 0, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var list taskListResponse
	if err := cli.ReadJSON(resp, &list); err != nil {
		return 0, err
	}

	return list.Total, nil
}
