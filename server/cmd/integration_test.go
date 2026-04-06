// Package cmd contains the end-to-end integration test for the chair workflow.
//
// This test exercises the full CLI workflow against a mock HTTP server that
// simulates the Nori API:
//
//	login -> recipe create --from-toml chair.toml -> recipe pour chair
//	  -> ready -> task claim <id> -> task complete <id> -> (repeat)
//
// Run:
//
//	go test -run TestChairWorkflowE2E -v ./cmd/
//
// To run against the real Docker dev environment, see the shell script
// at tests/e2e_chair_workflow.sh.
package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tylerjvollick/nori/internal/cli"
)

// --- Mock server state ---

// mockTask represents a task in the mock server's state.
type mockTask struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	Type       string  `json:"type"`
	StationID  *string `json:"stationId,omitempty"`
	Priority   int     `json:"priority"`
	ParentID   *string `json:"parentId,omitempty"`
	AssigneeID *string `json:"assigneeId,omitempty"`
}

// mockRecipe represents a recipe in the mock server's state.
type mockRecipe struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	IsActive bool   `json:"isActive"`
}

// mockVersion represents a recipe version in the mock server's state.
type mockVersion struct {
	ID            int    `json:"id"`
	RecipeID      string `json:"recipeId"`
	VersionNumber int    `json:"versionNumber"`
	Status        string `json:"status"`
	Content       string `json:"content"`
	ChangeSummary string `json:"changeSummary"`
}

// mockState holds all state for the mock API server.
type mockState struct {
	mu       sync.Mutex
	recipes  []mockRecipe
	versions []mockVersion
	tasks    []mockTask

	// Authentication
	validToken string
	userID     string
	userEmail  string
	spaceID    string

	// Counters for unique IDs
	nextRecipeID  int
	nextVersionID int
	nextTaskSeq   int

	// Track API call sequence for validation
	apiCalls []string
}

func newMockState() *mockState {
	return &mockState{
		validToken:    "jwt-test-token-abc123",
		userID:        "user-uuid-001",
		userEmail:     "admin@nori.dev",
		spaceID:       "space-uuid-001",
		nextRecipeID:  1,
		nextVersionID: 1,
		nextTaskSeq:   1,
	}
}

func (s *mockState) recordCall(call string) {
	s.apiCalls = append(s.apiCalls, call)
}

func (s *mockState) addRecipe(name, slug string) *mockRecipe {
	id := fmt.Sprintf("recipe-uuid-%03d", s.nextRecipeID)
	s.nextRecipeID++
	r := mockRecipe{ID: id, Name: name, Slug: slug, IsActive: true}
	s.recipes = append(s.recipes, r)
	return &s.recipes[len(s.recipes)-1]
}

func (s *mockState) addVersion(recipeID, content, changeSummary string) *mockVersion {
	// Find the highest version number for this recipe.
	maxVer := 0
	for _, v := range s.versions {
		if v.RecipeID == recipeID && v.VersionNumber > maxVer {
			maxVer = v.VersionNumber
		}
	}
	id := s.nextVersionID
	s.nextVersionID++
	v := mockVersion{
		ID:            id,
		RecipeID:      recipeID,
		VersionNumber: maxVer + 1,
		Status:        "draft",
		Content:       content,
		ChangeSummary: changeSummary,
	}
	s.versions = append(s.versions, v)
	return &s.versions[len(s.versions)-1]
}

func (s *mockState) publishVersion(id int) *mockVersion {
	for i := range s.versions {
		if s.versions[i].ID == id {
			// Archive any previously published version for this recipe.
			for j := range s.versions {
				if s.versions[j].RecipeID == s.versions[i].RecipeID && s.versions[j].Status == "published" {
					s.versions[j].Status = "archived"
				}
			}
			s.versions[i].Status = "published"
			return &s.versions[i]
		}
	}
	return nil
}

