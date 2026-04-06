package services

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/google/uuid"

	"github.com/tylerjvollick/nori/internal/formula"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
)

// --- Mock repositories ---

type mockRecipeRepo struct {
	recipes  map[uuid.UUID]*models.Recipe
	versions map[int]*models.RecipeVersion
}

func newMockRecipeRepo() *mockRecipeRepo {
	return &mockRecipeRepo{
		recipes:  make(map[uuid.UUID]*models.Recipe),
		versions: make(map[int]*models.RecipeVersion),
	}
}

func (m *mockRecipeRepo) GetByID(id uuid.UUID) (*models.Recipe, error) {
	r, ok := m.recipes[id]
	if !ok {
		return nil, errNotFound("recipe not found")
	}
	return r, nil
}

func (m *mockRecipeRepo) GetVersionByID(id int) (*models.RecipeVersion, error) {
	v, ok := m.versions[id]
	if !ok {
		return nil, errNotFound("recipe version not found")
	}
	return v, nil
}

func (m *mockRecipeRepo) CreateVersion(version *models.RecipeVersion) error {
	// Auto-assign an ID if not set.
	if version.ID == 0 {
		version.ID = len(m.versions) + 1
	}
	m.versions[version.ID] = version
	return nil
}

func (m *mockRecipeRepo) ListVersions(recipeID uuid.UUID) ([]models.RecipeVersion, error) {
	var result []models.RecipeVersion
	for _, v := range m.versions {
		if v.RecipeID == recipeID {
			result = append(result, *v)
		}
	}
	return result, nil
}

type mockTaskRepo struct {
	tasks map[string]*models.Task
}

func newMockTaskRepo() *mockTaskRepo {
	return &mockTaskRepo{tasks: make(map[string]*models.Task)}
}

func (m *mockTaskRepo) Create(task *models.Task) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepo) GetByID(id string) (*models.Task, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, errNotFound("task not found")
	}
	return t, nil
}

func (m *mockTaskRepo) List(_ repositories.TaskFilter) ([]models.Task, int64, error) {
	var tasks []models.Task
	for _, t := range m.tasks {
		tasks = append(tasks, *t)
	}
	return tasks, int64(len(tasks)), nil
}

func (m *mockTaskRepo) Update(task *models.Task) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepo) Delete(id string) error {
	delete(m.tasks, id)
	return nil
}

func (m *mockTaskRepo) GetChildren(parentID string) ([]models.Task, error) {
	var children []models.Task
	for _, t := range m.tasks {
		if t.ParentID != nil && *t.ParentID == parentID {
			children = append(children, *t)
		}
	}
	return children, nil
}

func (m *mockTaskRepo) GetRoot(taskID string) (*models.Task, error) {
	t, ok := m.tasks[taskID]
	if !ok {
		return nil, errNotFound("task not found")
	}
	for t.ParentID != nil {
		t, ok = m.tasks[*t.ParentID]
		if !ok {
			return nil, errNotFound("parent not found")
		}
	}
	return t, nil
}

type mockTaskDepRepo struct {
	deps []models.TaskDep
}

func newMockTaskDepRepo() *mockTaskDepRepo {
	return &mockTaskDepRepo{}
}

func (m *mockTaskDepRepo) AddDep(dep *models.TaskDep) error {
	m.deps = append(m.deps, *dep)
	return nil
}

func (m *mockTaskDepRepo) RemoveDep(_, _ string) error { return nil }

func (m *mockTaskDepRepo) GetBlockers(taskID string) ([]models.TaskDep, error) {
	var result []models.TaskDep
	for _, d := range m.deps {
		if d.ToTaskID == taskID && d.Type == models.DepTypeBlocks {
			result = append(result, d)
		}
	}
	return result, nil
}

func (m *mockTaskDepRepo) GetDependents(taskID string) ([]models.TaskDep, error) {
	var result []models.TaskDep
	for _, d := range m.deps {
		if d.FromTaskID == taskID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (m *mockTaskDepRepo) GetAllForTask(taskID string) ([]models.TaskDep, error) {
	var result []models.TaskDep
	for _, d := range m.deps {
		if d.FromTaskID == taskID || d.ToTaskID == taskID {
			result = append(result, d)
		}
	}
	return result, nil
}

type errNotFound string

func (e errNotFound) Error() string { return string(e) }

// --- Helpers ---

// simpleTOML returns a minimal recipe TOML with two steps and a dependency.
func simpleTOML() string {
	return `
formula = "simple-build"
description = "Build {{product}}"
version = 1
type = "workflow"

[vars.product]
description = "Product name"
required = true

[[steps]]
id = "cut"
title = "Cut {{product}} parts"
type = "task"

[[steps]]
id = "assemble"
title = "Assemble {{product}}"
type = "task"
depends_on = ["cut"]
`
}

// conditionalTOML returns a TOML with a conditional step.
func conditionalTOML() string {
	return `
formula = "cond-build"
description = "Build with optional finish"
version = 1
type = "workflow"

[vars.finish]
default = "false"

[[steps]]
id = "cut"
title = "Cut parts"
type = "task"

[[steps]]
id = "sand"
title = "Sand parts"
type = "task"
depends_on = ["cut"]

[[steps]]
id = "finish-coat"
title = "Apply finish coat"
type = "task"
depends_on = ["sand"]
condition = "{{finish}}"
`
}

func setupRecipeService(tomlContent string) (*RecipeService, uuid.UUID, uuid.UUID, uuid.UUID) {
	recipeRepo := newMockRecipeRepo()
	taskRepo := newMockTaskRepo()
	depRepo := newMockTaskDepRepo()

	recipeID := uuid.New()
	spaceID := uuid.New()
	userID := uuid.New()
	versionID := 1

	recipeRepo.recipes[recipeID] = &models.Recipe{
		ID:               recipeID,
		SpaceID:          spaceID,
		Name:             "Test Recipe",
		Slug:             "test-recipe",
		CurrentVersionID: &versionID,
		CreatedByID:      userID,
		IsActive:         true,
	}

	recipeRepo.versions[versionID] = &models.RecipeVersion{
		ID:            versionID,
		RecipeID:      recipeID,
		VersionNumber: 1,
		Status:        models.RecipeVersionStatusPublished,
		Content:       tomlContent,
		AuthorID:      userID,
	}

	svc := NewRecipeService(nil, recipeRepo, taskRepo, depRepo)
	return svc, recipeID, spaceID, userID
}

// --- Tests ---

func TestPourRecipe_BasicWorkflow(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(simpleTOML())

	vars := map[string]string{"product": "Dining Table"}
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	// Root task should be a job.
	if rootTask.Type != models.TaskTypeJob {
		t.Errorf("expected root type %q, got %q", models.TaskTypeJob, rootTask.Type)
	}

	// Title should have variables substituted.
	if rootTask.Title != "Build Dining Table" {
		t.Errorf("expected title %q, got %q", "Build Dining Table", rootTask.Title)
	}

	// Should have recipe linkage.
	if rootTask.RecipeID == nil || *rootTask.RecipeID != recipeID {
		t.Errorf("expected recipeID %s, got %v", recipeID, rootTask.RecipeID)
	}

	// Check child tasks were created (2 steps).
	taskRepo := svc.taskRepo.(*mockTaskRepo)
	childCount := 0
	for _, task := range taskRepo.tasks {
		if task.ParentID != nil && *task.ParentID == rootTask.ID {
			childCount++
		}
	}
	if childCount != 2 {
		t.Errorf("expected 2 child tasks, got %d", childCount)
	}

	// Check dependency edge was created (assemble depends on cut).
	depRepo := svc.taskDepRepo.(*mockTaskDepRepo)
	if len(depRepo.deps) != 1 {
		t.Errorf("expected 1 dependency edge, got %d", len(depRepo.deps))
	}
}

func TestPourRecipe_VariableSubstitution(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(simpleTOML())

	vars := map[string]string{"product": "Bookshelf"}
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	// Check child task titles have substituted vars.
	taskRepo := svc.taskRepo.(*mockTaskRepo)
	foundCut := false
	foundAssemble := false
	for _, task := range taskRepo.tasks {
		if task.Title == "Cut Bookshelf parts" {
			foundCut = true
		}
		if task.Title == "Assemble Bookshelf" {
			foundAssemble = true
		}
	}

	if !foundCut {
		t.Error("expected child task with title 'Cut Bookshelf parts'")
	}
	if !foundAssemble {
		t.Error("expected child task with title 'Assemble Bookshelf'")
	}

	_ = rootTask
}

func TestPourRecipe_MissingRequiredVar(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(simpleTOML())

	// Don't provide the required "product" variable.
	_, err := svc.PourRecipe(recipeID, spaceID, userID, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing required variable, got nil")
	}
}

func TestPourRecipe_NoPublishedVersion(t *testing.T) {
	recipeRepo := newMockRecipeRepo()
	taskRepo := newMockTaskRepo()
	depRepo := newMockTaskDepRepo()

	recipeID := uuid.New()
	spaceID := uuid.New()
	userID := uuid.New()

	// Recipe without a current version.
	recipeRepo.recipes[recipeID] = &models.Recipe{
		ID:               recipeID,
		SpaceID:          spaceID,
		Name:             "Empty Recipe",
		Slug:             "empty-recipe",
		CurrentVersionID: nil,
		CreatedByID:      userID,
		IsActive:         true,
	}

	svc := NewRecipeService(nil, recipeRepo, taskRepo, depRepo)
	_, err := svc.PourRecipe(recipeID, spaceID, userID, nil, nil)
	if err == nil {
		t.Fatal("expected error for recipe with no published version, got nil")
	}
}

func TestPourRecipe_ConditionalStepExcluded(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(conditionalTOML())

	// Don't set finish variable (default "false"), so the finish-coat step should be excluded.
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, nil, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	taskRepo := svc.taskRepo.(*mockTaskRepo)
	childCount := 0
	for _, task := range taskRepo.tasks {
		if task.ParentID != nil && *task.ParentID == rootTask.ID {
			childCount++
		}
	}
	// Only "cut" and "sand" should exist (finish-coat excluded).
	if childCount != 2 {
		t.Errorf("expected 2 child tasks (finish excluded), got %d", childCount)
	}
}

func TestPourRecipe_ConditionalStepIncluded(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(conditionalTOML())

	// Set finish=true so the finish-coat step should be included.
	vars := map[string]string{"finish": "true"}
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	taskRepo := svc.taskRepo.(*mockTaskRepo)
	childCount := 0
	for _, task := range taskRepo.tasks {
		if task.ParentID != nil && *task.ParentID == rootTask.ID {
			childCount++
		}
	}
	// All 3 steps should be created.
	if childCount != 3 {
		t.Errorf("expected 3 child tasks (finish included), got %d", childCount)
	}

	// Should have 2 dependency edges: sand→cut, finish-coat→sand.
	depRepo := svc.taskDepRepo.(*mockTaskDepRepo)
	if len(depRepo.deps) != 2 {
		t.Errorf("expected 2 dependency edges, got %d", len(depRepo.deps))
	}
}

func TestPourRecipe_HierarchicalIDs(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(simpleTOML())

	vars := map[string]string{"product": "Chair"}
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	// Check that child task IDs are hierarchical.
	taskRepo := svc.taskRepo.(*mockTaskRepo)
	expectedID1 := rootTask.ID + ".1"
	expectedID2 := rootTask.ID + ".2"

	if _, ok := taskRepo.tasks[expectedID1]; !ok {
		t.Errorf("expected task with hierarchical ID %q", expectedID1)
	}
	if _, ok := taskRepo.tasks[expectedID2]; !ok {
		t.Errorf("expected task with hierarchical ID %q", expectedID2)
	}
}

func TestPourRecipe_WithOrderID(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(simpleTOML())

	orderID := uuid.New()
	vars := map[string]string{"product": "Desk"}
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, &orderID)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	// Root task should have order ID in metadata.
	if rootTask.Metadata == nil {
		t.Fatal("expected metadata with orderID, got nil")
	}
	if rootTask.Metadata["orderID"] != orderID.String() {
		t.Errorf("expected orderID %q in metadata, got %q", orderID.String(), rootTask.Metadata["orderID"])
	}
}

func TestPourRecipe_DependencyEdgeDirection(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(simpleTOML())

	vars := map[string]string{"product": "Table"}
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	// "assemble" depends_on "cut"
	// In TaskDep model: FromTaskID = assemble task, ToTaskID = cut task
	depRepo := svc.taskDepRepo.(*mockTaskDepRepo)
	if len(depRepo.deps) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(depRepo.deps))
	}

	dep := depRepo.deps[0]
	// assemble is step index 1 → child ID = root.2
	// cut is step index 0 → child ID = root.1
	expectedFrom := rootTask.ID + ".2" // assemble
	expectedTo := rootTask.ID + ".1"   // cut

	if dep.FromTaskID != expectedFrom {
		t.Errorf("expected FromTaskID %q, got %q", expectedFrom, dep.FromTaskID)
	}
	if dep.ToTaskID != expectedTo {
		t.Errorf("expected ToTaskID %q, got %q", expectedTo, dep.ToTaskID)
	}
	if dep.Type != models.DepTypeBlocks {
		t.Errorf("expected dep type %q, got %q", models.DepTypeBlocks, dep.Type)
	}
}

