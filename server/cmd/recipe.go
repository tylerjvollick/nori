package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/BurntSushi/toml"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/tylerjvollick/nori/internal/cli"
)

// recipeListItem mirrors relevant fields from dtos.RecipeResponse for slug lookup.
type recipeListItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	IsActive bool   `json:"isActive"`
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

// createRecipeResponse mirrors relevant fields from dtos.RecipeResponse for create.
type createRecipeResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// createVersionResponse mirrors relevant fields from dtos.RecipeVersionResponse for create.
type createVersionResponse struct {
	ID            int    `json:"id"`
	VersionNumber int    `json:"versionNumber"`
	Status        string `json:"status"`
}

// versionListItem mirrors relevant fields from dtos.RecipeVersionResponse for version listing.
type versionListItem struct {
	ID            int    `json:"id"`
	VersionNumber int    `json:"versionNumber"`
	Status        string `json:"status"`
}

// versionListResponse mirrors the response from GET /api/v1/recipes/:id/versions.
type versionListResponse struct {
	Items []versionListItem `json:"items"`
	Total int               `json:"total"`
}

// recipeShowDetail is a richer view of a recipe for the show command.
type recipeShowDetail struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	Slug             string             `json:"slug"`
	Description      *string            `json:"description,omitempty"`
	IsActive         bool               `json:"isActive"`
	CurrentVersionID *int               `json:"currentVersionId,omitempty"`
	CreatedAt        string             `json:"createdAt"`
	UpdatedAt        string             `json:"updatedAt"`
	CurrentVersion   *recipeShowVersion `json:"currentVersion,omitempty"`
	Steps            []recipeShowStep   `json:"steps,omitempty"`
}

// recipeShowVersion mirrors relevant fields from dtos.RecipeVersionResponse for show.
type recipeShowVersion struct {
	ID            int     `json:"id"`
	VersionNumber int     `json:"versionNumber"`
	Status        string  `json:"status"`
	Content       string  `json:"content"`
	ChangeSummary *string `json:"changeSummary,omitempty"`
	PublishedAt   *string `json:"publishedAt,omitempty"`
}

// recipeShowStep represents a step extracted from the recipe TOML content.
type recipeShowStep struct {
	ID    string `json:"id" toml:"id"`
	Title string `json:"title" toml:"title"`
}

// recipeShowTOML is a minimal struct for extracting steps from the recipe TOML content.
type recipeShowTOML struct {
	Steps []recipeShowStep `toml:"steps"`
}

var (
	pourVarFlags  []string
	pourOrderFlag string
	pourJSONFlag  bool

	createFromTOMLFlag string
	createNameFlag     string
	createJSONFlag     bool

	publishJSONFlag bool

	listJSONFlag   bool
	listActiveFlag string

	showJSONFlag bool
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

var recipeCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new recipe from a TOML file",
	Long:  "Import a TOML file as a new recipe. Creates the recipe, adds version 1, and auto-publishes it.",
	RunE:  runRecipeCreate,
}

var recipeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recipes",
	Long:  "List recipes in the current space, optionally filtered by active status.",
	RunE:  runRecipeList,
}

var recipeShowCmd = &cobra.Command{
	Use:   "show <slug>",
	Short: "Display a recipe with its current published version",
	Long:  "Resolve a recipe by slug and display its details, including the step tree from the current published version.",
	Args:  cobra.ExactArgs(1),
	RunE:  runRecipeShow,
}

var recipePublishCmd = &cobra.Command{
	Use:   "publish <slug>",
	Short: "Publish the latest draft version of a recipe",
	Long:  "Resolve a recipe by slug, find its latest draft version, and publish it.",
	Args:  cobra.ExactArgs(1),
	RunE:  runRecipePublish,
}

func init() {
	recipePourCmd.Flags().StringArrayVar(&pourVarFlags, "var", nil, "Variable override in key=value format (repeatable)")
	recipePourCmd.Flags().StringVar(&pourOrderFlag, "order", "", "Order ID to link to the job")
	recipePourCmd.Flags().BoolVar(&pourJSONFlag, "json", false, "Output as JSON")

	recipeCreateCmd.Flags().StringVar(&createFromTOMLFlag, "from-toml", "", "Path to TOML file to import (required)")
	recipeCreateCmd.Flags().StringVar(&createNameFlag, "name", "", "Recipe name (defaults to 'formula' field from TOML)")
	recipeCreateCmd.Flags().BoolVar(&createJSONFlag, "json", false, "Output as JSON")
	recipeCreateCmd.MarkFlagRequired("from-toml")

	recipePublishCmd.Flags().BoolVar(&publishJSONFlag, "json", false, "Output as JSON")

	recipeListCmd.Flags().BoolVar(&listJSONFlag, "json", false, "Output as JSON")
	recipeListCmd.Flags().StringVar(&listActiveFlag, "active", "", "Filter by active status (true/false)")

	recipeShowCmd.Flags().BoolVar(&showJSONFlag, "json", false, "Output as JSON")

	recipeCmd.AddCommand(recipeListCmd)
	recipeCmd.AddCommand(recipeShowCmd)
	recipeCmd.AddCommand(recipePourCmd)
	recipeCmd.AddCommand(recipeCreateCmd)
	recipeCmd.AddCommand(recipePublishCmd)
	rootCmd.AddCommand(recipeCmd)
}

