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
	createDescFlag     string

	publishJSONFlag bool

	listJSONFlag   bool
	listActiveFlag string

	showJSONFlag bool

	recipeTasksJSONFlag bool

	rollTitleFlag    string
	rollOrderQtyFlag int
	rollJSONFlag     bool
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
	Use:   "create <name>",
	Short: "Create a new recipe",
	Long:  "Create a new recipe. Use --from-toml to import from a TOML file, or just provide a name to create a blank task-tree recipe.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRecipeCreate,
}

var recipeTasksCmd = &cobra.Command{
	Use:   "tasks <slug>",
	Short: "Show recipe task tree",
	Long:  "Display the task tree for a recipe's current version.",
	Args:  cobra.ExactArgs(1),
	RunE:  runRecipeTasks,
}

var recipeRollCmd = &cobra.Command{
	Use:   "roll <slug>",
	Short: "Roll a recipe into a new job",
	Long:  "Clone a published recipe's task tree into a new job.",
	Args:  cobra.ExactArgs(1),
	RunE:  runRecipeRoll,
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

	recipeCreateCmd.Flags().StringVar(&createFromTOMLFlag, "from-toml", "", "Path to TOML file to import (uses old TOML engine)")
	recipeCreateCmd.Flags().StringVar(&createNameFlag, "name", "", "Recipe name (defaults to 'formula' field from TOML, or positional arg)")
	recipeCreateCmd.Flags().StringVar(&createDescFlag, "description", "", "Recipe description")
	recipeCreateCmd.Flags().BoolVar(&createJSONFlag, "json", false, "Output as JSON")

	recipePublishCmd.Flags().BoolVar(&publishJSONFlag, "json", false, "Output as JSON")

	recipeListCmd.Flags().BoolVar(&listJSONFlag, "json", false, "Output as JSON")
	recipeListCmd.Flags().StringVar(&listActiveFlag, "active", "", "Filter by active status (true/false)")

	recipeShowCmd.Flags().BoolVar(&showJSONFlag, "json", false, "Output as JSON")

	recipeTasksCmd.Flags().BoolVar(&recipeTasksJSONFlag, "json", false, "Output as JSON")

	recipeRollCmd.Flags().StringVar(&rollTitleFlag, "title", "", "Job title (defaults to recipe name)")
	recipeRollCmd.Flags().IntVar(&rollOrderQtyFlag, "qty", 0, "Order quantity")
	recipeRollCmd.Flags().BoolVar(&rollJSONFlag, "json", false, "Output as JSON")

	recipeCmd.AddCommand(recipeListCmd)
	recipeCmd.AddCommand(recipeShowCmd)
	recipeCmd.AddCommand(recipePourCmd)
	recipeCmd.AddCommand(recipeCreateCmd)
	recipeCmd.AddCommand(recipePublishCmd)
	recipeCmd.AddCommand(recipeTasksCmd)
	recipeCmd.AddCommand(recipeRollCmd)
	rootCmd.AddCommand(recipeCmd)
}