func TestGenerateTaskID(t *testing.T) {
	id := generateTaskID()

	// Should start with "nori-".
	if len(id) < 6 || id[:5] != "nori-" {
		t.Errorf("expected ID starting with 'nori-', got %q", id)
	}

	// Should be unique.
	id2 := generateTaskID()
	if id == id2 {
		t.Errorf("expected unique IDs, got same: %q", id)
	}
}

// --- Diff Tests ---

// setupDiffTest pours a recipe and returns the service, root task, and recipe version ID
// for use in diff tests. Optionally mutates the poured task tree before returning.
func setupDiffTest(t *testing.T, tomlContent string, vars map[string]string) (*RecipeService, *models.Task, int) {
	t.Helper()
	svc, recipeID, spaceID, userID := setupRecipeService(tomlContent)

	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	return svc, rootTask, 1 // version ID is always 1 in setupRecipeService
}

func TestDiffJobToRecipe_NoDifferences(t *testing.T) {
	svc, rootTask, versionID := setupDiffTest(t, simpleTOML(), map[string]string{"product": "Dining Table"})

	diff, err := svc.DiffJobToRecipe(rootTask.ID, versionID)
	if err != nil {
		t.Fatalf("DiffJobToRecipe failed: %v", err)
	}

	if diff.JobID != rootTask.ID {
		t.Errorf("expected JobID %q, got %q", rootTask.ID, diff.JobID)
	}
	if diff.RecipeVersionID != versionID {
		t.Errorf("expected RecipeVersionID %d, got %d", versionID, diff.RecipeVersionID)
	}
	if len(diff.Added) != 0 {
		t.Errorf("expected 0 added, got %d: %+v", len(diff.Added), diff.Added)
	}
	if len(diff.Removed) != 0 {
		t.Errorf("expected 0 removed, got %d: %+v", len(diff.Removed), diff.Removed)
	}
	// Modified might be non-zero because diff uses default vars (no product var
	// substituted), so titles will differ. This tests the "structural match" case.
	// The recipe defaults apply "" for required vars, so titles with {{product}}
	// will resolve to empty substitution.
}

func TestDiffJobToRecipe_AddedTask(t *testing.T) {
	svc, rootTask, versionID := setupDiffTest(t, simpleTOML(), map[string]string{"product": "Table"})

	// Add an extra task that isn't in the recipe.
	taskRepo := svc.taskRepo.(*mockTaskRepo)
	extraID := rootTask.ID + ".3"
	parentID := rootTask.ID
	taskRepo.tasks[extraID] = &models.Task{
		ID:          extraID,
		SpaceID:     rootTask.SpaceID,
		ParentID:    &parentID,
		Title:       "Extra finishing step",
		Type:        models.TaskTypeTask,
		Status:      models.TaskStatusOpen,
		CreatedByID: rootTask.CreatedByID,
	}

	diff, err := svc.DiffJobToRecipe(rootTask.ID, versionID)
	if err != nil {
		t.Fatalf("DiffJobToRecipe failed: %v", err)
	}

	if len(diff.Added) != 1 {
		t.Fatalf("expected 1 added, got %d: %+v", len(diff.Added), diff.Added)
	}

	added := diff.Added[0]
	if added.Path != "3" {
		t.Errorf("expected added path %q, got %q", "3", added.Path)
	}
	if added.TaskID != extraID {
		t.Errorf("expected added TaskID %q, got %q", extraID, added.TaskID)
	}
	if added.Title != "Extra finishing step" {
		t.Errorf("expected added title %q, got %q", "Extra finishing step", added.Title)
	}
	if added.ChangeType != DiffChangeAdded {
		t.Errorf("expected change type %q, got %q", DiffChangeAdded, added.ChangeType)
	}
}

func TestDiffJobToRecipe_RemovedTask(t *testing.T) {
	svc, rootTask, versionID := setupDiffTest(t, simpleTOML(), map[string]string{"product": "Table"})

	// Remove the second child task (assemble) from the job.
	taskRepo := svc.taskRepo.(*mockTaskRepo)
	secondChildID := rootTask.ID + ".2"
	delete(taskRepo.tasks, secondChildID)

	diff, err := svc.DiffJobToRecipe(rootTask.ID, versionID)
	if err != nil {
		t.Fatalf("DiffJobToRecipe failed: %v", err)
	}

	if len(diff.Removed) != 1 {
		t.Fatalf("expected 1 removed, got %d: %+v", len(diff.Removed), diff.Removed)
	}

	removed := diff.Removed[0]
	if removed.Path != "2" {
		t.Errorf("expected removed path %q, got %q", "2", removed.Path)
	}
	if removed.StepID != "assemble" {
		t.Errorf("expected removed StepID %q, got %q", "assemble", removed.StepID)
	}
	if removed.ChangeType != DiffChangeRemoved {
		t.Errorf("expected change type %q, got %q", DiffChangeRemoved, removed.ChangeType)
	}
}

func TestDiffJobToRecipe_ModifiedTitle(t *testing.T) {
	svc, rootTask, versionID := setupDiffTest(t, simpleTOML(), map[string]string{"product": "Table"})

	// Modify the first child's title.
	taskRepo := svc.taskRepo.(*mockTaskRepo)
	firstChildID := rootTask.ID + ".1"
	task := taskRepo.tasks[firstChildID]
	task.Title = "Rough cut Table parts (modified)"

	diff, err := svc.DiffJobToRecipe(rootTask.ID, versionID)
	if err != nil {
		t.Fatalf("DiffJobToRecipe failed: %v", err)
	}

	// Find the modified item for path "1".
	var found *DiffItem
	for i := range diff.Modified {
		if diff.Modified[i].Path == "1" {
			found = &diff.Modified[i]
			break
		}
	}

	if found == nil {
		t.Fatalf("expected modified item at path '1', got modified: %+v", diff.Modified)
	}

	if found.Title != "Rough cut Table parts (modified)" {
		t.Errorf("expected modified title %q, got %q", "Rough cut Table parts (modified)", found.Title)
	}
	if found.ChangeType != DiffChangeModified {
		t.Errorf("expected change type %q, got %q", DiffChangeModified, found.ChangeType)
	}
	if found.TaskID != firstChildID {
		t.Errorf("expected TaskID %q, got %q", firstChildID, found.TaskID)
	}
	if found.StepID != "cut" {
		t.Errorf("expected StepID %q, got %q", "cut", found.StepID)
	}
}