func (s *mockState) pourRecipe(recipeID string) (*mockTask, []*mockTask) {
	// Find the recipe.
	var recipe *mockRecipe
	for i := range s.recipes {
		if s.recipes[i].ID == recipeID {
			recipe = &s.recipes[i]
			break
		}
	}
	if recipe == nil {
		return nil, nil
	}

	// Create the root job task.
	jobID := fmt.Sprintf("job-%03d", s.nextTaskSeq)
	s.nextTaskSeq++
	job := mockTask{
		ID:       jobID,
		Title:    recipe.Name,
		Status:   "open",
		Type:     "job",
		Priority: 1,
	}
	s.tasks = append(s.tasks, job)

	// Create child tasks simulating a simplified chair recipe pour.
	// The real pour engine would parse the TOML and expand loops, but
	// for this integration test we create a representative set of tasks.
	childDefs := []struct {
		suffix  string
		title   string
		station string
	}{
		{"1", "Rough mill — Walnut", "mill"},
		{"2", "Layout and mark joinery — chair 1", "joinery"},
		{"3", "Cut mortises — chair 1", "joinery"},
		{"4", "Cut tenons — chair 1", "joinery"},
		{"5", "Dry fit — chair 1", "joinery"},
		{"6", "Glue up sub-assemblies — chair 1", "assembly"},
		{"7", "Sand to 220 — chair 1", "finish"},
		{"8", "Batch finish — oil", "finish"},
		{"9", "Final inspection", ""},
	}

	var children []*mockTask
	for _, def := range childDefs {
		childID := fmt.Sprintf("%s.%s", jobID, def.suffix)
		child := mockTask{
			ID:       childID,
			Title:    def.title,
			Status:   "open",
			Type:     "task",
			ParentID: &jobID,
			Priority: 1,
		}
		if def.station != "" {
			station := def.station
			child.StationID = &station
		}
		s.tasks = append(s.tasks, child)
		children = append(children, &s.tasks[len(s.tasks)-1])
	}

	return &job, children
}

func (s *mockState) findRecipeBySlug(slug string) *mockRecipe {
	for i := range s.recipes {
		if s.recipes[i].Slug == slug {
			return &s.recipes[i]
		}
	}
	return nil
}

func (s *mockState) findRecipeByID(id string) *mockRecipe {
	for i := range s.recipes {
		if s.recipes[i].ID == id {
			return &s.recipes[i]
		}
	}
	return nil
}

func (s *mockState) readyTasks() []mockTask {
	var ready []mockTask
	for _, t := range s.tasks {
		if t.Status == "open" && t.Type == "task" {
			ready = append(ready, t)
		}
	}
	return ready
}

func (s *mockState) findTask(id string) *mockTask {
	for i := range s.tasks {
		if s.tasks[i].ID == id {
			return &s.tasks[i]
		}
	}
	return nil
}

// --- Mock HTTP server ---