func runRecipeList(cmd *cobra.Command, args []string) error {
	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	path := client.SpacePath("/recipes")
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

	// 1. Resolve slug to get recipe details via GET /api/v1/spaces/:spaceId/recipes?slug=<slug>&limit=1
	path := client.SpacePath(fmt.Sprintf("/recipes?slug=%s&limit=1", slug))
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

	// 2. Fetch the published version content via GET /api/v1/spaces/:spaceId/recipes/:id/versions.
	versionsPath := client.SpacePath(fmt.Sprintf("/recipes/%s/versions", recipe.ID))
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

	// 3. POST /api/v1/spaces/:spaceId/recipes/:id/pour
	path := client.SpacePath(fmt.Sprintf("/recipes/%s/pour", recipeID))
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
	// If --from-toml is provided, use the TOML import path.
	if createFromTOMLFlag != "" {
		return runRecipeCreateFromTOML(cmd, args)
	}

	// Otherwise, create a blank task-tree recipe.
	name := createNameFlag
	if name == "" && len(args) > 0 {
		name = args[0]
	}
	if name == "" {
		return fmt.Errorf("recipe name is required — provide as argument or use --name")
	}

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	createBody := map[string]interface{}{
		"name": name,
	}
	if createDescFlag != "" {
		createBody["description"] = createDescFlag
	}

	resp, err := client.Post(client.SpacePath("/recipes"), createBody)
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

	var recipe struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		Slug           string `json:"slug"`
		CurrentVersion *struct {
			ID         int     `json:"id"`
			RootTaskID *string `json:"rootTaskId,omitempty"`
		} `json:"currentVersion,omitempty"`
	}
	if err := cli.ReadJSON(resp, &recipe); err != nil {
		return fmt.Errorf("failed to parse recipe response: %w", err)
	}

	if createJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(recipe)
	}

	fmt.Printf("Recipe %q created (slug: %s)\n", recipe.Name, recipe.Slug)
	if recipe.CurrentVersion != nil && recipe.CurrentVersion.RootTaskID != nil {
		fmt.Printf("  Root task: %s\n", *recipe.CurrentVersion.RootTaskID)
		fmt.Printf("  Add tasks: nori task add <title> --parent %s\n", *recipe.CurrentVersion.RootTaskID)
	}

	return nil
}

func runRecipeCreateFromTOML(cmd *cobra.Command, args []string) error {
	tomlPath := createFromTOMLFlag

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

	// Determine the recipe name: --name flag > positional arg > TOML formula field.
	name := createNameFlag
	if name == "" && len(args) > 0 {
		name = args[0]
	}
	if name == "" {
		var f tomlFormula
		if err := toml.Unmarshal(data, &f); err == nil && f.Formula != "" {
			name = f.Formula
		}
	}
	if name == "" {
		return fmt.Errorf("could not determine recipe name — set 'formula' in the TOML or use --name")
	}

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	// POST /api/v1/recipes — create the recipe.
	createBody := map[string]interface{}{
		"name": name,
	}

	resp, err := client.Post(client.SpacePath("/recipes"), createBody)
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

	// POST /api/v1/recipes/:id/versions — create version with TOML content.
	versionBody := map[string]interface{}{
		"content":       content,
		"changeSummary": "Initial import from TOML file",
	}

	versionPath := client.SpacePath(fmt.Sprintf("/recipes/%s/versions", recipe.ID))
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

	// POST /api/v1/recipes/:id/publish — auto-publish.
	publishPath := client.SpacePath(fmt.Sprintf("/recipes/%s/publish", recipe.ID))
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
	versionsPath := client.SpacePath(fmt.Sprintf("/recipes/%s/versions", recipeID))
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

	// 3. Publish via the recipe-scoped endpoint.
	publishPath := client.SpacePath(fmt.Sprintf("/recipes/%s/publish", recipeID))
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

func runRecipeTasks(cmd *cobra.Command, args []string) error {
	slug := args[0]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	// Resolve slug to recipe ID.
	recipeID, err := resolveRecipeSlug(client, slug)
	if err != nil {
		return err
	}

	// GET /api/v1/spaces/:spaceId/recipes/:id — includes task tree in currentVersion.
	path := client.SpacePath(fmt.Sprintf("/recipes/%s", recipeID))
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

	var recipe struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		CurrentVersion *struct {
			RootTaskID *string         `json:"rootTaskId,omitempty"`
			TaskTree   *cliTaskTree    `json:"taskTree,omitempty"`
		} `json:"currentVersion,omitempty"`
	}
	if err := cli.ReadJSON(resp, &recipe); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	rootTaskID := ""
	if recipe.CurrentVersion != nil && recipe.CurrentVersion.RootTaskID != nil {
		rootTaskID = *recipe.CurrentVersion.RootTaskID
	}

	if rootTaskID == "" {
		fmt.Println("No task tree found for this recipe.")
		return nil
	}

	// Fetch the task tree via GET /api/v1/tasks/:id/tree.
	treePath := client.SpacePath(fmt.Sprintf("/tasks/%s/tree", rootTaskID))
	resp, err = client.Get(treePath)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
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

	if recipeTasksJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(tree)
	}

	// Also fetch deps for all tasks in the tree to show dependency info.
	allTasks := flattenTree(&tree)
	depMap := fetchAllDeps(client, allTasks)

	fmt.Printf("Recipe: %s (root: %s)\n\n", recipe.Name, tree.ID)
	printTaskTree(&tree, "", true, depMap)

	return nil
}