func TestDiffJobToRecipe_ModifiedDescription(t *testing.T) {
	// Use a TOML with descriptions on steps.
	toml := `
formula = "desc-test"
description = "Build {{product}}"
version = 1
type = "workflow"

[vars.product]
description = "Product name"
default = "Widget"

[[steps]]
id = "prep"
title = "Prep {{product}}"
description = "Prepare all {{product}} materials"
type = "task"
`
	svc, rootTask, versionID := setupDiffTest(t, toml, map[string]string{"product": "Widget"})

	// Modify the description.
	taskRepo := svc.taskRepo.(*mockTaskRepo)
	firstChildID := rootTask.ID + ".1"
	task := taskRepo.tasks[firstChildID]
	newDesc := "Prepare all Widget materials and tools"
	task.Description = &newDesc

	diff, err := svc.DiffJobToRecipe(rootTask.ID, versionID)
	if err != nil {
		t.Fatalf("DiffJobToRecipe failed: %v", err)
	}

	if len(diff.Modified) != 1 {
		t.Fatalf("expected 1 modified, got %d: %+v", len(diff.Modified), diff.Modified)
	}

	mod := diff.Modified[0]
	if mod.Description != "Prepare all Widget materials and tools" {
		t.Errorf("expected modified description %q, got %q", "Prepare all Widget materials and tools", mod.Description)
	}
	if mod.ExpectedDescription != "Prepare all Widget materials" {
		t.Errorf("expected expected description %q, got %q", "Prepare all Widget materials", mod.ExpectedDescription)
	}
}

func TestDiffJobToRecipe_MixedChanges(t *testing.T) {
	// Use simpleTOML (no conditionals) so the recipe side has a predictable shape.
	svc, rootTask, versionID := setupDiffTest(t, simpleTOML(), map[string]string{"product": "Table"})

	taskRepo := svc.taskRepo.(*mockTaskRepo)

	// Modify the first task's title (cut).
	firstChildID := rootTask.ID + ".1"
	taskRepo.tasks[firstChildID].Title = "Rough cut Table parts"

	// Remove the second task (assemble).
	secondChildID := rootTask.ID + ".2"
	delete(taskRepo.tasks, secondChildID)

	// Add an extra task.
	extraID := rootTask.ID + ".3"
	parentID := rootTask.ID
	taskRepo.tasks[extraID] = &models.Task{
		ID:          extraID,
		SpaceID:     rootTask.SpaceID,
		ParentID:    &parentID,
		Title:       "Quality check",
		Type:        models.TaskTypeTask,
		Status:      models.TaskStatusOpen,
		CreatedByID: rootTask.CreatedByID,
	}

	diff, err := svc.DiffJobToRecipe(rootTask.ID, versionID)
	if err != nil {
		t.Fatalf("DiffJobToRecipe failed: %v", err)
	}

	if len(diff.Added) != 1 {
		t.Errorf("expected 1 added, got %d: %+v", len(diff.Added), diff.Added)
	}
	if len(diff.Removed) != 1 {
		t.Errorf("expected 1 removed, got %d: %+v", len(diff.Removed), diff.Removed)
	}
	// The title for step "cut" was changed: recipe has "Cut  parts" (defaults apply
	// empty string for required vars with no default), job has "Rough cut Table parts".
	foundModified := false
	for _, m := range diff.Modified {
		if m.Path == "1" && m.Title == "Rough cut Table parts" {
			foundModified = true
		}
	}
	if !foundModified {
		t.Errorf("expected modified item at path '1' with title 'Rough cut Table parts', got: %+v", diff.Modified)
	}
}

func TestDiffJobToRecipe_NotAJob(t *testing.T) {
	svc, rootTask, versionID := setupDiffTest(t, simpleTOML(), map[string]string{"product": "Table"})

	// Try to diff a child task (not a job).
	childID := rootTask.ID + ".1"
	_, err := svc.DiffJobToRecipe(childID, versionID)
	if err == nil {
		t.Fatal("expected error for non-job task, got nil")
	}
}

func TestDiffJobToRecipe_InvalidVersion(t *testing.T) {
	svc, rootTask, _ := setupDiffTest(t, simpleTOML(), map[string]string{"product": "Table"})

	_, err := svc.DiffJobToRecipe(rootTask.ID, 999)
	if err == nil {
		t.Fatal("expected error for invalid version ID, got nil")
	}
}

func TestDiffJobToRecipe_InvalidJobID(t *testing.T) {
	svc, _, versionID := setupDiffTest(t, simpleTOML(), map[string]string{"product": "Table"})

	_, err := svc.DiffJobToRecipe("nonexistent-id", versionID)
	if err == nil {
		t.Fatal("expected error for invalid job ID, got nil")
	}
}

func TestExtractPath(t *testing.T) {
	tests := []struct {
		taskID   string
		rootID   string
		expected string
	}{
		{"nori-abc123.1", "nori-abc123", "1"},
		{"nori-abc123.1.2", "nori-abc123", "1.2"},
		{"nori-abc123.1.2.3", "nori-abc123", "1.2.3"},
		{"other-id", "nori-abc123", ""},     // doesn't match
		{"nori-abc123", "nori-abc123", ""},  // no suffix
		{"nori-abc123.", "nori-abc123", ""}, // empty suffix
	}

	for _, tt := range tests {
		result := extractPath(tt.taskID, tt.rootID)
		if result != tt.expected {
			t.Errorf("extractPath(%q, %q) = %q, want %q", tt.taskID, tt.rootID, result, tt.expected)
		}
	}
}

// --- Promote Tests ---

// setupPromoteTest pours a recipe and returns the service, root task, recipe ID,
// and recipe version ID for use in promote tests.
func setupPromoteTest(t *testing.T, tomlContent string, vars map[string]string) (*RecipeService, *models.Task, uuid.UUID, int) {
	t.Helper()
	svc, recipeID, spaceID, userID := setupRecipeService(tomlContent)

	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	return svc, rootTask, recipeID, 1 // version ID is always 1 in setupRecipeService
}

func TestPromoteJobToRecipe_NoDiff(t *testing.T) {
	svc, rootTask, recipeID, _ := setupPromoteTest(t, simpleTOML(), map[string]string{"product": "Table"})

	newVersion, err := svc.PromoteJobToRecipe(rootTask.ID, recipeID, "No real changes")
	if err != nil {
		t.Fatalf("PromoteJobToRecipe failed: %v", err)
	}

	// Should create a new draft version.
	if newVersion.Status != models.RecipeVersionStatusDraft {
		t.Errorf("expected status %q, got %q", models.RecipeVersionStatusDraft, newVersion.Status)
	}
	if newVersion.VersionNumber != 2 {
		t.Errorf("expected version number 2, got %d", newVersion.VersionNumber)
	}
	if newVersion.RecipeID != recipeID {
		t.Errorf("expected recipe ID %s, got %s", recipeID, newVersion.RecipeID)
	}
	if newVersion.ChangeSummary == nil || *newVersion.ChangeSummary != "No real changes" {
		t.Errorf("expected change summary %q, got %v", "No real changes", newVersion.ChangeSummary)
	}

	// Content should be valid TOML.
	if newVersion.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestPromoteJobToRecipe_ModifiedTitle(t *testing.T) {
	svc, rootTask, recipeID, _ := setupPromoteTest(t, simpleTOML(), map[string]string{"product": "Table"})

	// Modify the first child's title in the job.
	taskRepo := svc.taskRepo.(*mockTaskRepo)
	firstChildID := rootTask.ID + ".1"
	taskRepo.tasks[firstChildID].Title = "Rough cut Table parts"

	newVersion, err := svc.PromoteJobToRecipe(rootTask.ID, recipeID, "Updated cut step title")
	if err != nil {
		t.Fatalf("PromoteJobToRecipe failed: %v", err)
	}

	if newVersion.Status != models.RecipeVersionStatusDraft {
		t.Errorf("expected status %q, got %q", models.RecipeVersionStatusDraft, newVersion.Status)
	}

	// Parse the new content and verify the title was updated.
	parser := formula.NewParser()
	f, err := parser.ParseTOML([]byte(newVersion.Content))
	if err != nil {
		t.Fatalf("failed to parse new version TOML: %v", err)
	}

	if len(f.Steps) < 1 {
		t.Fatalf("expected at least 1 step, got %d", len(f.Steps))
	}

	// The first step's title should reflect the job's modified title.
	if f.Steps[0].Title != "Rough cut Table parts" {
		t.Errorf("expected step title %q, got %q", "Rough cut Table parts", f.Steps[0].Title)
	}
}

func TestPromoteJobToRecipe_AddedTask(t *testing.T) {
	svc, rootTask, recipeID, _ := setupPromoteTest(t, simpleTOML(), map[string]string{"product": "Table"})

	// Add an extra task in the job.
	taskRepo := svc.taskRepo.(*mockTaskRepo)
	extraID := rootTask.ID + ".3"
	parentID := rootTask.ID
	taskRepo.tasks[extraID] = &models.Task{
		ID:          extraID,
		SpaceID:     rootTask.SpaceID,
		ParentID:    &parentID,
		Title:       "Quality check",
		Type:        models.TaskTypeTask,
		Status:      models.TaskStatusOpen,
		CreatedByID: rootTask.CreatedByID,
	}

	newVersion, err := svc.PromoteJobToRecipe(rootTask.ID, recipeID, "Added quality check step")
	if err != nil {
		t.Fatalf("PromoteJobToRecipe failed: %v", err)
	}

	// Parse the new content and verify the added step.
	parser := formula.NewParser()
	f, err := parser.ParseTOML([]byte(newVersion.Content))
	if err != nil {
		t.Fatalf("failed to parse new version TOML: %v", err)
	}

	// Should have 3 steps (original 2 + 1 added).
	if len(f.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(f.Steps))
	}

	// The third step should be the added one.
	found := false
	for _, step := range f.Steps {
		if step.Title == "Quality check" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected added step with title 'Quality check' in new version")
	}
}

