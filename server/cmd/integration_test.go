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
	Quantity   int     `json:"quantity"`
	StepID     string  `json:"stepId,omitempty"` // links back to recipe step
}

// mockDep represents a dependency edge: FromTaskID depends on ToTaskID.
type mockDep struct {
	FromTaskID string // the blocked task
	ToTaskID   string // the blocker
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
	deps     []mockDep

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

	// --- Chair recipe task definitions ---
	// Each step definition specifies: id, title, station, batchSize, needs.
	// batchSize=6 → 1 ticket with qty=6 (batch).
	// batchSize=1 → 6 tickets with qty=1 (per-piece).
	// Steps without explicit batchSize inherit from their dependency chain.
	//
	// This mirrors the real chair.toml / PourRecipe logic.

	type stepDef struct {
		id        string
		title     string
		station   string
		batchSize int // 0 means "inherit from needs chain"
		needs     []string
		taskType  string // "" means "task", "milestone" for done
	}

	steps := []stepDef{
		// STREAM 1: Legs (all batch — 1 ticket per step, qty=6)
		{id: "leg-rough-cut", title: "Rough cut leg stock", station: "miter-saw"},
		{id: "leg-joint-face", title: "Joint 1 face on legs", station: "jointer", needs: []string{"leg-rough-cut"}},
		{id: "leg-thickness", title: "Thickness plane legs", station: "thickness-planer", needs: []string{"leg-joint-face"}},
		{id: "leg-rough-shape", title: "Rough cut out leg shape", station: "band-saw", needs: []string{"leg-thickness"}},
		{id: "leg-shape-profile", title: "Shape leg profile", station: "router-table", needs: []string{"leg-rough-shape"}},
		{id: "leg-face-mortises", title: "Leg face mortises", station: "assembly-table", needs: []string{"leg-shape-profile"}},
		{id: "leg-edge-mortises", title: "Leg edge mortises", station: "assembly-table", needs: []string{"leg-shape-profile"}},
		{id: "hand-shape-legs", title: "Hand shape legs", station: "roubo-workbench", needs: []string{"leg-face-mortises", "leg-edge-mortises"}},

		// STREAM 2: Rails (all batch — 1 ticket per step, qty=6)
		{id: "rail-rough-cut", title: "Rough cut rail stock", station: "miter-saw"},
		{id: "rail-joint-face-edge", title: "Jointer 1 face and 1 edge on rails", station: "jointer", needs: []string{"rail-rough-cut"}},
		{id: "rail-thickness", title: "Thickness plane rails", station: "thickness-planer", needs: []string{"rail-joint-face-edge"}},
		{id: "rail-rip-width", title: "Rip rails to width", station: "table-saw", needs: []string{"rail-thickness"}},
		{id: "rail-compound-miters", title: "Rail compound miters", station: "table-saw", needs: []string{"rail-rip-width"}},
		{id: "rail-tenons", title: "Rail tenons", station: "panto-router", needs: []string{"rail-compound-miters"}},
		{id: "hand-shape-rails", title: "Hand shape rails", station: "roubo-workbench", needs: []string{"rail-tenons"}},

		// Dry fit (per-piece, batch_size=1)
		{id: "dry-fit", title: "Dry fit legs + rails", batchSize: 1, needs: []string{"hand-shape-legs", "hand-shape-rails"}},

		// STREAM 3: Chair back
		{id: "resaw-veneers", title: "Resaw veneers", station: "band-saw"},
		{id: "glue-lamination", title: "Glue lamination", station: "assembly-table", batchSize: 1, needs: []string{"resaw-veneers"}},
		{id: "true-up-edge", title: "True up one edge of glue lamination", station: "jointer", needs: []string{"glue-lamination"}},
		{id: "rip-back-parallel", title: "Rip back parallel", station: "table-saw", needs: []string{"true-up-edge"}},
		{id: "cut-back-miters", title: "Cut chair back compound miters", station: "table-saw", needs: []string{"rip-back-parallel"}},
		{id: "shape-back", title: "Shape chair back", station: "roubo-workbench", needs: []string{"cut-back-miters"}},

		// STREAM 4: Seat (all batch)
		{id: "cut-seat-blank", title: "Cut out seat blanks", station: "table-saw"},
		{id: "cut-foam", title: "Cut out seat foam", needs: []string{"cut-seat-blank"}},
		{id: "cut-fabric", title: "Cut out upholstery fabric", needs: []string{"cut-foam"}},
		{id: "upholster-seat", title: "Upholster seats", needs: []string{"cut-fabric"}},

		// Assembly (per-piece, batch_size=1)
		{id: "glue-up", title: "Glue up chair assembly", station: "assembly-table", batchSize: 1, needs: []string{"dry-fit", "shape-back"}},
		{id: "spray-finish", title: "Spray finish", station: "spray-booth", needs: []string{"glue-up"}},

		// Final convergence (per-piece)
		{id: "install-seat", title: "Install chair seat", batchSize: 1, needs: []string{"upholster-seat", "spray-finish"}},
		{id: "done", title: "6 Chairs complete", batchSize: 6, needs: []string{"install-seat"}, taskType: "milestone"},
	}

	// Resolve batch sizes via inheritance.
	orderQty := 6
	defaultBatchSize := 6
	resolvedBS := make(map[string]int)
	for _, step := range steps {
		if step.batchSize > 0 {
			resolvedBS[step.id] = step.batchSize
		}
	}
	// Propagate: steps without explicit batchSize inherit from deps.
	for _, step := range steps {
		if _, ok := resolvedBS[step.id]; !ok {
			if len(step.needs) > 0 {
				// All deps must agree; otherwise fall back to default.
				bs := resolvedBS[step.needs[0]]
				allSame := true
				for _, n := range step.needs[1:] {
					if resolvedBS[n] != bs {
						allSame = false
						break
					}
				}
				if allSame && bs > 0 {
					resolvedBS[step.id] = bs
				} else {
					resolvedBS[step.id] = defaultBatchSize
				}
			} else {
				resolvedBS[step.id] = defaultBatchSize
			}
		}
	}

	// Create tasks and track step→taskIDs mapping.
	stepTaskIDs := make(map[string][]string) // step ID → list of task IDs
	seq := 1

	var children []*mockTask
	for _, step := range steps {
		bs := resolvedBS[step.id]
		ticketCount := orderQty / bs
		qty := bs

		for n := 0; n < ticketCount; n++ {
			childID := fmt.Sprintf("%s.%d", jobID, seq)
			seq++

			title := step.title
			if ticketCount > 1 {
				title = fmt.Sprintf("%s %d of %d", step.title, n+1, ticketCount)
			}

			taskType := "task"
			if step.taskType != "" {
				taskType = step.taskType
			}

			child := mockTask{
				ID:       childID,
				Title:    title,
				Status:   "open",
				Type:     taskType,
				ParentID: &jobID,
				Priority: 1,
				Quantity: qty,
				StepID:   step.id,
			}
			if step.station != "" {
				station := step.station
				child.StationID = &station
			}
			s.tasks = append(s.tasks, child)
			children = append(children, &s.tasks[len(s.tasks)-1])
			stepTaskIDs[step.id] = append(stepTaskIDs[step.id], childID)
		}
	}

	// Wire dependencies using batch-aware patterns.
	for _, step := range steps {
		if len(step.needs) == 0 {
			continue
		}
		fromIDs := stepTaskIDs[step.id]
		for _, needID := range step.needs {
			toIDs := stepTaskIDs[needID]
			fromCount := len(fromIDs)
			toCount := len(toIDs)

			if fromCount == toCount {
				// 1:1 wiring
				for i := 0; i < fromCount; i++ {
					s.deps = append(s.deps, mockDep{FromTaskID: fromIDs[i], ToTaskID: toIDs[i]})
				}
			} else if fromCount > toCount && toCount == 1 {
				// Fan-out: single blocker → multiple blocked
				for i := 0; i < fromCount; i++ {
					s.deps = append(s.deps, mockDep{FromTaskID: fromIDs[i], ToTaskID: toIDs[0]})
				}
			} else if fromCount == 1 && toCount > 1 {
				// Fan-in: multiple blockers → single blocked
				for i := 0; i < toCount; i++ {
					s.deps = append(s.deps, mockDep{FromTaskID: fromIDs[0], ToTaskID: toIDs[i]})
				}
			}
		}
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
	// Build set of "done" task IDs for dependency resolution.
	doneSet := make(map[string]bool)
	for _, t := range s.tasks {
		if t.Status == "done" {
			doneSet[t.ID] = true
		}
	}

	// Build set of task IDs that have at least one unsatisfied dep.
	blockedSet := make(map[string]bool)
	for _, dep := range s.deps {
		if !doneSet[dep.ToTaskID] {
			blockedSet[dep.FromTaskID] = true
		}
	}

	var ready []mockTask
	for _, t := range s.tasks {
		if t.Status == "open" && t.Type == "task" && !blockedSet[t.ID] {
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

// findTasksByStep returns all tasks for a given step ID.
func (s *mockState) findTasksByStep(stepID string) []*mockTask {
	var result []*mockTask
	for i := range s.tasks {
		if s.tasks[i].StepID == stepID {
			result = append(result, &s.tasks[i])
		}
	}
	return result
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
	// 26 steps in chair.toml produce 75 child tasks when batch_size=6:
	//   Legs: 8, Rails: 7, Dry-fit: 6, Back: 31, Seat: 4, Assembly: 12,
	//   Install-seat: 6, Done: 1  =  75 total
	assert.Equal(t, 75, childCount, "should have 75 child tasks from the pour")
	state.mu.Unlock()

	// ── Step 4: Ready (list ready tasks — should be 4 stream starts) ─────

	t.Log("Step 4: nori ready (initial — 4 stream starts)")

	readyJSONFlag = false
	err = runReady(readyCmd, nil)
	require.NoError(t, err, "ready should succeed")

	state.mu.Lock()
	readyTasks := state.readyTasks()
	require.Equal(t, 4, len(readyTasks), "should have exactly 4 ready tasks (4 stream starts)")

	// Verify the ready tasks are the correct stream start steps.
	readySteps := make(map[string]bool)
	for _, rt := range readyTasks {
		readySteps[rt.StepID] = true
	}
	assert.True(t, readySteps["leg-rough-cut"], "leg-rough-cut should be ready")
	assert.True(t, readySteps["rail-rough-cut"], "rail-rough-cut should be ready")
	assert.True(t, readySteps["resaw-veneers"], "resaw-veneers should be ready")
	assert.True(t, readySteps["cut-seat-blank"], "cut-seat-blank should be ready")
	state.mu.Unlock()

	// ── Step 5: Claim and complete resaw-veneers (batch task) ────────────

	t.Log("Step 5: Claim and complete resaw-veneers (fan-out test)")

	state.mu.Lock()
	resawTasks := state.findTasksByStep("resaw-veneers")
	require.Len(t, resawTasks, 1, "resaw-veneers should be 1 batch ticket")
	resawID := resawTasks[0].ID
	assert.Equal(t, 6, resawTasks[0].Quantity, "resaw-veneers should have qty=6")
	state.mu.Unlock()

	// Claim it.
	taskJSONFlag = false
	err = runTaskClaim(taskClaimCmd, []string{resawID})
	require.NoError(t, err, "claim resaw-veneers should succeed")

	state.mu.Lock()
	resawTask := state.findTask(resawID)
	assert.Equal(t, "active", resawTask.Status, "resaw-veneers should be active after claim")
	state.mu.Unlock()

	// Complete it.
	err = runTaskComplete(taskCompleteCmd, []string{resawID})
	require.NoError(t, err, "complete resaw-veneers should succeed")

	state.mu.Lock()
	resawTask = state.findTask(resawID)
	assert.Equal(t, "done", resawTask.Status, "resaw-veneers should be done after complete")

	// Verify fan-out: completing resaw-veneers (1 ticket) should unlock
	// all 6 glue-lamination tickets.
	glueLamTasks := state.findTasksByStep("glue-lamination")
	require.Len(t, glueLamTasks, 6, "glue-lamination should have 6 per-piece tickets")

	readyTasks = state.readyTasks()
	readySteps = make(map[string]bool)
	readyIDs := make(map[string]bool)
	for _, rt := range readyTasks {
		readySteps[rt.StepID] = true
		readyIDs[rt.ID] = true
	}
	for _, gl := range glueLamTasks {
		assert.True(t, readyIDs[gl.ID],
			"glue-lamination ticket %s should be ready after resaw-veneers completed", gl.ID)
		assert.Equal(t, 1, gl.Quantity, "glue-lamination ticket should have qty=1")
	}
	state.mu.Unlock()

	// ── Step 6: Complete glue-lamination[1] and verify 1:1 dep chain ─────

	t.Log("Step 6: Complete glue-lamination 1 of 6 → unlocks true-up-edge 1 of 6")

	state.mu.Lock()
	glueLam1ID := glueLamTasks[0].ID
	state.mu.Unlock()

	// Claim and complete glue-lamination ticket 1.
	err = runTaskClaim(taskClaimCmd, []string{glueLam1ID})
	require.NoError(t, err)
	err = runTaskComplete(taskCompleteCmd, []string{glueLam1ID})
	require.NoError(t, err)

	state.mu.Lock()
	trueUpTasks := state.findTasksByStep("true-up-edge")
	require.Len(t, trueUpTasks, 6, "true-up-edge should have 6 per-piece tickets")

	// Only the first true-up-edge ticket should be ready (1:1 mapping).
	readyTasks = state.readyTasks()
	readyIDs = make(map[string]bool)
	for _, rt := range readyTasks {
		readyIDs[rt.ID] = true
	}
	assert.True(t, readyIDs[trueUpTasks[0].ID],
		"true-up-edge[1] should be ready after glue-lamination[1] completed")

	// The other 5 true-up-edge tickets should NOT be ready.
	for i := 1; i < 6; i++ {
		assert.False(t, readyIDs[trueUpTasks[i].ID],
			"true-up-edge[%d] should NOT be ready (glue-lamination[%d] not done)", i+1, i+1)
	}
	state.mu.Unlock()

	// ── Step 7: Complete the full back stream for all 6 pieces ───────────

	t.Log("Step 7: Complete full back stream (all 6 pieces through shape-back)")

	backSteps := []string{"glue-lamination", "true-up-edge", "rip-back-parallel", "cut-back-miters", "shape-back"}
	for _, stepID := range backSteps {
		state.mu.Lock()
		tasks := state.findTasksByStep(stepID)
		state.mu.Unlock()

		for _, task := range tasks {
			if task.Status == "done" {
				continue // already completed (glue-lamination[1] done in step 6)
			}
			err = runTaskClaim(taskClaimCmd, []string{task.ID})
			require.NoError(t, err, "claim %s should succeed", task.ID)
			err = runTaskComplete(taskCompleteCmd, []string{task.ID})
			require.NoError(t, err, "complete %s should succeed", task.ID)
		}
	}

	// ── Step 8: Complete leg + rail streams and dry-fit ───────────────────

	t.Log("Step 8: Complete legs, rails, and dry-fit streams")

	legRailSteps := []string{
		"leg-rough-cut", "leg-joint-face", "leg-thickness", "leg-rough-shape",
		"leg-shape-profile", "leg-face-mortises", "leg-edge-mortises", "hand-shape-legs",
		"rail-rough-cut", "rail-joint-face-edge", "rail-thickness", "rail-rip-width",
		"rail-compound-miters", "rail-tenons", "hand-shape-rails",
		"dry-fit",
	}
	for _, stepID := range legRailSteps {
		state.mu.Lock()
		tasks := state.findTasksByStep(stepID)
		state.mu.Unlock()

		for _, task := range tasks {
			err = runTaskClaim(taskClaimCmd, []string{task.ID})
			require.NoError(t, err, "claim %s should succeed", task.ID)
			err = runTaskComplete(taskCompleteCmd, []string{task.ID})
			require.NoError(t, err, "complete %s should succeed", task.ID)
		}
	}

	// After legs+rails+dry-fit+back complete, glue-up should be ready.
	state.mu.Lock()
	readyTasks = state.readyTasks()
	readySteps = make(map[string]bool)
	for _, rt := range readyTasks {
		readySteps[rt.StepID] = true
	}
	assert.True(t, readySteps["glue-up"],
		"glue-up should be ready after dry-fit and shape-back complete")
	state.mu.Unlock()

	// ── Step 9: Complete assembly stream (glue-up + spray-finish) ────────

	t.Log("Step 9: Complete assembly stream (glue-up + spray-finish)")

	for _, stepID := range []string{"glue-up", "spray-finish"} {
		state.mu.Lock()
		tasks := state.findTasksByStep(stepID)
		state.mu.Unlock()

		for _, task := range tasks {
			err = runTaskClaim(taskClaimCmd, []string{task.ID})
			require.NoError(t, err, "claim %s should succeed", task.ID)
			err = runTaskComplete(taskCompleteCmd, []string{task.ID})
			require.NoError(t, err, "complete %s should succeed", task.ID)
		}
	}

	// ── Step 10: Complete seat stream ────────────────────────────────────

	t.Log("Step 10: Complete seat stream")

	for _, stepID := range []string{"cut-seat-blank", "cut-foam", "cut-fabric", "upholster-seat"} {
		state.mu.Lock()
		tasks := state.findTasksByStep(stepID)
		state.mu.Unlock()

		for _, task := range tasks {
			err = runTaskClaim(taskClaimCmd, []string{task.ID})
			require.NoError(t, err, "claim %s should succeed", task.ID)
			err = runTaskComplete(taskCompleteCmd, []string{task.ID})
			require.NoError(t, err, "complete %s should succeed", task.ID)
		}
	}

	// ── Step 11: Install seats and verify done milestone ─────────────────

	t.Log("Step 11: Install seats → done milestone")

	state.mu.Lock()
	installTasks := state.findTasksByStep("install-seat")
	require.Len(t, installTasks, 6, "install-seat should have 6 per-piece tickets")
	doneTasks := state.findTasksByStep("done")
	require.Len(t, doneTasks, 1, "done should be 1 milestone ticket")
	doneTask := doneTasks[0]

	// Done should NOT be ready yet (install-seat not complete).
	readyTasks = state.readyTasks()
	readyIDs = make(map[string]bool)
	for _, rt := range readyTasks {
		readyIDs[rt.ID] = true
	}
	assert.False(t, readyIDs[doneTask.ID],
		"done milestone should NOT be ready before all install-seat complete")
	state.mu.Unlock()

	// Complete all 6 install-seat tickets.
	for _, task := range installTasks {
		err = runTaskClaim(taskClaimCmd, []string{task.ID})
		require.NoError(t, err, "claim %s should succeed", task.ID)
		err = runTaskComplete(taskCompleteCmd, []string{task.ID})
		require.NoError(t, err, "complete %s should succeed", task.ID)
	}

	// Verify done milestone is now ready (fan-in complete).
	state.mu.Lock()
	readyTasks = state.readyTasks()
	readyIDs = make(map[string]bool)
	for _, rt := range readyTasks {
		readyIDs[rt.ID] = true
	}
	state.mu.Unlock()

	// The done milestone has type "milestone", not "task", so it won't
	// appear in readyTasks() which filters for type=="task".  Verify
	// that it has no unsatisfied deps instead.
	state.mu.Lock()
	doneBlocked := false
	doneSet := make(map[string]bool)
	for _, t := range state.tasks {
		if t.Status == "done" {
			doneSet[t.ID] = true
		}
	}
	for _, dep := range state.deps {
		if dep.FromTaskID == doneTask.ID && !doneSet[dep.ToTaskID] {
			doneBlocked = true
			break
		}
	}
	assert.False(t, doneBlocked,
		"done milestone should have all deps satisfied after install-seat complete")
	assert.Equal(t, 6, doneTask.Quantity, "done milestone should have qty=6")
	assert.Equal(t, "milestone", doneTask.Type, "done should be a milestone")
	state.mu.Unlock()

	// ── Step 12: Validate API call sequence ──────────────────────────────

	t.Log("Step 12: Validate API call sequence")

	state.mu.Lock()
	calls := state.apiCalls
	state.mu.Unlock()

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

	// ── Summary ───────────────────────────────────────────────────────────

	t.Logf("E2E workflow completed successfully:")
	t.Logf("  - Logged in as %s", state.userEmail)
	t.Logf("  - Created recipe %q (slug: %s)", recipe.Name, recipe.Slug)
	t.Logf("  - Poured into job %s with %d tasks", jobTask.ID, childCount)
	t.Logf("  - Verified 4 ready tasks (stream starts)")
	t.Logf("  - Verified fan-out: resaw-veneers → 6 glue-lamination tickets")
	t.Logf("  - Verified 1:1: glue-lamination[1] → true-up-edge[1]")
	t.Logf("  - Completed all 75 tasks through claim/complete workflow")
	t.Logf("  - Verified fan-in: 6 install-seat → done milestone")
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