// cliTaskTree mirrors the task tree response for CLI display.
type cliTaskTree struct {
	ID                string        `json:"id"`
	Title             string        `json:"title"`
	Type              string        `json:"type"`
	Status            string        `json:"status"`
	StationID         *string       `json:"stationId,omitempty"`
	Description       *string       `json:"description,omitempty"`
	EstimatedTimeSecs *int          `json:"estimatedTimeSeconds,omitempty"`
	Children          []cliTaskTree `json:"children"`
}

// printTaskTree recursively prints a task tree with indentation.
func printTaskTree(node *cliTaskTree, prefix string, isLast bool, depMap map[string][]string) {
	connector := "├── "
	childPrefix := "│   "
	if isLast {
		connector = "└── "
		childPrefix = "    "
	}

	// For the root node, don't print a connector.
	if prefix == "" {
		fmt.Printf("%s [%s]\n", node.Title, node.ID)
	} else {
		line := fmt.Sprintf("%s%s%s [%s]", prefix, connector, node.Title, node.ID)
		fmt.Print(line)

		// Show deps inline.
		if deps, ok := depMap[node.ID]; ok && len(deps) > 0 {
			fmt.Printf("  (after: %s)", strings.Join(deps, ", "))
		}
		fmt.Println()
	}

	for i, child := range node.Children {
		isChildLast := i == len(node.Children)-1
		printTaskTree(&child, prefix+childPrefix, isChildLast, depMap)
	}
}

// flattenTree collects all task IDs from a tree.
func flattenTree(node *cliTaskTree) []string {
	ids := []string{node.ID}
	for i := range node.Children {
		ids = append(ids, flattenTree(&node.Children[i])...)
	}
	return ids
}

// fetchAllDeps fetches dependencies for all tasks and returns a map of taskID -> blocker IDs.
func fetchAllDeps(client *cli.Client, taskIDs []string) map[string][]string {
	depMap := make(map[string][]string)
	for _, id := range taskIDs {
		path := client.SpacePath(fmt.Sprintf("/tasks/%s/deps", id))
		resp, err := client.Get(path)
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}

		var deps struct {
			Blockers []struct {
				ToTaskID string `json:"toTaskId"`
			} `json:"blockers"`
		}
		if err := cli.ReadJSON(resp, &deps); err != nil {
			continue
		}

		for _, b := range deps.Blockers {
			depMap[id] = append(depMap[id], b.ToTaskID)
		}
	}
	return depMap
}

func runRecipeRoll(cmd *cobra.Command, args []string) error {
	slug := args[0]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)

	recipeID, err := resolveRecipeSlug(client, slug)
	if err != nil {
		return err
	}

	rollBody := map[string]interface{}{}
	if rollTitleFlag != "" {
		rollBody["title"] = rollTitleFlag
	}
	if rollOrderQtyFlag > 0 {
		rollBody["orderQty"] = rollOrderQtyFlag
	}

	path := client.SpacePath(fmt.Sprintf("/recipes/%s/roll", recipeID))
	resp, err := client.Post(path, rollBody)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := cli.Handle401(resp); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("roll failed: %s", errResp.Error)
		}
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var job pourResponse
	if err := cli.ReadJSON(resp, &job); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if rollJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(job)
	}

	fmt.Printf("Job %s created: %s\n", job.ID, job.Title)
	return nil
}

// resolveRecipeSlug looks up a recipe by slug via the list endpoint and returns its ID.
func resolveRecipeSlug(client *cli.Client, slug string) (string, error) {
	path := client.SpacePath(fmt.Sprintf("/recipes?slug=%s&limit=1", slug))
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
	path := client.SpacePath(fmt.Sprintf("/tasks?parentId=%s&limit=1", parentID))
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