func runRecipeList(cmd *cobra.Command, args []string) error {
	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	path := "/api/v1/recipes"
	if listActiveFlag != "" {
		path += "?isActive=" + listActiveFlag
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

	var list recipeListResponse
	if err := cli.ReadJSON(resp, &list); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if listJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(list.Items)
	}

	if len(list.Items) == 0 {
		fmt.Println("No recipes found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SLUG\tNAME\tACTIVE")
	for _, r := range list.Items {
		fmt.Fprintf(w, "%s\t%s\t%v\n", r.Slug, r.Name, r.IsActive)
	}
	return w.Flush()
}

func runRecipeShow(cmd *cobra.Command, args []string) error {
	slug := args[0]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	// 1. Resolve slug to get recipe details via GET /api/v1/recipes?slug=<slug>&limit=1
	path := fmt.Sprintf("/api/v1/recipes?slug=%s&limit=1", slug)
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

	var list struct {
		Items []recipeShowDetail `json:"items"`
		Total int64              `json:"total"`
	}
	if err := cli.ReadJSON(resp, &list); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(list.Items) == 0 {
		return fmt.Errorf("recipe %q not found", slug)
	}

	recipe := list.Items[0]

	// 2. Fetch the published version content via GET /api/v1/recipes/:id/versions.
	versionsPath := fmt.Sprintf("/api/v1/recipes/%s/versions", recipe.ID)
	versionResp, err := client.Get(versionsPath)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if versionResp.StatusCode == http.StatusOK {
		var versions struct {
			Items []recipeShowVersion `json:"items"`
			Total int                 `json:"total"`
		}
		if err := cli.ReadJSON(versionResp, &versions); err == nil {
			// Find the published version (versions ordered by version_number DESC).
			for i := range versions.Items {
				if versions.Items[i].Status == "published" {
					recipe.CurrentVersion = &versions.Items[i]
					break
				}
			}
		}
	} else {
		versionResp.Body.Close()
	}

	// 3. Extract steps from the TOML content if we have a published version.
	if recipe.CurrentVersion != nil && recipe.CurrentVersion.Content != "" {
		var parsed recipeShowTOML
		if err := toml.Unmarshal([]byte(recipe.CurrentVersion.Content), &parsed); err == nil {
			recipe.Steps = parsed.Steps
		}
	}

	if showJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(recipe)
	}

	// Human-readable output.
	fmt.Printf("Recipe: %s (%s)\n", recipe.Name, recipe.Slug)
	fmt.Printf("  Active: %v\n", recipe.IsActive)
	if recipe.Description != nil {
		fmt.Printf("  Description: %s\n", *recipe.Description)
	}

	if recipe.CurrentVersion != nil {
		fmt.Printf("  Published Version: %d (%s)\n", recipe.CurrentVersion.VersionNumber, recipe.CurrentVersion.Status)
		if recipe.CurrentVersion.ChangeSummary != nil {
			fmt.Printf("  Change Summary: %s\n", *recipe.CurrentVersion.ChangeSummary)
		}
	} else {
		fmt.Println("  No published version.")
	}

	if len(recipe.Steps) > 0 {
		fmt.Printf("\n  Steps (%d):\n", len(recipe.Steps))
		for i, s := range recipe.Steps {
			fmt.Printf("  %d. [%s] %s\n", i+1, s.ID, s.Title)
		}
	}

	return nil
}