func TestPromoteJobToRecipe_RemovedTask(t *testing.T) {
	svc, rootTask, recipeID, _ := setupPromoteTest(t, simpleTOML(), map[string]string{"product": "Table"})

	// Remove the second child task from the job.
	taskRepo := svc.taskRepo.(*mockTaskRepo)
	secondChildID := rootTask.ID + ".2"
	delete(taskRepo.tasks, secondChildID)

	newVersion, err := svc.PromoteJobToRecipe(rootTask.ID, recipeID, "Removed assemble step")
	if err != nil {
		t.Fatalf("PromoteJobToRecipe failed: %v", err)
	}

	// Parse the new content and verify the step was removed.
	parser := formula.NewParser()
	f, err := parser.ParseTOML([]byte(newVersion.Content))
	if err != nil {
		t.Fatalf("failed to parse new version TOML: %v", err)
	}

	// Should have 1 step (original 2 - 1 removed).
	if len(f.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(f.Steps))
	}

	// The remaining step should be "cut", not "assemble".
	if f.Steps[0].ID != "cut" {
		t.Errorf("expected remaining step ID %q, got %q", "cut", f.Steps[0].ID)
	}
}

func TestPromoteJobToRecipe_MixedChanges(t *testing.T) {
	svc, rootTask, recipeID, _ := setupPromoteTest(t, simpleTOML(), map[string]string{"product": "Table"})

	taskRepo := svc.taskRepo.(*mockTaskRepo)

	// Modify the first task's title.
	firstChildID := rootTask.ID + ".1"
	taskRepo.tasks[firstChildID].Title = "Rough cut Table parts"

	// Remove the second task.
	secondChildID := rootTask.ID + ".2"
	delete(taskRepo.tasks, secondChildID)

	// Add an extra task.
	extraID := rootTask.ID + ".3"
	parentID := rootTask.ID
	taskRepo.tasks[extraID] = &models.Task{
		ID:          extraID,
		SpaceID:     rootTask.SpaceID,
		ParentID:    &parentID,
		Title:       "Finishing",
		Type:        models.TaskTypeTask,
		Status:      models.TaskStatusOpen,
		CreatedByID: rootTask.CreatedByID,
	}

	newVersion, err := svc.PromoteJobToRecipe(rootTask.ID, recipeID, "Major rework")
	if err != nil {
		t.Fatalf("PromoteJobToRecipe failed: %v", err)
	}

	parser := formula.NewParser()
	f, err := parser.ParseTOML([]byte(newVersion.Content))
	if err != nil {
		t.Fatalf("failed to parse new version TOML: %v", err)
	}

	// Should have 2 steps: modified "cut" + added "Finishing" (assemble removed).
	if len(f.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(f.Steps))
	}

	// Verify the modified step.
	if f.Steps[0].Title != "Rough cut Table parts" {
		t.Errorf("expected first step title %q, got %q", "Rough cut Table parts", f.Steps[0].Title)
	}

	// Verify the added step exists.
	foundFinishing := false
	for _, step := range f.Steps {
		if step.Title == "Finishing" {
			foundFinishing = true
		}
	}
	if !foundFinishing {
		t.Error("expected step with title 'Finishing'")
	}
}

func TestPromoteJobToRecipe_NotAJob(t *testing.T) {
	svc, rootTask, recipeID, _ := setupPromoteTest(t, simpleTOML(), map[string]string{"product": "Table"})

	// Try to promote a child task (not a job).
	childID := rootTask.ID + ".1"
	_, err := svc.PromoteJobToRecipe(childID, recipeID, "Should fail")
	if err == nil {
		t.Fatal("expected error for non-job task, got nil")
	}
}

func TestPromoteJobToRecipe_WrongRecipe(t *testing.T) {
	svc, rootTask, _, _ := setupPromoteTest(t, simpleTOML(), map[string]string{"product": "Table"})

	// Try to promote with a different recipe ID than the job's source.
	wrongRecipeID := uuid.New()
	_, err := svc.PromoteJobToRecipe(rootTask.ID, wrongRecipeID, "Should fail")
	if err == nil {
		t.Fatal("expected error for wrong recipe ID, got nil")
	}
}

func TestPromoteJobToRecipe_VersionNumberIncrement(t *testing.T) {
	svc, rootTask, recipeID, _ := setupPromoteTest(t, simpleTOML(), map[string]string{"product": "Table"})

	// First promote.
	v1, err := svc.PromoteJobToRecipe(rootTask.ID, recipeID, "First promote")
	if err != nil {
		t.Fatalf("first PromoteJobToRecipe failed: %v", err)
	}
	if v1.VersionNumber != 2 {
		t.Errorf("expected version number 2, got %d", v1.VersionNumber)
	}

	// Second promote.
	v2, err := svc.PromoteJobToRecipe(rootTask.ID, recipeID, "Second promote")
	if err != nil {
		t.Fatalf("second PromoteJobToRecipe failed: %v", err)
	}
	if v2.VersionNumber != 3 {
		t.Errorf("expected version number 3, got %d", v2.VersionNumber)
	}
}

func TestPromoteJobToRecipe_EmptyChangeSummary(t *testing.T) {
	svc, rootTask, recipeID, _ := setupPromoteTest(t, simpleTOML(), map[string]string{"product": "Table"})

	newVersion, err := svc.PromoteJobToRecipe(rootTask.ID, recipeID, "")
	if err != nil {
		t.Fatalf("PromoteJobToRecipe failed: %v", err)
	}

	if newVersion.ChangeSummary != nil {
		t.Errorf("expected nil change summary for empty string, got %q", *newVersion.ChangeSummary)
	}
}

func TestPromoteJobToRecipe_PreservesFormulaMetadata(t *testing.T) {
	svc, rootTask, recipeID, _ := setupPromoteTest(t, simpleTOML(), map[string]string{"product": "Table"})

	newVersion, err := svc.PromoteJobToRecipe(rootTask.ID, recipeID, "Preserve metadata")
	if err != nil {
		t.Fatalf("PromoteJobToRecipe failed: %v", err)
	}

	// Parse the new content and verify formula metadata is preserved.
	parser := formula.NewParser()
	f, err := parser.ParseTOML([]byte(newVersion.Content))
	if err != nil {
		t.Fatalf("failed to parse new version TOML: %v", err)
	}

	if f.Formula != "simple-build" {
		t.Errorf("expected formula name %q, got %q", "simple-build", f.Formula)
	}
	if f.Version != 1 {
		t.Errorf("expected formula version 1, got %d", f.Version)
	}
	if f.Type != formula.TypeWorkflow {
		t.Errorf("expected formula type %q, got %q", formula.TypeWorkflow, f.Type)
	}

	// Vars should be preserved.
	productVar, ok := f.Vars["product"]
	if !ok {
		t.Fatal("expected 'product' variable to be preserved")
	}
	if !productVar.Required {
		t.Error("expected 'product' variable to be required")
	}
}

func TestPromoteJobToRecipe_PreservesDependencies(t *testing.T) {
	svc, rootTask, recipeID, _ := setupPromoteTest(t, simpleTOML(), map[string]string{"product": "Table"})

	newVersion, err := svc.PromoteJobToRecipe(rootTask.ID, recipeID, "Check deps")
	if err != nil {
		t.Fatalf("PromoteJobToRecipe failed: %v", err)
	}

	parser := formula.NewParser()
	f, err := parser.ParseTOML([]byte(newVersion.Content))
	if err != nil {
		t.Fatalf("failed to parse new version TOML: %v", err)
	}

	// The "assemble" step should still depend on "cut".
	if len(f.Steps) < 2 {
		t.Fatalf("expected at least 2 steps, got %d", len(f.Steps))
	}

	assembleStep := f.Steps[1]
	if assembleStep.ID != "assemble" {
		t.Fatalf("expected second step ID %q, got %q", "assemble", assembleStep.ID)
	}
	if len(assembleStep.DependsOn) != 1 || assembleStep.DependsOn[0] != "cut" {
		t.Errorf("expected assemble depends_on [cut], got %v", assembleStep.DependsOn)
	}
}

// --- Batch-Aware Pour Tests ---

// batchTOML returns a recipe with explicit batch_size on steps.
// order_qty=6, cut has batch_size=6 (1 batch ticket), sand has batch_size=1 (6 per-piece tickets).
func batchTOML() string {
	return `
formula = "batch-build"
description = "Build {{product}}"
version = 1
type = "workflow"

[vars.product]
description = "Product name"
required = true

[vars.order_qty]
description = "Number of pieces to produce"
required = true

[vars.batch_size]
default = "1"

[[steps]]
id = "cut"
title = "Cut all parts"
type = "task"
batch_size = 6

[[steps]]
id = "sand"
title = "Sand piece — {{n}} of {{batch_count}}"
type = "task"
batch_size = 1
depends_on = ["cut"]
`
}

