package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/tylerjvollick/nori/internal/formula"
	"github.com/tylerjvollick/nori/internal/models"
)

// RecipeRepositoryInterface defines the methods needed from a recipe repository.
type RecipeRepositoryInterface interface {
	GetByID(id uuid.UUID) (*models.Recipe, error)
	GetVersionByID(id int) (*models.RecipeVersion, error)
}

// RecipeService handles recipe operations including pouring recipes into task graphs.
type RecipeService struct {
	recipeRepo  RecipeRepositoryInterface
	taskRepo    TaskRepositoryInterface
	taskDepRepo TaskDepRepositoryInterface
}

// NewRecipeService creates a new RecipeService.
func NewRecipeService(
	recipeRepo RecipeRepositoryInterface,
	taskRepo TaskRepositoryInterface,
	taskDepRepo TaskDepRepositoryInterface,
) *RecipeService {
	return &RecipeService{
		recipeRepo:  recipeRepo,
		taskRepo:    taskRepo,
		taskDepRepo: taskDepRepo,
	}
}

// PourRecipe loads a recipe version's TOML content, parses it through the formula
// engine, and creates a task graph (root job + child tasks + dependency edges).
//
// Parameters:
//   - recipeID: the recipe to pour
//   - spaceID: the space to create tasks in
//   - createdByID: the user performing the pour
//   - vars: variable overrides for formula expansion
//   - orderID: optional order to associate with the root job
//
// Returns the root job task.
func (s *RecipeService) PourRecipe(
	recipeID uuid.UUID,
	spaceID uuid.UUID,
	createdByID uuid.UUID,
	vars map[string]string,
	orderID *uuid.UUID,
) (*models.Task, error) {
	// 1. Load recipe and its current version.
	recipe, err := s.recipeRepo.GetByID(recipeID)
	if err != nil {
		return nil, fmt.Errorf("loading recipe: %w", err)
	}

	if recipe.CurrentVersionID == nil {
		return nil, fmt.Errorf("recipe %q has no published version", recipe.Name)
	}

	version, err := s.recipeRepo.GetVersionByID(*recipe.CurrentVersionID)
	if err != nil {
		return nil, fmt.Errorf("loading recipe version: %w", err)
	}

	// 2. Parse TOML content through formula engine.
	parser := formula.NewParser()
	f, err := parser.ParseTOML([]byte(version.Content))
	if err != nil {
		return nil, fmt.Errorf("parsing recipe TOML: %w", err)
	}

	if err := f.Validate(); err != nil {
		return nil, fmt.Errorf("validating formula: %w", err)
	}

	// 3. Apply variable defaults and validate.
	allVars := formula.ApplyDefaults(f, vars)
	if err := formula.ValidateVars(f, allVars); err != nil {
		return nil, fmt.Errorf("validating variables: %w", err)
	}

	// 4. Process steps through the formula pipeline:
	//    conditions → control flow → variable substitution
	steps := f.Steps

	// Filter steps by condition (compile-time step filtering based on vars).
	steps, err = formula.FilterStepsByCondition(steps, allVars)
	if err != nil {
		return nil, fmt.Errorf("filtering steps by condition: %w", err)
	}

	// Apply control flow: loops, branches, gates.
	steps, err = formula.ApplyControlFlow(steps, f.Compose)
	if err != nil {
		return nil, fmt.Errorf("applying control flow: %w", err)
	}

	// 5. Create root job task.
	now := time.Now()
	rootID := generateTaskID()
	jobTitle := formula.Substitute(f.Description, allVars)
	if jobTitle == "" {
		jobTitle = recipe.Name
	}

	rootTask := &models.Task{
		ID:              rootID,
		SpaceID:         spaceID,
		CreatedByID:     createdByID,
		Type:            models.TaskTypeJob,
		Status:          models.TaskStatusOpen,
		Title:           jobTitle,
		RecipeID:        &recipeID,
		RecipeVersionID: recipe.CurrentVersionID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if orderID != nil {
		rootTask.CustomerID = nil // orders link via metadata for now
		rootTask.Metadata = models.JSONB{"orderID": orderID.String()}
	}

	if err := s.taskRepo.Create(rootTask); err != nil {
		return nil, fmt.Errorf("creating root job task: %w", err)
	}

	// 6. Create child tasks from formula steps with hierarchical IDs.
	//    Also build a map from formula step ID → task ID for dependency wiring.
	stepToTaskID := make(map[string]string)
	if err := s.createChildTasks(rootID, spaceID, createdByID, steps, allVars, &recipeID, recipe.CurrentVersionID, now, stepToTaskID); err != nil {
		return nil, fmt.Errorf("creating child tasks: %w", err)
	}

	// 7. Create TaskDep edges from formula step dependencies.
	if err := s.createDependencyEdges(steps, stepToTaskID); err != nil {
		return nil, fmt.Errorf("creating dependency edges: %w", err)
	}

	return rootTask, nil
}

// createChildTasks recursively creates child tasks from formula steps.
// Each step gets a hierarchical ID: parentID.1, parentID.2, etc.
// The stepToTaskID map is populated with formula step ID → task ID mappings.
func (s *RecipeService) createChildTasks(
	parentID string,
	spaceID uuid.UUID,
	createdByID uuid.UUID,
	steps []*formula.Step,
	vars map[string]string,
	recipeID *uuid.UUID,
	recipeVersionID *int,
	now time.Time,
	stepToTaskID map[string]string,
) error {
	for i, step := range steps {
		childID := fmt.Sprintf("%s.%d", parentID, i+1)
		stepToTaskID[step.ID] = childID

		title := formula.Substitute(step.Title, vars)
		var description *string
		if step.Description != "" {
			desc := formula.Substitute(step.Description, vars)
			description = &desc
		}

		taskType := models.TaskTypeTask
		if step.Type == "milestone" {
			taskType = models.TaskTypeMilestone
		} else if step.Type == "gate" {
			taskType = models.TaskTypeGate
		}

		priority := 0
		if step.Priority != nil {
			priority = *step.Priority
		}

		task := &models.Task{
			ID:              childID,
			SpaceID:         spaceID,
			ParentID:        &parentID,
			CreatedByID:     createdByID,
			RecipeID:        recipeID,
			RecipeVersionID: recipeVersionID,
			Type:            taskType,
			Status:          models.TaskStatusOpen,
			Title:           title,
			Description:     description,
			Priority:        priority,
			DisplayOrder:    i + 1,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		if err := s.taskRepo.Create(task); err != nil {
			return fmt.Errorf("creating task for step %q: %w", step.ID, err)
		}

		// Recursively create children if the step has nested children.
		if len(step.Children) > 0 {
			if err := s.createChildTasks(childID, spaceID, createdByID, step.Children, vars, recipeID, recipeVersionID, now, stepToTaskID); err != nil {
				return err
			}
		}
	}

	return nil
}

// createDependencyEdges creates TaskDep edges from formula step dependencies.
// It processes both DependsOn and Needs fields (which are semantically equivalent).
func (s *RecipeService) createDependencyEdges(steps []*formula.Step, stepToTaskID map[string]string) error {
	for _, step := range steps {
		taskID, ok := stepToTaskID[step.ID]
		if !ok {
			continue
		}

		// Merge DependsOn and Needs — both express "this step depends on X".
		allDeps := make([]string, 0, len(step.DependsOn)+len(step.Needs))
		allDeps = append(allDeps, step.DependsOn...)
		allDeps = append(allDeps, step.Needs...)

		for _, depStepID := range allDeps {
			depTaskID, ok := stepToTaskID[depStepID]
			if !ok {
				// Dependency references a step that was filtered out or doesn't exist.
				// Skip silently — this can happen when conditions filter steps.
				continue
			}

			dep := &models.TaskDep{
				ID:         uuid.New(),
				FromTaskID: taskID,
				ToTaskID:   depTaskID,
				Type:       models.DepTypeBlocks,
			}

			if err := s.taskDepRepo.AddDep(dep); err != nil {
				return fmt.Errorf("adding dependency %s → %s: %w", taskID, depTaskID, err)
			}
		}

		// Recursively process children.
		if len(step.Children) > 0 {
			if err := s.createDependencyEdges(step.Children, stepToTaskID); err != nil {
				return err
			}
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Recipe Diff
// ---------------------------------------------------------------------------

// DiffChangeType classifies a difference between a job and its source recipe.
type DiffChangeType string

const (
	DiffChangeAdded    DiffChangeType = "added"
	DiffChangeModified DiffChangeType = "modified"
	DiffChangeRemoved  DiffChangeType = "removed"
)

// DiffItem represents a single difference between the job task tree and the
// source recipe's expected steps.
type DiffItem struct {
	// ChangeType is whether the step was added, modified, or removed.
	ChangeType DiffChangeType `json:"changeType"`

	// Path is the hierarchical position in the tree (e.g., "1", "1.2", "2.1.3").
	// This is the suffix after the root job ID.
	Path string `json:"path"`

	// TaskID is the actual task ID (set for added and modified items).
	TaskID string `json:"taskId,omitempty"`

	// StepID is the recipe step ID (set for modified and removed items).
	StepID string `json:"stepId,omitempty"`

	// Title is the task/step title.
	Title string `json:"title"`

	// ExpectedTitle is the recipe step's expected title (set for modified items).
	ExpectedTitle string `json:"expectedTitle,omitempty"`

	// Description is the task description (set for added and modified items).
	Description string `json:"description,omitempty"`

	// ExpectedDescription is the recipe step's expected description (set for modified items).
	ExpectedDescription string `json:"expectedDescription,omitempty"`
}

// RecipeDiff holds the structured comparison between a job's actual task tree
// and the expected steps from its source recipe version.
type RecipeDiff struct {
	// JobID is the root job task ID.
	JobID string `json:"jobId"`

	// RecipeVersionID is the recipe version that was compared against.
	RecipeVersionID int `json:"recipeVersionId"`

	// Added contains tasks present in the job but not in the recipe.
	Added []DiffItem `json:"added"`

	// Modified contains tasks whose title or description differ from the recipe.
	Modified []DiffItem `json:"modified"`

	// Removed contains steps expected by the recipe but absent from the job.
	Removed []DiffItem `json:"removed"`
}

// DiffJobToRecipe compares a job's actual task tree against the expected steps
// from a recipe version. It identifies steps that were added (present in the
// job but not the recipe), modified (title or description changed), or removed
// (expected by recipe but not present in the job).
//
// The comparison uses hierarchical position in the tree as the matching key.
// For example, child #2 of child #1 of the root has path "1.2" in both the
// recipe step list and the job task tree.
//
// Parameters:
//   - jobID: the root job task to compare
//   - recipeVersionID: the recipe version to compare against
//
// Returns a RecipeDiff with the structured comparison.
func (s *RecipeService) DiffJobToRecipe(jobID string, recipeVersionID int) (*RecipeDiff, error) {
	// 1. Load and validate the root job task.
	rootTask, err := s.taskRepo.GetByID(jobID)
	if err != nil {
		return nil, fmt.Errorf("loading job %q: %w", jobID, err)
	}
	if rootTask.Type != models.TaskTypeJob {
		return nil, fmt.Errorf("task %q is not a job (type=%s)", jobID, rootTask.Type)
	}

	// 2. Load the recipe version and parse through the formula engine.
	version, err := s.recipeRepo.GetVersionByID(recipeVersionID)
	if err != nil {
		return nil, fmt.Errorf("loading recipe version %d: %w", recipeVersionID, err)
	}

	parser := formula.NewParser()
	f, err := parser.ParseTOML([]byte(version.Content))
	if err != nil {
		return nil, fmt.Errorf("parsing recipe TOML: %w", err)
	}

	if err := f.Validate(); err != nil {
		return nil, fmt.Errorf("validating formula: %w", err)
	}

	// Use empty vars with defaults applied — we don't know the original vars
	// used during pour, so we use defaults to get the expected shape. Variable
	// substitution affects titles/descriptions, not the step structure.
	allVars := formula.ApplyDefaults(f, nil)

	steps := f.Steps
	steps, err = formula.FilterStepsByCondition(steps, allVars)
	if err != nil {
		return nil, fmt.Errorf("filtering steps by condition: %w", err)
	}

	steps, err = formula.ApplyControlFlow(steps, f.Compose)
	if err != nil {
		return nil, fmt.Errorf("applying control flow: %w", err)
	}

	// 3. Build expected step map keyed by hierarchical path.
	expectedSteps := flattenSteps(steps, "", allVars)

	// 4. Build actual task map keyed by hierarchical path (suffix after root ID).
	actualTasks, err := s.flattenJobTree(jobID)
	if err != nil {
		return nil, fmt.Errorf("loading job task tree: %w", err)
	}

	// 5. Compare expected vs actual.
	diff := &RecipeDiff{
		JobID:           jobID,
		RecipeVersionID: recipeVersionID,
		Added:           []DiffItem{},
		Modified:        []DiffItem{},
		Removed:         []DiffItem{},
	}

	// Find modified and removed: iterate expected steps.
	for path, expected := range expectedSteps {
		actual, exists := actualTasks[path]
		if !exists {
			// Step exists in recipe but not in job → removed.
			diff.Removed = append(diff.Removed, DiffItem{
				ChangeType: DiffChangeRemoved,
				Path:       path,
				StepID:     expected.stepID,
				Title:      expected.title,
			})
			continue
		}

		// Check for modifications.
		titleChanged := actual.title != expected.title
		descChanged := actual.description != expected.description

		if titleChanged || descChanged {
			item := DiffItem{
				ChangeType: DiffChangeModified,
				Path:       path,
				TaskID:     actual.taskID,
				StepID:     expected.stepID,
				Title:      actual.title,
			}
			if titleChanged {
				item.ExpectedTitle = expected.title
			}
			if descChanged {
				item.Description = actual.description
				item.ExpectedDescription = expected.description
			}
			diff.Modified = append(diff.Modified, item)
		}
	}

	// Find added: tasks in job but not in recipe.
	for path, actual := range actualTasks {
		if _, exists := expectedSteps[path]; !exists {
			diff.Added = append(diff.Added, DiffItem{
				ChangeType:  DiffChangeAdded,
				Path:        path,
				TaskID:      actual.taskID,
				Title:       actual.title,
				Description: actual.description,
			})
		}
	}

	return diff, nil
}

// flatStepInfo holds the comparable fields for a recipe step or task.
type flatStepInfo struct {
	stepID      string // formula step ID (for recipe steps) or empty
	taskID      string // task ID (for job tasks) or empty
	title       string
	description string
}

// flattenSteps recursively flattens formula steps into a map keyed by
// hierarchical position path (e.g., "1", "1.2", "2.1").
// The path represents the position in the tree using 1-based display order.
func flattenSteps(steps []*formula.Step, parentPath string, vars map[string]string) map[string]flatStepInfo {
	result := make(map[string]flatStepInfo)

	for i, step := range steps {
		path := fmt.Sprintf("%d", i+1)
		if parentPath != "" {
			path = fmt.Sprintf("%s.%d", parentPath, i+1)
		}

		title := formula.Substitute(step.Title, vars)
		description := ""
		if step.Description != "" {
			description = formula.Substitute(step.Description, vars)
		}

		result[path] = flatStepInfo{
			stepID:      step.ID,
			title:       title,
			description: description,
		}

		// Recursively flatten children.
		if len(step.Children) > 0 {
			childMap := flattenSteps(step.Children, path, vars)
			for k, v := range childMap {
				result[k] = v
			}
		}
	}

	return result
}

// flattenJobTree loads the job task tree from the database and returns a map
// keyed by hierarchical position path (suffix after root ID).
// For a task with ID "nori-abc123.1.2", the path is "1.2".
func (s *RecipeService) flattenJobTree(rootID string) (map[string]flatStepInfo, error) {
	result := make(map[string]flatStepInfo)

	if err := s.collectChildren(rootID, rootID, result); err != nil {
		return nil, err
	}

	return result, nil
}

// collectChildren recursively loads children of a task and populates the
// result map with their paths (relative to rootID).
func (s *RecipeService) collectChildren(parentID string, rootID string, result map[string]flatStepInfo) error {
	children, err := s.taskRepo.GetChildren(parentID)
	if err != nil {
		return fmt.Errorf("getting children of %q: %w", parentID, err)
	}

	for _, child := range children {
		// Extract the path suffix: remove "rootID." prefix from the child's ID.
		path := extractPath(child.ID, rootID)
		if path == "" {
			// Fallback: use the full ID if it doesn't follow the expected pattern.
			path = child.ID
		}

		desc := ""
		if child.Description != nil {
			desc = *child.Description
		}

		result[path] = flatStepInfo{
			taskID:      child.ID,
			title:       child.Title,
			description: desc,
		}

		// Recursively collect grandchildren.
		if err := s.collectChildren(child.ID, rootID, result); err != nil {
			return err
		}
	}

	return nil
}

// extractPath extracts the hierarchical path suffix from a task ID relative to
// the root ID. For example, extractPath("nori-abc.1.2", "nori-abc") returns "1.2".
func extractPath(taskID, rootID string) string {
	prefix := rootID + "."
	if len(taskID) > len(prefix) && taskID[:len(prefix)] == prefix {
		return taskID[len(prefix):]
	}
	return ""
}

// generateTaskID creates a short, unique task ID prefix for a root job.
// Format: "nori-" + 8 hex chars from a UUID.
func generateTaskID() string {
	id := uuid.New()
	hex := fmt.Sprintf("%x", id[:4])
	return "nori-" + hex
}