func runRecipePour(cmd *cobra.Command, args []string) error {
	slug := args[0]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

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

// tomlFormula is a minimal struct to extract the recipe name from a TOML file's "formula" key.
type tomlFormula struct {
	Formula string `toml:"formula"`
}

func runRecipeCreate(cmd *cobra.Command, args []string) error {
	// 1. Read and validate the TOML file.
	tomlPath := createFromTOMLFlag
	if tomlPath == "" {
		return fmt.Errorf("--from-toml is required")
	}

	// Resolve relative paths.
	if !filepath.IsAbs(tomlPath) {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to resolve working directory: %w", err)
		}
		tomlPath = filepath.Join(wd, tomlPath)
	}

	data, err := os.ReadFile(tomlPath)
	if err != nil {
		return fmt.Errorf("failed to read TOML file: %w", err)
	}

	content := string(data)
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("TOML file is empty")
	}

	// 2. Determine the recipe name: --name flag takes precedence, else extract from TOML.
	name := createNameFlag
	if name == "" {
		var f tomlFormula
		if err := toml.Unmarshal(data, &f); err == nil && f.Formula != "" {
			name = f.Formula
		}
	}
	if name == "" {
		return fmt.Errorf("could not determine recipe name — set 'formula' in the TOML or use --name")
	}

	// 3. Connect to the server.
	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	// 4. POST /api/v1/recipes — create the recipe.
	createBody := map[string]interface{}{
		"name": name,
	}

	resp, err := client.Post("/api/v1/recipes", createBody)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := cli.Handle401(resp); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("create recipe failed: %s", errResp.Error)
		}
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var recipe createRecipeResponse
	if err := cli.ReadJSON(resp, &recipe); err != nil {
		return fmt.Errorf("failed to parse recipe response: %w", err)
	}

	// 5. POST /api/v1/recipes/:id/versions — create version 1 with the TOML content.
	versionBody := map[string]interface{}{
		"content":       content,
		"changeSummary": "Initial import from TOML file",
	}

	versionPath := fmt.Sprintf("/api/v1/recipes/%s/versions", recipe.ID)
	resp, err = client.Post(versionPath, versionBody)
	if err != nil {
		return fmt.Errorf("recipe created but failed to create version: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("recipe created but version creation failed: %s", errResp.Error)
		}
		return fmt.Errorf("recipe created but version creation returned status %d", resp.StatusCode)
	}

	var version createVersionResponse
	if err := cli.ReadJSON(resp, &version); err != nil {
		return fmt.Errorf("failed to parse version response: %w", err)
	}

	// 6. POST /api/v1/recipes/:id/versions/:vid/publish — auto-publish the first version.
	publishPath := fmt.Sprintf("/api/v1/recipes/%s/versions/%d/publish", recipe.ID, version.ID)
	resp, err = client.Post(publishPath, nil)
	if err != nil {
		return fmt.Errorf("recipe and version created but publish failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("recipe and version created but publish failed: %s", errResp.Error)
		}
		return fmt.Errorf("recipe and version created but publish returned status %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 7. Output result.
	if createJSONFlag {
		output := map[string]interface{}{
			"recipeId": recipe.ID,
			"name":     recipe.Name,
			"slug":     recipe.Slug,
			"version":  version.VersionNumber,
			"status":   "published",
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	fmt.Printf("Recipe %q created (slug: %s)\n  Version %d published\n", recipe.Name, recipe.Slug, version.VersionNumber)
	return nil
}

// publishVersionResponse mirrors relevant fields from dtos.RecipeVersionResponse for publish.
type publishVersionResponse struct {
	ID            int    `json:"id"`
	VersionNumber int    `json:"versionNumber"`
	Status        string `json:"status"`
}

func runRecipePublish(cmd *cobra.Command, args []string) error {
	slug := args[0]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	// 1. Resolve slug to recipe ID.
	recipeID, err := resolveRecipeSlug(client, slug)
	if err != nil {
		return err
	}

	// 2. List versions to find the latest draft.
	versionsPath := fmt.Sprintf("/api/v1/recipes/%s/versions", recipeID)
	resp, err := client.Get(versionsPath)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := cli.Handle401(resp); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("failed to list versions: %s", errResp.Error)
		}
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var versions versionListResponse
	if err := cli.ReadJSON(resp, &versions); err != nil {
		return fmt.Errorf("failed to parse versions response: %w", err)
	}

	// Find the latest draft version (versions are ordered by version_number DESC).
	var draft *versionListItem
	for i := range versions.Items {
		if versions.Items[i].Status == "draft" {
			draft = &versions.Items[i]
			break
		}
	}

	if draft == nil {
		return fmt.Errorf("recipe %q has no draft version to publish", slug)
	}

	// 3. Publish via the flat recipe-versions endpoint.
	publishPath := fmt.Sprintf("/api/v1/recipe-versions/%d/publish", draft.ID)
	resp, err = client.Post(publishPath, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := cli.Handle401(resp); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("publish failed: %s", errResp.Error)
		}
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var published publishVersionResponse
	if err := cli.ReadJSON(resp, &published); err != nil {
		return fmt.Errorf("failed to parse publish response: %w", err)
	}

	if publishJSONFlag {
		output := map[string]interface{}{
			"recipeSlug":    slug,
			"versionId":     published.ID,
			"versionNumber": published.VersionNumber,
			"status":        published.Status,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	fmt.Printf("Recipe %q version %d published\n", slug, published.VersionNumber)
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