func TestPourRecipe_BatchStep_OneBatchTicket(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(batchTOML())

	vars := map[string]string{"product": "Chair", "order_qty": "6"}
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	taskRepo := svc.taskRepo.(*mockTaskRepo)

	// "cut" has batch_size=6, order_qty=6 → ticket_count=1 → 1 task.
	cutTaskID := rootTask.ID + ".1"
	cutTask, ok := taskRepo.tasks[cutTaskID]
	if !ok {
		t.Fatalf("expected batch task at %q", cutTaskID)
	}
	if cutTask.Title != "Cut all parts" {
		t.Errorf("expected title %q, got %q", "Cut all parts", cutTask.Title)
	}
	if cutTask.Quantity != 6 {
		t.Errorf("expected Quantity 6, got %d", cutTask.Quantity)
	}
}

func TestPourRecipe_PerPieceStep_MultipleTickets(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(batchTOML())

	vars := map[string]string{"product": "Chair", "order_qty": "6"}
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	taskRepo := svc.taskRepo.(*mockTaskRepo)

	// "sand" has batch_size=1, order_qty=6 → ticket_count=6 → 6 tasks.
	// They occupy positions .2 through .7 (cut is .1).
	for n := 1; n <= 6; n++ {
		taskID := fmt.Sprintf("%s.%d", rootTask.ID, n+1) // +1 because cut is .1
		task, ok := taskRepo.tasks[taskID]
		if !ok {
			t.Fatalf("expected per-piece task at %q", taskID)
		}

		expectedTitle := fmt.Sprintf("Sand piece — %d of 6", n)
		if task.Title != expectedTitle {
			t.Errorf("task %s: expected title %q, got %q", taskID, expectedTitle, task.Title)
		}
		if task.Quantity != 1 {
			t.Errorf("task %s: expected Quantity 1, got %d", taskID, task.Quantity)
		}
	}

	// Total child tasks under root: 1 (cut) + 6 (sand) = 7.
	childCount := 0
	for _, task := range taskRepo.tasks {
		if task.ParentID != nil && *task.ParentID == rootTask.ID {
			childCount++
		}
	}
	if childCount != 7 {
		t.Errorf("expected 7 child tasks, got %d", childCount)
	}
}

func TestPourRecipe_BatchAware_StepToTaskIDsMap(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(batchTOML())

	vars := map[string]string{"product": "Chair", "order_qty": "6"}
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	// Verify dependency edges: each sand task depends on the cut task.
	// sand has 6 tasks, cut has 1 → 6 dependency edges.
	depRepo := svc.taskDepRepo.(*mockTaskDepRepo)
	if len(depRepo.deps) != 6 {
		t.Fatalf("expected 6 dependency edges, got %d", len(depRepo.deps))
	}

	cutTaskID := rootTask.ID + ".1"
	for _, dep := range depRepo.deps {
		if dep.ToTaskID != cutTaskID {
			t.Errorf("expected all deps to point to cut task %q, got ToTaskID %q", cutTaskID, dep.ToTaskID)
		}
	}
}

func TestPourRecipe_BatchAware_UnevenDivisionError(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(batchTOML())

	// order_qty=7 is not evenly divisible by batch_size=6 (for cut step).
	vars := map[string]string{"product": "Chair", "order_qty": "7"}
	_, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err == nil {
		t.Fatal("expected error for uneven division, got nil")
	}
}

func TestPourRecipe_BatchAware_NoOrderQty(t *testing.T) {
	// When order_qty is not provided, it defaults to the recipe-level batch_size (1).
	// This means ticket_count = 1/1 = 1 for each step → no per-piece expansion.
	svc, recipeID, spaceID, userID := setupRecipeService(simpleTOML())

	vars := map[string]string{"product": "Chair"}
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	// Should still create exactly 2 child tasks (cut, assemble) — no batch expansion.
	taskRepo := svc.taskRepo.(*mockTaskRepo)
	childCount := 0
	for _, task := range taskRepo.tasks {
		if task.ParentID != nil && *task.ParentID == rootTask.ID {
			childCount++
		}
	}
	if childCount != 2 {
		t.Errorf("expected 2 child tasks, got %d", childCount)
	}

	// All tasks should have Quantity = 1 (default).
	for _, task := range taskRepo.tasks {
		if task.ParentID != nil && *task.ParentID == rootTask.ID {
			if task.Quantity != 1 {
				t.Errorf("task %s: expected Quantity 1, got %d", task.ID, task.Quantity)
			}
		}
	}
}

// allBatchTOML returns a recipe where all steps have batch_size = order_qty,
// so every step produces exactly 1 ticket.
func allBatchTOML() string {
	return `
formula = "all-batch"
description = "All batch steps"
version = 1
type = "workflow"

[vars.order_qty]
required = true

[[steps]]
id = "prep"
title = "Prep materials"
type = "task"
batch_size = 4

[[steps]]
id = "assemble"
title = "Assemble"
type = "task"
batch_size = 4
depends_on = ["prep"]
`
}

func TestPourRecipe_AllBatchSteps(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(allBatchTOML())

	vars := map[string]string{"order_qty": "4"}
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	taskRepo := svc.taskRepo.(*mockTaskRepo)

	// Both steps have batch_size=4, order_qty=4 → 1 ticket each.
	childCount := 0
	for _, task := range taskRepo.tasks {
		if task.ParentID != nil && *task.ParentID == rootTask.ID {
			childCount++
			if task.Quantity != 4 {
				t.Errorf("task %s: expected Quantity 4, got %d", task.ID, task.Quantity)
			}
		}
	}
	if childCount != 2 {
		t.Errorf("expected 2 child tasks, got %d", childCount)
	}

	// 1 dependency edge: assemble → prep.
	depRepo := svc.taskDepRepo.(*mockTaskDepRepo)
	if len(depRepo.deps) != 1 {
		t.Errorf("expected 1 dependency edge, got %d", len(depRepo.deps))
	}
}

// perPieceChainTOML returns a recipe where two per-piece steps are chained.
func perPieceChainTOML() string {
	return `
formula = "per-piece-chain"
description = "Per-piece chain"
version = 1
type = "workflow"

[vars.order_qty]
required = true

[[steps]]
id = "sand"
title = "Sand piece {{n}}"
type = "task"
batch_size = 1

[[steps]]
id = "finish"
title = "Finish piece {{n}}"
type = "task"
batch_size = 1
depends_on = ["sand"]
`
}

func TestPourRecipe_PerPieceChain_OneToOneDependencies(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(perPieceChainTOML())

	vars := map[string]string{"order_qty": "3"}
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	taskRepo := svc.taskRepo.(*mockTaskRepo)

	// sand: 3 tasks (.1, .2, .3), finish: 3 tasks (.4, .5, .6).
	childCount := 0
	for _, task := range taskRepo.tasks {
		if task.ParentID != nil && *task.ParentID == rootTask.ID {
			childCount++
		}
	}
	if childCount != 6 {
		t.Errorf("expected 6 child tasks, got %d", childCount)
	}

	// 1:1 wiring: same ticket count (3 sand, 3 finish) → positional.
	// finish[0] depends on sand[0], finish[1] on sand[1], finish[2] on sand[2].
	// = 3 edges total.
	depRepo := svc.taskDepRepo.(*mockTaskDepRepo)
	if len(depRepo.deps) != 3 {
		t.Errorf("expected 3 dependency edges (1:1 positional), got %d", len(depRepo.deps))
	}

	// Verify each finish task depends on the corresponding sand task.
	for n := 1; n <= 3; n++ {
		sandID := fmt.Sprintf("%s.%d", rootTask.ID, n)
		finishID := fmt.Sprintf("%s.%d", rootTask.ID, n+3)

		found := false
		for _, dep := range depRepo.deps {
			if dep.FromTaskID == finishID && dep.ToTaskID == sandID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected dependency %s → %s (finish %d → sand %d)", finishID, sandID, n, n)
		}
	}

	// Verify titles use {{n}} substitution.
	for n := 1; n <= 3; n++ {
		sandID := fmt.Sprintf("%s.%d", rootTask.ID, n)
		sandTask, ok := taskRepo.tasks[sandID]
		if !ok {
			t.Fatalf("expected sand task at %q", sandID)
		}
		expectedTitle := fmt.Sprintf("Sand piece %d", n)
		if sandTask.Title != expectedTitle {
			t.Errorf("task %s: expected title %q, got %q", sandID, expectedTitle, sandTask.Title)
		}

		finishID := fmt.Sprintf("%s.%d", rootTask.ID, n+3)
		finishTask, ok := taskRepo.tasks[finishID]
		if !ok {
			t.Fatalf("expected finish task at %q", finishID)
		}
		expectedTitle = fmt.Sprintf("Finish piece %d", n)
		if finishTask.Title != expectedTitle {
			t.Errorf("task %s: expected title %q, got %q", finishID, expectedTitle, finishTask.Title)
		}
	}
}

func TestPourRecipe_BatchAware_InvalidOrderQty(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(batchTOML())

	// Non-integer order_qty.
	vars := map[string]string{"product": "Chair", "order_qty": "abc"}
	_, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err == nil {
		t.Fatal("expected error for non-integer order_qty, got nil")
	}
}

func TestPourRecipe_BatchAware_ZeroOrderQty(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(batchTOML())

	vars := map[string]string{"product": "Chair", "order_qty": "0"}
	_, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err == nil {
		t.Fatal("expected error for zero order_qty, got nil")
	}
}