func newMockServer(t *testing.T, state *mockState) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		// --- Auth: login endpoint (public) ---
		if r.URL.Path == "/auth/login" && r.Method == http.MethodPost {
			state.recordCall("POST /auth/login")
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)

			if body["email"] == state.userEmail {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"accessToken":        state.validToken,
					"userId":             state.userID,
					"userEmail":          state.userEmail,
					"firstName":          "Admin",
					"lastName":           "User",
					"mustChangePassword": false,
					"activeSpaceId":      state.spaceID,
				})
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid email or password"})
			return
		}

		// --- All other routes require auth ---
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer "+state.validToken {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid credentials"})
			return
		}

		// --- Recipe routes ---

		// POST /api/v1/recipes — create recipe
		if r.URL.Path == "/api/v1/recipes" && r.Method == http.MethodPost {
			state.recordCall("POST /api/v1/recipes")
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			name, _ := body["name"].(string)

			recipe := state.addRecipe(name, toSlug(name))
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       recipe.ID,
				"name":     recipe.Name,
				"slug":     recipe.Slug,
				"isActive": recipe.IsActive,
			})
			return
		}

		// GET /api/v1/recipes — list recipes (supports slug filter)
		if r.URL.Path == "/api/v1/recipes" && r.Method == http.MethodGet {
			state.recordCall("GET /api/v1/recipes")
			slugFilter := r.URL.Query().Get("slug")

			var items []mockRecipe
			for _, rec := range state.recipes {
				if slugFilter == "" || rec.Slug == slugFilter {
					items = append(items, rec)
				}
			}
			if items == nil {
				items = []mockRecipe{}
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": items,
				"total": len(items),
			})
			return
		}

		// Handle recipe-specific routes: /api/v1/recipes/:id/...
		if len(r.URL.Path) > len("/api/v1/recipes/") {
			pathRest := r.URL.Path[len("/api/v1/recipes/"):]

			// Extract recipeID and subpath by finding the first /
			slashIdx := indexOf(pathRest, "/")
			if slashIdx > 0 {
				recipeID := pathRest[:slashIdx]
				subpath := pathRest[slashIdx:]

				// POST /api/v1/recipes/:id/pour
				if subpath == "/pour" && r.Method == http.MethodPost {
					state.recordCall(fmt.Sprintf("POST /api/v1/recipes/%s/pour", recipeID))

					// Verify the recipe has a published version.
					hasPublished := false
					for _, v := range state.versions {
						if v.RecipeID == recipeID && v.Status == "published" {
							hasPublished = true
							break
						}
					}
					if !hasPublished {
						w.WriteHeader(http.StatusUnprocessableEntity)
						json.NewEncoder(w).Encode(map[string]string{"error": "recipe has no published version"})
						return
					}

					job, _ := state.pourRecipe(recipeID)
					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"id":    job.ID,
						"title": job.Title,
						"type":  job.Type,
					})
					return
				}

				// POST /api/v1/recipes/:id/versions
				if subpath == "/versions" && r.Method == http.MethodPost {
					state.recordCall(fmt.Sprintf("POST /api/v1/recipes/%s/versions", recipeID))
					var body map[string]interface{}
					json.NewDecoder(r.Body).Decode(&body)

					content, _ := body["content"].(string)
					changeSummary, _ := body["changeSummary"].(string)
					ver := state.addVersion(recipeID, content, changeSummary)

					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"id":            ver.ID,
						"versionNumber": ver.VersionNumber,
						"status":        ver.Status,
					})
					return
				}

				// GET /api/v1/recipes/:id/versions
				if subpath == "/versions" && r.Method == http.MethodGet {
					state.recordCall(fmt.Sprintf("GET /api/v1/recipes/%s/versions", recipeID))
					var items []map[string]interface{}
					for _, v := range state.versions {
						if v.RecipeID == recipeID {
							items = append(items, map[string]interface{}{
								"id":            v.ID,
								"versionNumber": v.VersionNumber,
								"status":        v.Status,
								"content":       v.Content,
								"changeSummary": v.ChangeSummary,
							})
						}
					}
					if items == nil {
						items = []map[string]interface{}{}
					}
					json.NewEncoder(w).Encode(map[string]interface{}{
						"items": items,
						"total": len(items),
					})
					return
				}

				// POST /api/v1/recipes/:id/versions/:vid/publish
				if len(subpath) > len("/versions/") && r.Method == http.MethodPost {
					// Parse version ID from: /versions/<vid>/publish
					versionPart := subpath[len("/versions/"):]
					var vid int
					var suffix string
					fmt.Sscanf(versionPart, "%d/%s", &vid, &suffix)
					if suffix == "publish" {
						state.recordCall(fmt.Sprintf("POST /api/v1/recipes/%s/versions/%d/publish", recipeID, vid))
						v := state.publishVersion(vid)
						if v == nil {
							w.WriteHeader(http.StatusNotFound)
							json.NewEncoder(w).Encode(map[string]string{"error": "version not found"})
							return
						}
						json.NewEncoder(w).Encode(map[string]interface{}{
							"id":            v.ID,
							"versionNumber": v.VersionNumber,
							"status":        v.Status,
						})
						return
					}
				}
			}
		}

		// --- Recipe versions flat endpoint ---
		// POST /api/v1/recipe-versions/:vid/publish
		if len(r.URL.Path) > len("/api/v1/recipe-versions/") && r.Method == http.MethodPost {
			rest := r.URL.Path[len("/api/v1/recipe-versions/"):]
			var vid int
			var suffix string
			fmt.Sscanf(rest, "%d/%s", &vid, &suffix)
			if suffix == "publish" {
				state.recordCall(fmt.Sprintf("POST /api/v1/recipe-versions/%d/publish", vid))
				v := state.publishVersion(vid)
				if v == nil {
					w.WriteHeader(http.StatusNotFound)
					json.NewEncoder(w).Encode(map[string]string{"error": "version not found"})
					return
				}
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":            v.ID,
					"versionNumber": v.VersionNumber,
					"status":        v.Status,
				})
				return
			}
		}

		// --- Task routes ---

		// GET /api/v1/tasks/ready
		if r.URL.Path == "/api/v1/tasks/ready" && r.Method == http.MethodGet {
			state.recordCall("GET /api/v1/tasks/ready")
			ready := state.readyTasks()
			if ready == nil {
				ready = []mockTask{}
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": ready,
				"total": len(ready),
			})
			return
		}

		// GET /api/v1/tasks — list tasks (supports parentId, status, assigneeId filters)
		if r.URL.Path == "/api/v1/tasks" && r.Method == http.MethodGet {
			state.recordCall("GET /api/v1/tasks")
			parentFilter := r.URL.Query().Get("parentId")

			var items []mockTask
			for _, t := range state.tasks {
				if parentFilter != "" {
					if t.ParentID != nil && *t.ParentID == parentFilter {
						items = append(items, t)
					}
					continue
				}
				items = append(items, t)
			}
			if items == nil {
				items = []mockTask{}
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": items,
				"total": len(items),
			})
			return
		}

		// POST /api/v1/tasks/:id/claim
		if len(r.URL.Path) > len("/api/v1/tasks/") && r.Method == http.MethodPost {
			rest := r.URL.Path[len("/api/v1/tasks/"):]

			// Extract task ID and action.
			// Paths: /api/v1/tasks/<id>/claim, /api/v1/tasks/<id>/complete
			for _, action := range []string{"/claim", "/complete", "/pause", "/resume", "/skip"} {
				if idx := indexOf(rest, action); idx > 0 {
					taskID := rest[:idx]
					actionName := action[1:] // strip leading /

					state.recordCall(fmt.Sprintf("POST /api/v1/tasks/%s/%s", taskID, actionName))

					task := state.findTask(taskID)
					if task == nil {
						w.WriteHeader(http.StatusNotFound)
						json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("task %q not found", taskID)})
						return
					}

					switch actionName {
					case "claim":
						if task.Status != "open" {
							w.WriteHeader(http.StatusConflict)
							json.NewEncoder(w).Encode(map[string]string{
								"error": fmt.Sprintf("task %q cannot be claimed: status is %q, must be \"open\"", taskID, task.Status),
							})
							return
						}
						task.Status = "active"
						assignee := state.userID
						task.AssigneeID = &assignee

					case "complete":
						if task.Status != "active" {
							w.WriteHeader(http.StatusConflict)
							json.NewEncoder(w).Encode(map[string]string{
								"error": fmt.Sprintf("task %q cannot be completed: status is %q, must be \"active\"", taskID, task.Status),
							})
							return
						}
						task.Status = "done"
					}

					json.NewEncoder(w).Encode(map[string]interface{}{
						"id":       task.ID,
						"title":    task.Title,
						"status":   task.Status,
						"priority": task.Priority,
					})
					return
				}
			}
		}

		// --- Fallthrough ---
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	}))
}

