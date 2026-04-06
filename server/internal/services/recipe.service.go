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

// generateTaskID creates a short, unique task ID prefix for a root job.
// Format: "nori-" + 8 hex chars from a UUID.
func generateTaskID() string {
	id := uuid.New()
	hex := fmt.Sprintf("%x", id[:4])
	return "nori-" + hex
}