func TestPourRecipe_BatchAware_RootJobAndMilestoneUnchanged(t *testing.T) {
	// Milestones and the root job should be unaffected by batch logic.
	toml := `
formula = "milestone-test"
description = "Build with milestone"
version = 1
type = "workflow"

[vars.order_qty]
default = "6"

[[steps]]
id = "cut"
title = "Cut parts"
type = "task"
batch_size = 6

[[steps]]
id = "done"
title = "All done"
type = "milestone"
batch_size = 6
depends_on = ["cut"]
`
	svc, recipeID, spaceID, userID := setupRecipeService(toml)

	vars := map[string]string{"order_qty": "6"}
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	// Root job should be a job type.
	if rootTask.Type != models.TaskTypeJob {
		t.Errorf("expected root type %q, got %q", models.TaskTypeJob, rootTask.Type)
	}

	// Milestone should be created as a milestone type.
	taskRepo := svc.taskRepo.(*mockTaskRepo)
	milestoneID := rootTask.ID + ".2"
	milestone, ok := taskRepo.tasks[milestoneID]
	if !ok {
		t.Fatalf("expected milestone task at %q", milestoneID)
	}
	if milestone.Type != models.TaskTypeMilestone {
		t.Errorf("expected milestone type %q, got %q", models.TaskTypeMilestone, milestone.Type)
	}

	// Both are batch steps (batch_size=6, order_qty=6 → 1 ticket each).
	childCount := 0
	for _, task := range taskRepo.tasks {
		if task.ParentID != nil && *task.ParentID == rootTask.ID {
			childCount++
		}
	}
	if childCount != 2 {
		t.Errorf("expected 2 child tasks, got %d", childCount)
	}
}

// --- Dependency Wiring Pattern Tests (US-005) ---

// fanOutTOML: resaw (batch_size=6, 1 ticket) → glue (batch_size=1, 6 tickets).
// Fan-out: fewer upstream → more downstream.
func fanOutTOML() string {
	return `
formula = "fan-out-test"
description = "Fan-out wiring test"
version = 1
type = "workflow"

[vars.order_qty]
required = true

[[steps]]
id = "resaw"
title = "Resaw veneers"
type = "task"
batch_size = 6

[[steps]]
id = "glue"
title = "Glue lamination {{n}} of {{batch_count}}"
type = "task"
batch_size = 1
depends_on = ["resaw"]
`
}

func TestDependencyWiring_FanOut(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(fanOutTOML())

	vars := map[string]string{"order_qty": "6"}
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	taskRepo := svc.taskRepo.(*mockTaskRepo)
	depRepo := svc.taskDepRepo.(*mockTaskDepRepo)

	// resaw: 1 ticket (.1), glue: 6 tickets (.2 through .7).
	childCount := 0
	for _, task := range taskRepo.tasks {
		if task.ParentID != nil && *task.ParentID == rootTask.ID {
			childCount++
		}
	}
	if childCount != 7 {
		t.Errorf("expected 7 child tasks (1 resaw + 6 glue), got %d", childCount)
	}

	// Fan-out: 1 upstream, 6 downstream → 6 edges (each glue depends on the 1 resaw).
	if len(depRepo.deps) != 6 {
		t.Errorf("expected 6 dependency edges (fan-out), got %d", len(depRepo.deps))
	}

	resawTaskID := rootTask.ID + ".1"
	for _, dep := range depRepo.deps {
		if dep.ToTaskID != resawTaskID {
			t.Errorf("expected all deps to point to resaw task %q, got ToTaskID %q", resawTaskID, dep.ToTaskID)
		}
	}
}

// fanInTOML: install-seat (batch_size=1, 6 tickets) → done (batch_size=6, 1 ticket).
// Fan-in: more upstream → fewer downstream.
func fanInTOML() string {
	return `
formula = "fan-in-test"
description = "Fan-in wiring test"
version = 1
type = "workflow"

[vars.order_qty]
required = true

[[steps]]
id = "install-seat"
title = "Install seat {{n}} of {{batch_count}}"
type = "task"
batch_size = 1

[[steps]]
id = "done"
title = "All seats installed"
type = "milestone"
batch_size = 6
depends_on = ["install-seat"]
`
}

func TestDependencyWiring_FanIn(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(fanInTOML())

	vars := map[string]string{"order_qty": "6"}
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	taskRepo := svc.taskRepo.(*mockTaskRepo)
	depRepo := svc.taskDepRepo.(*mockTaskDepRepo)

	// install-seat: 6 tickets (.1-.6), done: 1 ticket (.7).
	childCount := 0
	for _, task := range taskRepo.tasks {
		if task.ParentID != nil && *task.ParentID == rootTask.ID {
			childCount++
		}
	}
	if childCount != 7 {
		t.Errorf("expected 7 child tasks (6 install + 1 done), got %d", childCount)
	}

	// Fan-in: 6 upstream, 1 downstream → 6 edges (done depends on all 6 install tasks).
	if len(depRepo.deps) != 6 {
		t.Errorf("expected 6 dependency edges (fan-in), got %d", len(depRepo.deps))
	}

	doneTaskID := rootTask.ID + ".7"
	for _, dep := range depRepo.deps {
		if dep.FromTaskID != doneTaskID {
			t.Errorf("expected all deps from done task %q, got FromTaskID %q", doneTaskID, dep.FromTaskID)
		}
	}

	// Verify all 6 install tasks are referenced as upstream deps.
	upstreamIDs := make(map[string]bool)
	for _, dep := range depRepo.deps {
		upstreamIDs[dep.ToTaskID] = true
	}
	for n := 1; n <= 6; n++ {
		installID := fmt.Sprintf("%s.%d", rootTask.ID, n)
		if !upstreamIDs[installID] {
			t.Errorf("expected install task %q in upstream deps", installID)
		}
	}
}

// oneToOneTOML: sand (batch_size=2, 3 tickets) → finish (batch_size=2, 3 tickets).
// 1:1 wiring: same ticket count.
func oneToOneTOML() string {
	return `
formula = "one-to-one-test"
description = "1:1 wiring test"
version = 1
type = "workflow"

[vars.order_qty]
required = true

[[steps]]
id = "sand"
title = "Sand batch {{n}}"
type = "task"
batch_size = 2

[[steps]]
id = "finish"
title = "Finish batch {{n}}"
type = "task"
batch_size = 2
depends_on = ["sand"]
`
}

func TestDependencyWiring_OneToOne(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(oneToOneTOML())

	vars := map[string]string{"order_qty": "6"}
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	taskRepo := svc.taskRepo.(*mockTaskRepo)
	depRepo := svc.taskDepRepo.(*mockTaskDepRepo)

	// sand: 3 tickets (.1, .2, .3), finish: 3 tickets (.4, .5, .6).
	childCount := 0
	for _, task := range taskRepo.tasks {
		if task.ParentID != nil && *task.ParentID == rootTask.ID {
			childCount++
		}
	}
	if childCount != 6 {
		t.Errorf("expected 6 child tasks (3 sand + 3 finish), got %d", childCount)
	}

	// 1:1: same ticket count → 3 positional edges.
	if len(depRepo.deps) != 3 {
		t.Fatalf("expected 3 dependency edges (1:1 positional), got %d", len(depRepo.deps))
	}

	// Verify positional wiring: finish[i] depends on sand[i].
	for n := 0; n < 3; n++ {
		sandID := fmt.Sprintf("%s.%d", rootTask.ID, n+1)
		finishID := fmt.Sprintf("%s.%d", rootTask.ID, n+4)

		found := false
		for _, dep := range depRepo.deps {
			if dep.FromTaskID == finishID && dep.ToTaskID == sandID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected positional dependency %s → %s (finish %d → sand %d)", finishID, sandID, n+1, n+1)
		}
	}
}

// mixedConvergenceTOML: glue-up depends on dry-fit (1 ticket, fan-out) AND
// shape-back (6 tickets, 1:1). Tests mixed-dependency convergence.
func mixedConvergenceTOML() string {
	return `
formula = "mixed-convergence-test"
description = "Mixed convergence wiring test"
version = 1
type = "workflow"

[vars.order_qty]
required = true

[[steps]]
id = "dry-fit"
title = "Dry fit assembly"
type = "task"
batch_size = 6

[[steps]]
id = "shape-back"
title = "Shape back {{n}} of {{batch_count}}"
type = "task"
batch_size = 1

[[steps]]
id = "glue-up"
title = "Glue up piece {{n}} of {{batch_count}}"
type = "task"
batch_size = 1
depends_on = ["dry-fit", "shape-back"]
`
}