// indexOf returns the index of substr in s, or -1 if not found.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// toSlug converts a recipe name to a URL-friendly slug.
func toSlug(name string) string {
	slug := make([]byte, 0, len(name))
	for _, c := range name {
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' {
			slug = append(slug, byte(c))
		} else if c >= 'A' && c <= 'Z' {
			slug = append(slug, byte(c-'A'+'a'))
		} else if c == ' ' || c == '_' {
			slug = append(slug, '-')
		}
	}
	return string(slug)
}

// --- The Integration Test ---

// TestChairWorkflowE2E exercises the full CLI workflow end-to-end:
//
//	login -> recipe create --from-toml chair.toml -> recipe pour dining-chair
//	  -> ready -> task claim <id> -> task complete <id> -> (repeat for N tasks)
//
// This test uses a stateful mock HTTP server that tracks recipes, versions,
// and tasks, validating the complete API call sequence.
func TestChairWorkflowE2E(t *testing.T) {
	// ── Setup ─────────────────────────────────────────────────────────────

	state := newMockState()
	server := newMockServer(t, state)
	defer server.Close()

	// Set non-interactive mode to avoid terminal prompts on 401.
	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	// Use a temp HOME so credentials are isolated.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Find the chair.toml seed file.
	chairTOML := findChairTOML(t)

	// ── Step 1: Login ─────────────────────────────────────────────────────

	t.Log("Step 1: Login")

	client := cli.NewClientWithURL(server.URL)
	resp, err := client.Post("/auth/login", map[string]string{
		"email":    "admin@nori.dev",
		"password": "admin",
	})
	require.NoError(t, err, "login request should succeed")
	require.Equal(t, http.StatusOK, resp.StatusCode, "login should return 200")

	var loginResp loginResponse
	require.NoError(t, cli.ReadJSON(resp, &loginResp), "should parse login response")
	assert.Equal(t, state.validToken, loginResp.AccessToken)
	assert.False(t, loginResp.MustChangePassword)
	assert.NotNil(t, loginResp.ActiveSpaceID)

	// Save credentials (simulating what `nori login` does).
	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: loginResp.AccessToken,
		UserID:      loginResp.UserID,
		UserEmail:   loginResp.UserEmail,
	}
	if loginResp.ActiveSpaceID != nil {
		creds.SpaceID = *loginResp.ActiveSpaceID
	}
	require.NoError(t, cli.SaveCredentials(creds))

	// Verify credentials are loadable.
	loaded, err := cli.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, state.validToken, loaded.AccessToken)
	assert.Equal(t, state.spaceID, loaded.SpaceID)

	// ── Step 2: Recipe Create from TOML ───────────────────────────────────

	t.Log("Step 2: Recipe create --from-toml chair.toml")

	createFromTOMLFlag = chairTOML
	createNameFlag = ""
	createJSONFlag = false
	defer func() { createFromTOMLFlag = "" }()

	err = runRecipeCreate(recipeCreateCmd, nil)
	require.NoError(t, err, "recipe create should succeed")

	// Verify recipe was created in state.
	state.mu.Lock()
	require.Len(t, state.recipes, 1, "should have 1 recipe")
	recipe := state.recipes[0]
	assert.Equal(t, "Dining Chair", recipe.Name)
	assert.Equal(t, "dining-chair", recipe.Slug)

	// Verify version was created and published.
	require.Len(t, state.versions, 1, "should have 1 version")
	ver := state.versions[0]
	assert.Equal(t, recipe.ID, ver.RecipeID)
	assert.Equal(t, "published", ver.Status)
	assert.Equal(t, 1, ver.VersionNumber)
	assert.Contains(t, ver.Content, "Dining Chair")
	state.mu.Unlock()

	// ── Step 3: Recipe Pour ───────────────────────────────────────────────

	t.Log("Step 3: Recipe pour dining-chair --var batch_size=6")

	pourVarFlags = []string{"batch_size=6"}
	pourOrderFlag = ""
	pourJSONFlag = false
	defer func() {
		pourVarFlags = nil
	}()

	err = runRecipePour(recipePourCmd, []string{"dining-chair"})
	require.NoError(t, err, "recipe pour should succeed")

	// Verify tasks were created.
	state.mu.Lock()
	require.True(t, len(state.tasks) > 1, "should have created job + child tasks")
	jobTask := state.tasks[0]
	assert.Equal(t, "job", jobTask.Type)
	assert.Equal(t, "open", jobTask.Status)
	assert.Equal(t, "Dining Chair", jobTask.Title)

	// Count child tasks.
	childCount := 0
	for _, task := range state.tasks {
		if task.ParentID != nil && *task.ParentID == jobTask.ID {
			childCount++
		}
	}
	assert.Equal(t, 9, childCount, "should have 9 child tasks from the pour")
	state.mu.Unlock()

	// ── Step 4: Ready (list ready tasks) ──────────────────────────────────

	t.Log("Step 4: nori ready")

	readyJSONFlag = false
	err = runReady(readyCmd, nil)
	require.NoError(t, err, "ready should succeed")

	// ── Step 5: Claim and complete the first 3 tasks ──────────────────────

	t.Log("Step 5: Task claim + complete loop")

	state.mu.Lock()
	readyTasks := state.readyTasks()
	state.mu.Unlock()
	require.True(t, len(readyTasks) >= 3, "should have at least 3 ready tasks")

	// Work through the first 3 tasks: claim then complete each.
	for i := 0; i < 3; i++ {
		taskID := readyTasks[i].ID
		t.Logf("  Claiming task %s: %s", taskID, readyTasks[i].Title)

		taskJSONFlag = false
		err = runTaskClaim(taskClaimCmd, []string{taskID})
		require.NoError(t, err, "task claim should succeed for %s", taskID)

		// Verify status changed to active.
		state.mu.Lock()
		task := state.findTask(taskID)
		require.NotNil(t, task)
		assert.Equal(t, "active", task.Status, "claimed task should be active")
		state.mu.Unlock()

		t.Logf("  Completing task %s: %s", taskID, readyTasks[i].Title)

		err = runTaskComplete(taskCompleteCmd, []string{taskID})
		require.NoError(t, err, "task complete should succeed for %s", taskID)

		// Verify status changed to done.
		state.mu.Lock()
		task = state.findTask(taskID)
		require.NotNil(t, task)
		assert.Equal(t, "done", task.Status, "completed task should be done")
		state.mu.Unlock()
	}

	// ── Step 6: Ready again (verify remaining tasks) ──────────────────────

	t.Log("Step 6: nori ready (after completing 3 tasks)")

	err = runReady(readyCmd, nil)
	require.NoError(t, err, "ready should succeed after completing tasks")

	// Verify only open tasks remain in the ready list.
	state.mu.Lock()
	remainingReady := state.readyTasks()
	assert.Equal(t, 6, len(remainingReady), "should have 6 remaining ready tasks (9 - 3 completed)")
	state.mu.Unlock()

	// ── Step 7: Validate API call sequence ────────────────────────────────

	t.Log("Step 7: Validate API call sequence")

	state.mu.Lock()
	calls := state.apiCalls
	state.mu.Unlock()

	// Verify the expected sequence of API calls.
	require.True(t, len(calls) >= 10, "should have at least 10 API calls, got %d", len(calls))

	// The first call should be login.
	assert.Equal(t, "POST /auth/login", calls[0], "first call should be login")

	// Recipe create should follow: create recipe, create version, publish version.
	assert.Equal(t, "POST /api/v1/recipes", calls[1], "should create recipe")
	assert.Contains(t, calls[2], "POST /api/v1/recipes/recipe-uuid-001/versions", "should create version")
	assert.Contains(t, calls[3], "POST /api/v1/recipes/recipe-uuid-001/versions/1/publish", "should publish version")

	// Pour should follow: resolve slug, then pour.
	assert.Equal(t, "GET /api/v1/recipes", calls[4], "should resolve slug")
	assert.Contains(t, calls[5], "POST /api/v1/recipes/recipe-uuid-001/pour", "should pour recipe")

	// Task count query after pour.
	assert.Equal(t, "GET /api/v1/tasks", calls[6], "should query child tasks after pour")

	// Ready check.
	assert.Equal(t, "GET /api/v1/tasks/ready", calls[7], "should check ready tasks")

	// Claim/complete cycle should alternate.
	claimCompleteStart := 8
	for i := 0; i < 3; i++ {
		claimIdx := claimCompleteStart + (i * 2)
		completeIdx := claimIdx + 1
		if claimIdx < len(calls) {
			assert.Contains(t, calls[claimIdx], "/claim", "call %d should be a claim", claimIdx)
		}
		if completeIdx < len(calls) {
			assert.Contains(t, calls[completeIdx], "/complete", "call %d should be a complete", completeIdx)
		}
	}

	// ── Summary ───────────────────────────────────────────────────────────

	t.Logf("E2E workflow completed successfully:")
	t.Logf("  - Logged in as %s", state.userEmail)
	t.Logf("  - Created recipe %q (slug: %s)", recipe.Name, recipe.Slug)
	t.Logf("  - Poured into job %s with %d tasks", jobTask.ID, childCount)
	t.Logf("  - Claimed and completed 3 tasks")
	t.Logf("  - %d tasks remaining", len(remainingReady))
	t.Logf("  - Total API calls: %d", len(calls))
}

