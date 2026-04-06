package services

import (
	"testing"

	"github.com/google/uuid"

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

	svc := NewRecipeService(recipeRepo, taskRepo, depRepo)
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

	svc := NewRecipeService(recipeRepo, taskRepo, depRepo)
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