func TestDependencyWiring_MixedConvergence(t *testing.T) {
	svc, recipeID, spaceID, userID := setupRecipeService(mixedConvergenceTOML())

	vars := map[string]string{"order_qty": "6"}
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	taskRepo := svc.taskRepo.(*mockTaskRepo)
	depRepo := svc.taskDepRepo.(*mockTaskDepRepo)

	// dry-fit: batch_size=6 → 1 ticket (.1)
	// shape-back: batch_size=1 → 6 tickets (.2-.7)
	// glue-up: batch_size=1 → 6 tickets (.8-.13)
	childCount := 0
	for _, task := range taskRepo.tasks {
		if task.ParentID != nil && *task.ParentID == rootTask.ID {
			childCount++
		}
	}
	if childCount != 13 {
		t.Errorf("expected 13 child tasks (1 dry-fit + 6 shape-back + 6 glue-up), got %d", childCount)
	}

	// Dependency from dry-fit (1 task) → glue-up (6 tasks): fan-out = 6 edges.
	// Dependency from shape-back (6 tasks) → glue-up (6 tasks): 1:1 = 6 edges.
	// Total: 12 edges.
	if len(depRepo.deps) != 12 {
		t.Fatalf("expected 12 dependency edges (6 fan-out + 6 one-to-one), got %d", len(depRepo.deps))
	}

	dryFitID := rootTask.ID + ".1"

	// Count deps by upstream type.
	dryFitDeps := 0
	shapeBackDeps := 0
	for _, dep := range depRepo.deps {
		if dep.ToTaskID == dryFitID {
			dryFitDeps++
		} else {
			shapeBackDeps++
		}
	}

	// Fan-out from dry-fit: 6 glue-up tasks each depend on the 1 dry-fit task.
	if dryFitDeps != 6 {
		t.Errorf("expected 6 fan-out edges from dry-fit, got %d", dryFitDeps)
	}

	// 1:1 from shape-back: 6 glue-up tasks each depend on their corresponding shape-back task.
	if shapeBackDeps != 6 {
		t.Errorf("expected 6 one-to-one edges from shape-back, got %d", shapeBackDeps)
	}

	// Verify positional shape-back wiring: glue-up[i] depends on shape-back[i].
	for n := 0; n < 6; n++ {
		shapeBackID := fmt.Sprintf("%s.%d", rootTask.ID, n+2) // .2-.7
		glueUpID := fmt.Sprintf("%s.%d", rootTask.ID, n+8)    // .8-.13

		found := false
		for _, dep := range depRepo.deps {
			if dep.FromTaskID == glueUpID && dep.ToTaskID == shapeBackID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected positional dependency %s → %s (glue-up %d → shape-back %d)", glueUpID, shapeBackID, n+1, n+1)
		}
	}

	// Verify all glue-up tasks also depend on dry-fit (fan-out).
	for n := 0; n < 6; n++ {
		glueUpID := fmt.Sprintf("%s.%d", rootTask.ID, n+8)

		found := false
		for _, dep := range depRepo.deps {
			if dep.FromTaskID == glueUpID && dep.ToTaskID == dryFitID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected fan-out dependency %s → %s (glue-up %d → dry-fit)", glueUpID, dryFitID, n+1)
		}
	}
}

// ==========================================================================
// Chair Recipe Integration Test (US-007)
// ==========================================================================
//
// Pours the actual seeds/chair.toml with batch_size=6 and verifies:
//   - Exact task counts per stream
//   - Batch size / quantity on each task
//   - Dependency edge counts and wiring patterns (fan-out, 1:1, fan-in)
//   - Ready tasks (no blockers)