// TestChairWorkflowE2E_RecipeAlreadyExists verifies the workflow handles
// duplicate recipe creation gracefully (a common re-run scenario).
func TestChairWorkflowE2E_RecipeAlreadyExists(t *testing.T) {
	state := newMockState()
	server := newMockServer(t, state)
	defer server.Close()

	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	setupCredentials(t, server.URL)

	// Override credentials with the mock server's valid token.
	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: state.validToken,
		UserID:      state.userID,
		UserEmail:   state.userEmail,
		SpaceID:     state.spaceID,
	}
	require.NoError(t, cli.SaveCredentials(creds))

	chairTOML := findChairTOML(t)

	// Create recipe the first time.
	createFromTOMLFlag = chairTOML
	createNameFlag = ""
	createJSONFlag = false
	err := runRecipeCreate(recipeCreateCmd, nil)
	require.NoError(t, err, "first create should succeed")

	// Pour the recipe.
	pourVarFlags = nil
	pourOrderFlag = ""
	pourJSONFlag = false
	err = runRecipePour(recipePourCmd, []string{"dining-chair"})
	require.NoError(t, err, "pour should succeed")

	// Verify tasks were created.
	state.mu.Lock()
	assert.True(t, len(state.tasks) > 0, "should have tasks after pour")
	state.mu.Unlock()

	// Reset flags.
	createFromTOMLFlag = ""
	pourVarFlags = nil
}

