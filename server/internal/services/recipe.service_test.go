package services

import (
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