func TestPourRecipe_ChairRecipe_FullWorkflow(t *testing.T) {
	// Load the real chair.toml from the seeds directory.
	chairTOML, err := os.ReadFile("../../../seeds/chair.toml")
	if err != nil {
		t.Fatalf("failed to read seeds/chair.toml: %v", err)
	}

	svc, recipeID, spaceID, userID := setupRecipeService(string(chairTOML))

	vars := map[string]string{"batch_size": "6"}
	rootTask, err := svc.PourRecipe(recipeID, spaceID, userID, vars, nil)
	if err != nil {
		t.Fatalf("PourRecipe failed: %v", err)
	}

	taskRepo := svc.taskRepo.(*mockTaskRepo)
	depRepo := svc.taskDepRepo.(*mockTaskDepRepo)

	// ── Root job ──────────────────────────────────────────────────────
	if rootTask.Type != models.TaskTypeJob {
		t.Errorf("root task type: expected %q, got %q", models.TaskTypeJob, rootTask.Type)
	}

	// ── Count child tasks (excluding root) ───────────────────────────
	childCount := 0
	for _, task := range taskRepo.tasks {
		if task.ParentID != nil && *task.ParentID == rootTask.ID {
			childCount++
		}
	}

	// Expected task counts per step (26 steps in chair.toml):
	//
	// Legs (8 steps, all batch_size=6 → 1 ticket each):
	//   leg-rough-cut, leg-joint-face, leg-thickness, leg-rough-shape,
	//   leg-shape-profile, leg-face-mortises, leg-edge-mortises, hand-shape-legs
	//                                                             = 8 tasks
	//
	// Rails (7 steps, all batch_size=6 → 1 ticket each):
	//   rail-rough-cut, rail-joint-face-edge, rail-thickness, rail-rip-width,
	//   rail-compound-miters, rail-tenons, hand-shape-rails
	//                                                             = 7 tasks
	//
	// Dry fit (batch_size=1 → 6 tickets):                        = 6 tasks
	//
	// Back stream:
	//   resaw-veneers (bs=6 → 1 ticket)                          = 1 task
	//   glue-lamination (bs=1 → 6 tickets)                       = 6 tasks
	//   true-up-edge (bs=1 explicit → 6 tickets)                 = 6 tasks
	//   rip-back-parallel (bs=1 explicit → 6 tickets)            = 6 tasks
	//   cut-back-miters (bs=1 explicit → 6 tickets)              = 6 tasks
	//   shape-back (bs=1 explicit → 6 tickets)                   = 6 tasks
	//                                                      subtotal = 31 tasks
	//
	// Seat stream (4 steps, all batch_size=6 → 1 ticket each):
	//   cut-seat-blank, cut-foam, cut-fabric, upholster-seat
	//                                                             = 4 tasks
	//
	// Assembly:
	//   glue-up (bs=1 → 6 tickets)                               = 6 tasks
	//   spray-finish (no explicit bs → default 6 → 1 ticket)     = 1 task
	//                                                      subtotal = 7 tasks
	//
	// Final convergence:
	//   install-seat (no explicit bs → default 6 → 1 ticket)     = 1 task
	//   done (no explicit bs → default 6 → 1 ticket)             = 1 task
	//                                                      subtotal = 2 tasks
	//
	// TOTAL child tasks: 8 + 7 + 6 + 31 + 4 + 7 + 2 = 65
	expectedChildCount := 65
	if childCount != expectedChildCount {
		t.Errorf("expected %d child tasks, got %d", expectedChildCount, childCount)
	}

	// ── Build step→taskIDs map from the created tasks ────────────────
	// We reconstruct this by examining task IDs and their creation order.
	// Tasks are created sequentially: .1, .2, .3, ... under rootTask.ID.

	// Collect all child task IDs sorted by their sequence number.
	type childInfo struct {
		id       string
		seq      int // sequence number extracted from ID
		title    string
		quantity int
		taskType models.TaskType
	}
	var children []childInfo
	for _, task := range taskRepo.tasks {
		if task.ParentID != nil && *task.ParentID == rootTask.ID {
			// Extract sequence from ID: "nori-xxx.N" → N
			path := extractPath(task.ID, rootTask.ID)
			var seq int
			fmt.Sscanf(path, "%d", &seq)
			children = append(children, childInfo{
				id:       task.ID,
				seq:      seq,
				title:    task.Title,
				quantity: task.Quantity,
				taskType: task.Type,
			})
		}
	}
	sort.Slice(children, func(i, j int) bool { return children[i].seq < children[j].seq })

	if len(children) != expectedChildCount {
		t.Fatalf("sorted children count mismatch: expected %d, got %d", expectedChildCount, len(children))
	}

	// ── Verify task sequence and quantities ──────────────────────────
	// Map out the expected sequence of tasks by step ID:
	//
	// .1  leg-rough-cut          qty=6
	// .2  leg-joint-face         qty=6
	// .3  leg-thickness          qty=6
	// .4  leg-rough-shape        qty=6
	// .5  leg-shape-profile      qty=6
	// .6  leg-face-mortises      qty=6
	// .7  leg-edge-mortises      qty=6
	// .8  hand-shape-legs        qty=6
	// .9  rail-rough-cut         qty=6
	// .10 rail-joint-face-edge   qty=6
	// .11 rail-thickness         qty=6
	// .12 rail-rip-width         qty=6
	// .13 rail-compound-miters   qty=6
	// .14 rail-tenons            qty=6
	// .15 hand-shape-rails       qty=6
	// .16-.21 dry-fit (6×)       qty=1
	// .22 resaw-veneers          qty=6
	// .23-.28 glue-lamination    qty=1
	// .29-.34 true-up-edge       qty=1
	// .35-.40 rip-back-parallel  qty=1
	// .41-.46 cut-back-miters    qty=1
	// .47-.52 shape-back         qty=1
	// .53 cut-seat-blank         qty=6
	// .54 cut-foam               qty=6
	// .55 cut-fabric             qty=6
	// .56 upholster-seat         qty=6
	// .57-.62 glue-up            qty=1
	// .63 spray-finish           qty=6
	// .64 install-seat           qty=6
	// .65 done                   qty=6

	// Helper to get task at a 1-based sequence position.
	taskAt := func(seq int) childInfo {
		idx := seq - 1
		if idx < 0 || idx >= len(children) {
			t.Fatalf("invalid sequence %d (have %d children)", seq, len(children))
		}
		return children[idx]
	}

	// Spot-check leg stream (batch tasks, qty=6).
	legRoughCut := taskAt(1)
	if legRoughCut.quantity != 6 {
		t.Errorf("leg-rough-cut: expected qty 6, got %d", legRoughCut.quantity)
	}
	if legRoughCut.title != "Rough cut leg stock" {
		t.Errorf("leg-rough-cut: expected title %q, got %q", "Rough cut leg stock", legRoughCut.title)
	}

	handShapeLegs := taskAt(8)
	if handShapeLegs.quantity != 6 {
		t.Errorf("hand-shape-legs: expected qty 6, got %d", handShapeLegs.quantity)
	}

	// Spot-check rail stream.
	railRoughCut := taskAt(9)
	if railRoughCut.quantity != 6 {
		t.Errorf("rail-rough-cut: expected qty 6, got %d", railRoughCut.quantity)
	}

	handShapeRails := taskAt(15)
	if handShapeRails.quantity != 6 {
		t.Errorf("hand-shape-rails: expected qty 6, got %d", handShapeRails.quantity)
	}

	// Spot-check dry-fit (per-piece, qty=1).
	for i := 16; i <= 21; i++ {
		df := taskAt(i)
		if df.quantity != 1 {
			t.Errorf("dry-fit ticket %d: expected qty 1, got %d", i-15, df.quantity)
		}
	}

	// Spot-check back stream.
	resawVeneers := taskAt(22)
	if resawVeneers.quantity != 6 {
		t.Errorf("resaw-veneers: expected qty 6, got %d", resawVeneers.quantity)
	}

	// glue-lamination per-piece tickets (qty=1).
	for i := 23; i <= 28; i++ {
		gl := taskAt(i)
		if gl.quantity != 1 {
			t.Errorf("glue-lamination ticket %d: expected qty 1, got %d", i-22, gl.quantity)
		}
	}

	// Spot-check seat stream (batch tasks, qty=6).
	cutSeatBlank := taskAt(53)
	if cutSeatBlank.quantity != 6 {
		t.Errorf("cut-seat-blank: expected qty 6, got %d", cutSeatBlank.quantity)
	}
	if cutSeatBlank.title != "Cut out seat blanks" {
		t.Errorf("cut-seat-blank: expected title %q, got %q", "Cut out seat blanks", cutSeatBlank.title)
	}

	// Spot-check done milestone (batch, qty=6).
	done := taskAt(65)
	if done.quantity != 6 {
		t.Errorf("done: expected qty 6, got %d", done.quantity)
	}
	if done.title != "6 Chairs complete" {
		t.Errorf("done: expected title %q, got %q", "6 Chairs complete", done.title)
	}
	if done.taskType != models.TaskTypeMilestone {
		t.Errorf("done: expected type %q, got %q", models.TaskTypeMilestone, done.taskType)
	}

	// ── Dependency edge count ────────────────────────────────────────
	//
	// Legs chain (all 1:1 between single-ticket steps):
	//   7 single-dep chains + hand-shape-legs has 2 deps = 8 edges
	//
	// Rails chain (all 1:1 between single-ticket steps):
	//   6 edges
	//
	// Dry fit (6 tickets) ← hand-shape-legs (1) + hand-shape-rails (1):
	//   fan-out: 6×1 + 6×1 = 12 edges
	//
	// Back: glue-lamination (6) ← resaw-veneers (1): fan-out = 6
	//   true-up-edge (6) ← glue-lamination (6): 1:1 = 6
	//   rip-back-parallel (6) ← true-up-edge (6): 1:1 = 6
	//   cut-back-miters (6) ← rip-back-parallel (6): 1:1 = 6
	//   shape-back (6) ← cut-back-miters (6): 1:1 = 6
	//   subtotal = 30 edges
	//
	// Seat: cut-foam←cut-seat-blank + cut-fabric←cut-foam + upholster-seat←cut-fabric
	//   3 edges (all 1:1 between single-ticket steps)
	//
	// Assembly:
	//   glue-up (6) ← dry-fit (6): 1:1 = 6
	//   glue-up (6) ← shape-back (6): 1:1 = 6
	//   spray-finish (1) ← glue-up (6): fan-in = 6
	//   subtotal = 18 edges
	//
	// Install-seat (1) ← upholster-seat (1) + spray-finish (1):
	//   1:1 = 1 + 1:1 = 1 → 2 edges
	//
	// Done (1) ← install-seat (1): 1:1 = 1 edge
	//
	// TOTAL: 8 + 6 + 12 + 30 + 3 + 18 + 2 + 1 = 80 edges
	expectedEdgeCount := 80
	if len(depRepo.deps) != expectedEdgeCount {
		t.Errorf("expected %d dependency edges, got %d", expectedEdgeCount, len(depRepo.deps))
	}

	// ── Verify ready tasks (no blockers) ─────────────────────────────
	// Tasks with no incoming dependencies are "ready" — these are the
	// starting points for all four streams.
	//
	// Expected ready tasks:
	//   leg-rough-cut (.1), rail-rough-cut (.9), resaw-veneers (.22), cut-seat-blank (.53)

	// Build set of all task IDs that have at least one blocker.
	blockedTasks := make(map[string]bool)
	for _, dep := range depRepo.deps {
		blockedTasks[dep.FromTaskID] = true
	}

	// Find ready child tasks (not blocked, not the root).
	var readyTasks []childInfo
	for _, child := range children {
		if !blockedTasks[child.id] {
			readyTasks = append(readyTasks, child)
		}
	}

	expectedReadySeqs := []int{1, 9, 22, 53} // leg-rough-cut, rail-rough-cut, resaw-veneers, cut-seat-blank
	if len(readyTasks) != len(expectedReadySeqs) {
		var readySeqs []int
		for _, rt := range readyTasks {
			readySeqs = append(readySeqs, rt.seq)
		}
		t.Errorf("expected %d ready tasks at positions %v, got %d at positions %v",
			len(expectedReadySeqs), expectedReadySeqs, len(readyTasks), readySeqs)
	} else {
		sort.Slice(readyTasks, func(i, j int) bool { return readyTasks[i].seq < readyTasks[j].seq })
		for i, expected := range expectedReadySeqs {
			if readyTasks[i].seq != expected {
				t.Errorf("ready task %d: expected seq %d, got %d (%q)", i, expected, readyTasks[i].seq, readyTasks[i].title)
			}
		}
	}

	// ── Verify specific wiring patterns ──────────────────────────────

	// 1. Fan-out: resaw-veneers (.22, 1 ticket) → glue-lamination (.23-.28, 6 tickets)
	resawID := taskAt(22).id
	for i := 23; i <= 28; i++ {
		glueID := taskAt(i).id
		found := false
		for _, dep := range depRepo.deps {
			if dep.FromTaskID == glueID && dep.ToTaskID == resawID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("fan-out: expected glue-lamination ticket %d (%s) → resaw-veneers (%s)", i-22, glueID, resawID)
		}
	}

	// 2. 1:1 chain: glue-lamination[i] → true-up-edge[i]
	for i := 0; i < 6; i++ {
		glueID := taskAt(23 + i).id
		trueUpID := taskAt(29 + i).id
		found := false
		for _, dep := range depRepo.deps {
			if dep.FromTaskID == trueUpID && dep.ToTaskID == glueID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("1:1: expected true-up-edge[%d] (%s) → glue-lamination[%d] (%s)", i+1, trueUpID, i+1, glueID)
		}
	}

	// 3. Fan-in: spray-finish (.63, 1 ticket) ← glue-up (.57-.62, 6 tickets)
	sprayFinishID := taskAt(63).id
	for i := 57; i <= 62; i++ {
		glueUpID := taskAt(i).id
		found := false
		for _, dep := range depRepo.deps {
			if dep.FromTaskID == sprayFinishID && dep.ToTaskID == glueUpID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("fan-in: expected spray-finish (%s) → glue-up ticket %d (%s)", sprayFinishID, i-56, glueUpID)
		}
	}

	// 4. Done (.65, 1 ticket) ← install-seat (.64, 1 ticket): 1:1
	doneID := taskAt(65).id
	installSeatID := taskAt(64).id
	{
		found := false
		for _, dep := range depRepo.deps {
			if dep.FromTaskID == doneID && dep.ToTaskID == installSeatID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("1:1: expected done (%s) → install-seat (%s)", doneID, installSeatID)
		}
	}

	// 4. Mixed convergence at glue-up:
	//    glue-up[i] depends on dry-fit[i] (1:1) AND shape-back[i] (1:1)
	for i := 0; i < 6; i++ {
		glueUpID := taskAt(57 + i).id
		dryFitID := taskAt(16 + i).id
		shapeBackID := taskAt(47 + i).id

		foundDryFit := false
		foundShapeBack := false
		for _, dep := range depRepo.deps {
			if dep.FromTaskID == glueUpID && dep.ToTaskID == dryFitID {
				foundDryFit = true
			}
			if dep.FromTaskID == glueUpID && dep.ToTaskID == shapeBackID {
				foundShapeBack = true
			}
		}
		if !foundDryFit {
			t.Errorf("convergence: expected glue-up[%d] (%s) → dry-fit[%d] (%s)", i+1, glueUpID, i+1, dryFitID)
		}
		if !foundShapeBack {
			t.Errorf("convergence: expected glue-up[%d] (%s) → shape-back[%d] (%s)", i+1, glueUpID, i+1, shapeBackID)
		}
	}

	// 5. install-seat (.64, 1 ticket) depends on both upholster-seat (.56, 1 ticket, 1:1) and spray-finish (.63, 1 ticket, 1:1)
	upholsterID := taskAt(56).id // upholster-seat, 1 ticket
	{
		foundUpholster := false
		foundSpray := false
		for _, dep := range depRepo.deps {
			if dep.FromTaskID == installSeatID && dep.ToTaskID == upholsterID {
				foundUpholster = true
			}
			if dep.FromTaskID == installSeatID && dep.ToTaskID == sprayFinishID {
				foundSpray = true
			}
		}
		if !foundUpholster {
			t.Errorf("install-seat (%s): expected dep → upholster-seat (%s)", installSeatID, upholsterID)
		}
		if !foundSpray {
			t.Errorf("install-seat (%s): expected dep → spray-finish (%s)", installSeatID, sprayFinishID)
		}
	}
}