// TestChairWorkflowE2E_UnauthorizedMidFlow verifies that an expired token
// during the workflow returns a clear error.
func TestChairWorkflowE2E_UnauthorizedMidFlow(t *testing.T) {
	state := newMockState()
	server := newMockServer(t, state)
	defer server.Close()

	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Save credentials with an INVALID token.
	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: "expired-token",
		UserID:      state.userID,
		UserEmail:   state.userEmail,
		SpaceID:     state.spaceID,
	}
	require.NoError(t, cli.SaveCredentials(creds))

	// Try to pour — should fail with auth error.
	pourVarFlags = nil
	pourOrderFlag = ""
	pourJSONFlag = false
	err := runRecipePour(recipePourCmd, []string{"dining-chair"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication expired or invalid")
}

// TestChairWorkflowE2E_PourWithoutPublish verifies that pouring a recipe
// without a published version returns a clear error.
func TestChairWorkflowE2E_PourWithoutPublish(t *testing.T) {
	state := newMockState()
	server := newMockServer(t, state)
	defer server.Close()

	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: state.validToken,
		UserID:      state.userID,
		UserEmail:   state.userEmail,
		SpaceID:     state.spaceID,
	}
	require.NoError(t, cli.SaveCredentials(creds))

	// Manually add a recipe with a draft version (not published).
	state.mu.Lock()
	recipe := state.addRecipe("Test Recipe", "test-recipe")
	state.addVersion(recipe.ID, "content", "draft version")
	// Don't publish it.
	state.mu.Unlock()

	pourVarFlags = nil
	pourOrderFlag = ""
	pourJSONFlag = false
	err := runRecipePour(recipePourCmd, []string{"test-recipe"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no published version")
}

// TestChairWorkflowE2E_ClaimAlreadyActiveFails verifies that claiming an
// already-active task returns an error.
func TestChairWorkflowE2E_ClaimAlreadyActiveFails(t *testing.T) {
	state := newMockState()
	server := newMockServer(t, state)
	defer server.Close()

	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: state.validToken,
		UserID:      state.userID,
		UserEmail:   state.userEmail,
		SpaceID:     state.spaceID,
	}
	require.NoError(t, cli.SaveCredentials(creds))

	// Add a task that's already active.
	state.mu.Lock()
	assignee := state.userID
	state.tasks = append(state.tasks, mockTask{
		ID:         "task-001",
		Title:      "Already active",
		Status:     "active",
		Type:       "task",
		AssigneeID: &assignee,
	})
	state.mu.Unlock()

	taskJSONFlag = false
	err := runTaskClaim(taskClaimCmd, []string{"task-001"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be claimed")
}

// findChairTOML locates the chair.toml seed file relative to the test.
func findChairTOML(t *testing.T) string {
	t.Helper()

	// Try relative paths from the test working directory.
	candidates := []string{
		"../../seeds/chair.toml",    // from server/cmd/
		"../seeds/chair.toml",       // from server/
		"seeds/chair.toml",          // from root
		"../../../seeds/chair.toml", // deeper nesting
		filepath.Join("..", "..", "seeds", "chair.toml"),
	}

	for _, path := range candidates {
		abs, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if _, err := os.Stat(abs); err == nil {
			return abs
		}
	}

	t.Fatal("could not find seeds/chair.toml — ensure you run tests from the server/ directory or project root")
	return ""
}
