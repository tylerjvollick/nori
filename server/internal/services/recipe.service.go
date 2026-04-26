package services

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tylerjvollick/nori/internal/formula"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
)

// RecipeRepositoryInterface defines the methods needed from a recipe repository.
type RecipeRepositoryInterface interface {
	Create(recipe *models.Recipe) error
	Update(recipe *models.Recipe) error
	GetByID(id uuid.UUID) (*models.Recipe, error)
	GetVersionByID(id int) (*models.RecipeVersion, error)
	GetVersionWithTaskTree(id int) (*models.RecipeVersion, []models.Task, error)
	CreateVersion(version *models.RecipeVersion) error
	UpdateVersion(version *models.RecipeVersion) error
	ListVersions(recipeID uuid.UUID) ([]models.RecipeVersion, error)
	PublishVersion(versionID int) error
}

// RecipeService handles recipe operations including pouring recipes into task graphs.
type RecipeService struct {
	db          *gorm.DB
	recipeRepo  RecipeRepositoryInterface
	taskRepo    TaskRepositoryInterface
	taskDepRepo TaskDepRepositoryInterface
}

// NewRecipeService creates a new RecipeService.
// The db parameter is used to create transactions for operations like PourRecipe
// that span multiple repositories. Pass nil in tests that use mock repositories.
func NewRecipeService(
	db *gorm.DB,
	recipeRepo RecipeRepositoryInterface,
	taskRepo TaskRepositoryInterface,
	taskDepRepo TaskDepRepositoryInterface,
) *RecipeService {
	return &RecipeService{
		db:          db,
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
		return nil, fmt.Errorf("recipe %q has no published version", recipe.Slug)
	}

	version, err := s.recipeRepo.GetVersionByID(*recipe.CurrentVersionID)
	if err != nil {
		return nil, fmt.Errorf("loading recipe version: %w", err)
	}

	// 2. Parse TOML content through formula engine.
	parser := formula.NewParser()
	f, err := parser.ParseTOML([]byte(*version.Content))
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
	//    conditions → control flow → batch sizes
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

	// Resolve batch sizes on all steps (explicit → inherited → default).
	// The recipe-level default comes from vars["batch_size"] or falls back to 1.
	defaultBatchSize := 1
	if bsStr, ok := allVars["batch_size"]; ok {
		if bs, err := strconv.Atoi(bsStr); err == nil && bs > 0 {
			defaultBatchSize = bs
		}
	}
	formula.ResolveBatchSizes(steps, defaultBatchSize)

	// Parse order_qty from vars. If not provided, default to the recipe-level
	// batch_size (meaning every step gets exactly 1 ticket).
	orderQty := defaultBatchSize
	if oqStr, ok := allVars["order_qty"]; ok {
		oq, err := strconv.Atoi(oqStr)
		if err != nil {
			return nil, fmt.Errorf("order_qty %q is not a valid integer", oqStr)
		}
		if oq <= 0 {
			return nil, fmt.Errorf("order_qty must be > 0, got %d", oq)
		}
		orderQty = oq
	}

	// 5. Build the root task (not yet persisted).
	now := time.Now()
	rootID := generateTaskID()
	jobTitle := formula.Substitute(f.Description, allVars)
	if jobTitle == "" {
		jobTitle = recipe.Slug
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

	// 6. Wrap all database writes in a transaction.
	//    If any task or dependency creation fails, the entire pour is rolled back.
	pourFn := func(taskRepo TaskRepositoryInterface, taskDepRepo TaskDepRepositoryInterface) error {
		if err := taskRepo.Create(rootTask); err != nil {
			return fmt.Errorf("creating root job task: %w", err)
		}

		// Create child tasks from formula steps with hierarchical IDs.
		// stepToTaskIDs maps formula step ID → list of created task IDs.
		// Per-piece steps create multiple tasks; batch steps create one.
		stepToTaskIDs := make(map[string][]string)
		if err := createChildTasks(taskRepo, rootID, spaceID, createdByID, steps, allVars, &recipeID, recipe.CurrentVersionID, now, stepToTaskIDs, orderQty); err != nil {
			return fmt.Errorf("creating child tasks: %w", err)
		}

		// Create TaskDep edges from formula step dependencies.
		if err := createDependencyEdges(taskDepRepo, steps, stepToTaskIDs); err != nil {
			return fmt.Errorf("creating dependency edges: %w", err)
		}

		return nil
	}

	if s.db != nil {
		// Production path: wrap writes in a database transaction.
		err = s.db.Transaction(func(tx *gorm.DB) error {
			txTaskRepo := repositories.NewTaskRepository(tx)
			txTaskDepRepo := repositories.NewTaskDepRepository(tx)
			return pourFn(txTaskRepo, txTaskDepRepo)
		})
	} else {
		// Test path: no DB available, use injected mock repos directly.
		err = pourFn(s.taskRepo, s.taskDepRepo)
	}

	if err != nil {
		return nil, err
	}

	return rootTask, nil
}

// createChildTasks recursively creates child tasks from formula steps.
// Each step gets a hierarchical ID: parentID.1, parentID.2, etc.
//
// Batch-aware: for each step, ticket count = orderQty / step.BatchSize.
//   - ticket_count == 1 (batch step): creates 1 task with Quantity = batch_size.
//   - ticket_count > 1 (per-piece step): creates N tasks, each with Quantity = batch_size.
//     Titles support {{n}} (1-based piece number) and {{batch_count}} (total tickets).
//
// The stepToTaskIDs map is populated with formula step ID → []taskID.
func createChildTasks(
	taskRepo TaskRepositoryInterface,
	parentID string,
	spaceID uuid.UUID,
	createdByID uuid.UUID,
	steps []*formula.Step,
	vars map[string]string,
	recipeID *uuid.UUID,
	recipeVersionID *int,
	now time.Time,
	stepToTaskIDs map[string][]string,
	orderQty int,
) error {
	// childSeq tracks the next child sequence number under parentID.
	// This auto-increments as we create tasks, even when a per-piece step
	// creates multiple tasks (each gets its own sequence number).
	childSeq := 1

	for _, step := range steps {
		// nil BatchSize means "inherit orderQty" — produce 1 task with Quantity=orderQty.
		// Explicit BatchSize means per-piece expansion: ticketCount = orderQty / batchSize.
		batchSize := orderQty
		if step.BatchSize != nil {
			batchSize = *step.BatchSize
		}

		// Calculate ticket count.
		if orderQty%batchSize != 0 {
			return fmt.Errorf("step %q: order_qty %d is not evenly divisible by batch_size %d", step.ID, orderQty, batchSize)
		}
		ticketCount := orderQty / batchSize

		taskType := models.TaskTypeTask

		priority := 0
		if step.Priority != nil {
			priority = *step.Priority
		}

		var description *string
		if step.Description != "" {
			desc := formula.Substitute(step.Description, vars)
			description = &desc
		}

		var taskIDs []string

		for n := 1; n <= ticketCount; n++ {
			childID := fmt.Sprintf("%s.%d", parentID, childSeq)
			childSeq++

			// Build title with {{n}} and {{batch_count}} substitution.
			title := formula.Substitute(step.Title, vars)
			if ticketCount > 1 {
				title = strings.ReplaceAll(title, "{{n}}", strconv.Itoa(n))
				title = strings.ReplaceAll(title, "{{batch_count}}", strconv.Itoa(ticketCount))
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
				Quantity:        batchSize,
				BatchSize:       step.BatchSize,
				Priority:        priority,
				DisplayOrder:    childSeq - 1, // use the sequence number we just used
				CreatedAt:       now,
				UpdatedAt:       now,
			}

			if err := taskRepo.Create(task); err != nil {
				return fmt.Errorf("creating task for step %q (ticket %d/%d): %w", step.ID, n, ticketCount, err)
			}

			taskIDs = append(taskIDs, childID)
		}

		stepToTaskIDs[step.ID] = taskIDs

		// Recursively create children if the step has nested children.
		// Children are created under the first task of this step.
		if len(step.Children) > 0 {
			if err := createChildTasks(taskRepo, taskIDs[0], spaceID, createdByID, step.Children, vars, recipeID, recipeVersionID, now, stepToTaskIDs, orderQty); err != nil {
				return err
			}
		}
	}

	return nil
}

// createDependencyEdges creates TaskDep edges from formula step dependencies.
// It processes both DependsOn and Needs fields (which are semantically equivalent).
//
// Dependency wiring is batch-aware and uses three patterns based on the
// ticket counts of the upstream (dependency) and downstream (dependent) steps:
//
//   - 1:1 (same ticket count): Wire positionally — downstream task[i] depends
//     on upstream task[i]. Example: sand piece 3 depends on cut piece 3.
//
//   - Fan-out (fewer upstream → more downstream): Each downstream task depends
//     on ALL upstream tasks. Example: 1 resaw ticket → 6 glue tickets: all 6
//     depend on the 1.
//
//   - Fan-in (more upstream → fewer downstream): Each downstream task depends
//     on ALL upstream tasks. Example: 6 install tickets → 1 done ticket: done
//     depends on all 6.
//
// When a step has multiple dependencies with different ticket counts, each
// dependency is wired independently using the appropriate pattern.
func createDependencyEdges(taskDepRepo TaskDepRepositoryInterface, steps []*formula.Step, stepToTaskIDs map[string][]string) error {
	for _, step := range steps {
		taskIDs, ok := stepToTaskIDs[step.ID]
		if !ok {
			continue
		}

		// Merge DependsOn and Needs — both express "this step depends on X".
		allDeps := make([]string, 0, len(step.DependsOn)+len(step.Needs))
		allDeps = append(allDeps, step.DependsOn...)
		allDeps = append(allDeps, step.Needs...)

		for _, depStepID := range allDeps {
			depTaskIDs, ok := stepToTaskIDs[depStepID]
			if !ok {
				// Dependency references a step that was filtered out or doesn't exist.
				// Skip silently — this can happen when conditions filter steps.
				continue
			}

			if err := wireDependencies(taskDepRepo, taskIDs, depTaskIDs); err != nil {
				return err
			}
		}

		// Recursively process children.
		if len(step.Children) > 0 {
			if err := createDependencyEdges(taskDepRepo, step.Children, stepToTaskIDs); err != nil {
				return err
			}
		}
	}

	return nil
}

// wireDependencies creates TaskDep edges between downstream and upstream task
// lists using the appropriate wiring pattern:
//   - 1:1 when counts match (positional wiring)
//   - Cross-product (fan-in/fan-out) when counts differ
func wireDependencies(taskDepRepo TaskDepRepositoryInterface, downstreamIDs, upstreamIDs []string) error {
	if len(downstreamIDs) == len(upstreamIDs) {
		// 1:1 positional wiring: downstream[i] depends on upstream[i].
		for i, taskID := range downstreamIDs {
			dep := &models.TaskDep{
				ID:         uuid.New(),
				FromTaskID: taskID,
				ToTaskID:   upstreamIDs[i],
				Type:       models.DepTypeBlocks,
			}
			if err := taskDepRepo.AddDep(dep); err != nil {
				return fmt.Errorf("adding dependency %s → %s: %w", taskID, upstreamIDs[i], err)
			}
		}
	} else {
		// Fan-out or fan-in: every downstream task depends on every upstream task.
		for _, taskID := range downstreamIDs {
			for _, depTaskID := range upstreamIDs {
				dep := &models.TaskDep{
					ID:         uuid.New(),
					FromTaskID: taskID,
					ToTaskID:   depTaskID,
					Type:       models.DepTypeBlocks,
				}
				if err := taskDepRepo.AddDep(dep); err != nil {
					return fmt.Errorf("adding dependency %s → %s: %w", taskID, depTaskID, err)
				}
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Deep Clone Task Tree
// ---------------------------------------------------------------------------

// TaskTreeCloneOptions controls how a task tree is cloned. Fields left nil
// inherit values from the source tasks.
type TaskTreeCloneOptions struct {
	NewRootID       string     // Required: the ID for the cloned root task.
	SpaceID         *uuid.UUID // Override SpaceID on all cloned tasks.
	CreatedByID     *uuid.UUID // Override CreatedByID on all cloned tasks.
	RecipeID        *uuid.UUID // Set RecipeID on all cloned tasks.
	RecipeVersionID *int       // Set RecipeVersionID on all cloned tasks.
	ResetStatus     bool       // If true, reset all cloned tasks to "open".
	ClearAssignedTo bool       // If true, nil out AssignedToID on all cloned tasks.
	BackfillEstimatedFromActual bool // If true, populate EstimatedTimeSecs from ActualTimeSecs when estimated is nil.
}

// DeepCloneTaskTree clones an entire task tree (root + descendants) and all
// internal dependency edges. It generates new hierarchical IDs rooted at
// opts.NewRootID while preserving the tree structure.
//
// This is the shared utility used by:
//  1. Publishing a recipe version (clone draft tree → frozen snapshot)
//  2. Rolling a recipe (clone published tree → new job)
//  3. Save-as-recipe (clone job tree → recipe draft tree)
//
// The function wraps all writes in a transaction when a real DB handle is
// available (production), or uses the injected repos directly (tests).
func (s *RecipeService) DeepCloneTaskTree(
	sourceRootID string,
	opts TaskTreeCloneOptions,
) (*models.Task, error) {
	if opts.NewRootID == "" {
		return nil, fmt.Errorf("NewRootID is required")
	}

	// 1. Load all source tasks using the hierarchical ID prefix.
	sourceTasks, err := s.taskRepo.GetDescendants(sourceRootID)
	if err != nil {
		return nil, fmt.Errorf("loading source task tree: %w", err)
	}
	if len(sourceTasks) == 0 {
		return nil, fmt.Errorf("no tasks found for root %q", sourceRootID)
	}

	// 2. Build the old→new ID mapping. The root maps to NewRootID, and
	//    descendants have their prefix replaced: "oldRoot.1.2" → "newRoot.1.2".
	idMap := make(map[string]string, len(sourceTasks))
	for _, t := range sourceTasks {
		if t.ID == sourceRootID {
			idMap[t.ID] = opts.NewRootID
		} else {
			suffix := t.ID[len(sourceRootID):]
			idMap[t.ID] = opts.NewRootID + suffix
		}
	}

	// 3. Load internal dependency edges (both endpoints within this tree).
	taskIDs := make([]string, len(sourceTasks))
	for i, t := range sourceTasks {
		taskIDs[i] = t.ID
	}
	sourceDeps, err := s.taskDepRepo.GetDepsAmongTasks(taskIDs)
	if err != nil {
		return nil, fmt.Errorf("loading source dependencies: %w", err)
	}

	// 4. Clone tasks and deps within a transaction.
	now := time.Now()
	var clonedRoot *models.Task

	cloneFn := func(taskRepo TaskRepositoryInterface, taskDepRepo TaskDepRepositoryInterface) error {
		for _, src := range sourceTasks {
			clone := src // copy all fields
			clone.ID = idMap[src.ID]
			clone.CreatedAt = now
			clone.UpdatedAt = now

			// Remap ParentID.
			if src.ParentID != nil {
				newParent := idMap[*src.ParentID]
				clone.ParentID = &newParent
			}

			// Apply overrides.
			if opts.SpaceID != nil {
				clone.SpaceID = *opts.SpaceID
			}
			if opts.CreatedByID != nil {
				clone.CreatedByID = *opts.CreatedByID
			}
			if opts.RecipeID != nil {
				clone.RecipeID = opts.RecipeID
			}
			if opts.RecipeVersionID != nil {
				clone.RecipeVersionID = opts.RecipeVersionID
			}
			if opts.ResetStatus {
				clone.Status = models.TaskStatusOpen
				clone.StartedAt = nil
				clone.PausedAt = nil
				clone.CompletedAt = nil
				clone.ActualTimeSecs = 0
			}
			if opts.ClearAssignedTo {
				clone.AssignedToID = nil
			}
			if opts.BackfillEstimatedFromActual && clone.EstimatedTimeSecs == nil && src.ActualTimeSecs > 0 {
				est := src.ActualTimeSecs
				clone.EstimatedTimeSecs = &est
			}
			// Evaluate time formula when rolling into a job. For the simple
			// clone path (orderQty ≤ 1), batch_size defaults to 1. If the task
			// has an explicit BatchSize, use that; otherwise use 1.
			if opts.ResetStatus && clone.EstimatedTimeFormula != nil {
				bs := 1
				if clone.BatchSize != nil {
					bs = *clone.BatchSize
				}
				estimated, evalErr := formula.EvalTimeFormula(clone.EstimatedTimeFormula, bs)
				if evalErr != nil {
					return fmt.Errorf("task %q: %w", clone.Title, evalErr)
				}
				clone.EstimatedTimeFromRecipeSecs = estimated
				// Clear the formula on the job task — it was only needed for evaluation.
				clone.EstimatedTimeFormula = nil
			}

			// Clear GORM relation pointers to avoid accidental nested creates.
			clone.Space = nil
			clone.Parent = nil
			clone.Children = nil
			clone.Station = nil
			clone.Customer = nil
			clone.AssignedTo = nil
			clone.CreatedBy = nil
			clone.Tags = nil

			if err := taskRepo.Create(&clone); err != nil {
				return fmt.Errorf("cloning task %q → %q: %w", src.ID, clone.ID, err)
			}

			if src.ID == sourceRootID {
				saved := clone
				clonedRoot = &saved
			}
		}

		// Clone dependency edges with remapped IDs.
		for _, dep := range sourceDeps {
			newFrom, ok1 := idMap[dep.FromTaskID]
			newTo, ok2 := idMap[dep.ToTaskID]
			if !ok1 || !ok2 {
				continue // skip deps referencing tasks outside the tree
			}

			cloneDep := &models.TaskDep{
				ID:         uuid.New(),
				FromTaskID: newFrom,
				ToTaskID:   newTo,
				Type:       dep.Type,
			}
			if err := taskDepRepo.AddDep(cloneDep); err != nil {
				return fmt.Errorf("cloning dep %s→%s: %w", newFrom, newTo, err)
			}
		}

		return nil
	}

	if s.db != nil {
		err = s.db.Transaction(func(tx *gorm.DB) error {
			txTaskRepo := repositories.NewTaskRepository(tx)
			txTaskDepRepo := repositories.NewTaskDepRepository(tx)
			return cloneFn(txTaskRepo, txTaskDepRepo)
		})
	} else {
		err = cloneFn(s.taskRepo, s.taskDepRepo)
	}

	if err != nil {
		return nil, err
	}

	return clonedRoot, nil
}

// ---------------------------------------------------------------------------
// Recipe Authoring (Task-Tree Model)
// ---------------------------------------------------------------------------

// AddStepOptions configures optional fields when adding a recipe step.
type AddStepOptions struct {
	Description          *string
	StationID            *uuid.UUID
	EstimatedTimeSecs    *int
	EstimatedTimeFormula *string
	BatchSize            *int
}

// UpdateStepOptions configures which fields to update on a recipe step.
type UpdateStepOptions struct {
	Title                *string
	Description          *string
	StationID            *uuid.UUID
	EstimatedTimeSecs    *int
	EstimatedTimeFormula *string
	BatchSize            *int
}

// CreateRecipeWithTaskTree creates a new recipe with a root task (Type='recipe')
// as the draft task tree, and a draft RecipeVersion pointing to the root task.
// This is the entry point for the visual recipe editor, as opposed to the
// TOML-based CreateVersion flow.
func (s *RecipeService) CreateRecipeWithTaskTree(
	spaceID uuid.UUID,
	createdByID uuid.UUID,
	name string,
	description *string,
	categoryID *uuid.UUID,
) (*models.Recipe, *models.RecipeVersion, error) {
	if name == "" {
		return nil, nil, fmt.Errorf("recipe name is required")
	}

	recipeID := uuid.New()
	rootTaskID := uuid.New().String()
	now := time.Now()

	recipe := &models.Recipe{
		ID:          recipeID,
		SpaceID:     spaceID,
		Slug:        serviceSlugify(name),
		CategoryID:  categoryID,
		CreatedByID: createdByID,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	rootTask := &models.Task{
		ID:          rootTaskID,
		SpaceID:     spaceID,
		CreatedByID: createdByID,
		RecipeID:    &recipeID,
		Type:        models.TaskTypeRecipe,
		Status:      models.TaskStatusOpen,
		Title:       name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	version := &models.RecipeVersion{
		RecipeID:      recipeID,
		VersionNumber: 1,
		Status:        models.RecipeVersionStatusDraft,
		RootTaskID:    &rootTaskID,
		AuthorID:      createdByID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	createFn := func(recipeRepo RecipeRepositoryInterface, taskRepo TaskRepositoryInterface) error {
		if err := recipeRepo.Create(recipe); err != nil {
			return fmt.Errorf("creating recipe: %w", err)
		}
		if err := taskRepo.Create(rootTask); err != nil {
			return fmt.Errorf("creating root task: %w", err)
		}
		if err := recipeRepo.CreateVersion(version); err != nil {
			return fmt.Errorf("creating draft version: %w", err)
		}
		// Link the recipe to its initial draft version.
		recipe.CurrentVersionID = &version.ID
		if err := recipeRepo.Update(recipe); err != nil {
			return fmt.Errorf("linking recipe to draft version: %w", err)
		}
		return nil
	}

	var err error
	if s.db != nil {
		err = s.db.Transaction(func(tx *gorm.DB) error {
			txRecipeRepo := repositories.NewRecipeRepository(tx)
			txTaskRepo := repositories.NewTaskRepository(tx)
			return createFn(txRecipeRepo, txTaskRepo)
		})
	} else {
		err = createFn(s.recipeRepo, s.taskRepo)
	}

	if err != nil {
		return nil, nil, err
	}

	return recipe, version, nil
}

// AddRecipeStep adds a child task under a parent task in a recipe's draft tree.
// The parent can be the recipe root or another step.
func (s *RecipeService) AddRecipeStep(
	recipeID uuid.UUID,
	parentTaskID string,
	title string,
	opts AddStepOptions,
) (*models.Task, error) {
	if title == "" {
		return nil, fmt.Errorf("step title is required")
	}

	version, err := s.getDraftVersion(recipeID)
	if err != nil {
		return nil, err
	}

	rootTaskID := *version.RootTaskID
	if !isTaskInTree(parentTaskID, rootTaskID) {
		return nil, fmt.Errorf("parent task %q is not in recipe %s tree", parentTaskID, recipeID)
	}

	// Get existing children to determine next sequence number.
	children, err := s.taskRepo.GetChildren(parentTaskID)
	if err != nil {
		return nil, fmt.Errorf("getting children: %w", err)
	}

	parent, err := s.taskRepo.GetByID(parentTaskID)
	if err != nil {
		return nil, fmt.Errorf("parent task %q not found: %w", parentTaskID, err)
	}

	nextSeq := len(children) + 1
	childID := fmt.Sprintf("%s.%d", parentTaskID, nextSeq)
	now := time.Now()

	child := &models.Task{
		ID:                childID,
		SpaceID:           parent.SpaceID,
		ParentID:          &parentTaskID,
		CreatedByID:       parent.CreatedByID,
		RecipeID:          &recipeID,
		Type:              models.TaskTypeTask,
		Status:            models.TaskStatusOpen,
		Title:             title,
		Description:       opts.Description,
		StationID:         opts.StationID,
		EstimatedTimeSecs:    opts.EstimatedTimeSecs,
		EstimatedTimeFormula: opts.EstimatedTimeFormula,
		BatchSize:            opts.BatchSize,
		DisplayOrder:         nextSeq,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.taskRepo.Create(child); err != nil {
		return nil, fmt.Errorf("creating step: %w", err)
	}

	return child, nil
}

// RemoveRecipeStep removes a step and all its descendants from a recipe's
// draft tree, including any dependency edges that reference removed tasks.
func (s *RecipeService) RemoveRecipeStep(recipeID uuid.UUID, taskID string) error {
	version, err := s.getDraftVersion(recipeID)
	if err != nil {
		return err
	}

	rootTaskID := *version.RootTaskID
	if taskID == rootTaskID {
		return fmt.Errorf("cannot remove the root task of a recipe")
	}
	if !isTaskInTree(taskID, rootTaskID) {
		return fmt.Errorf("task %q is not in recipe %s tree", taskID, recipeID)
	}

	// Get all descendants (including the task itself).
	descendants, err := s.taskRepo.GetDescendants(taskID)
	if err != nil {
		return fmt.Errorf("getting descendants: %w", err)
	}

	// Clean up dependency edges referencing any of the removed tasks.
	for _, task := range descendants {
		deps, err := s.taskDepRepo.GetAllForTask(task.ID)
		if err != nil {
			return fmt.Errorf("getting deps for %q: %w", task.ID, err)
		}
		for _, dep := range deps {
			if err := s.taskDepRepo.RemoveDep(dep.FromTaskID, dep.ToTaskID); err != nil {
				return fmt.Errorf("removing dep %s→%s: %w", dep.FromTaskID, dep.ToTaskID, err)
			}
		}
	}

	// Delete tasks deepest-first to avoid FK violations on parent_id.
	sort.Slice(descendants, func(i, j int) bool {
		return len(descendants[i].ID) > len(descendants[j].ID)
	})
	for _, task := range descendants {
		if err := s.taskRepo.Delete(task.ID); err != nil {
			return fmt.Errorf("deleting task %q: %w", task.ID, err)
		}
	}

	return nil
}

// ReorderRecipeSteps updates the DisplayOrder of steps under a parent task.
// orderedTaskIDs must contain the IDs of all children of parentTaskID in the
// desired order.
func (s *RecipeService) ReorderRecipeSteps(recipeID uuid.UUID, parentTaskID string, orderedTaskIDs []string) error {
	version, err := s.getDraftVersion(recipeID)
	if err != nil {
		return err
	}

	rootTaskID := *version.RootTaskID
	if !isTaskInTree(parentTaskID, rootTaskID) {
		return fmt.Errorf("parent task %q is not in recipe %s tree", parentTaskID, recipeID)
	}

	for i, taskID := range orderedTaskIDs {
		if !isTaskInTree(taskID, rootTaskID) {
			return fmt.Errorf("task %q is not in recipe tree", taskID)
		}
		task, err := s.taskRepo.GetByID(taskID)
		if err != nil {
			return fmt.Errorf("task %q not found: %w", taskID, err)
		}
		if task.ParentID == nil || *task.ParentID != parentTaskID {
			return fmt.Errorf("task %q is not a child of %q", taskID, parentTaskID)
		}
		task.DisplayOrder = i + 1
		task.UpdatedAt = time.Now()
		if err := s.taskRepo.Update(task); err != nil {
			return fmt.Errorf("updating display order for %q: %w", taskID, err)
		}
	}

	return nil
}

// UpdateRecipeStep updates fields on a step in a recipe's draft tree.
func (s *RecipeService) UpdateRecipeStep(recipeID uuid.UUID, taskID string, opts UpdateStepOptions) (*models.Task, error) {
	version, err := s.getDraftVersion(recipeID)
	if err != nil {
		return nil, err
	}

	rootTaskID := *version.RootTaskID
	if !isTaskInTree(taskID, rootTaskID) {
		return nil, fmt.Errorf("task %q is not in recipe %s tree", taskID, recipeID)
	}

	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, fmt.Errorf("task %q not found: %w", taskID, err)
	}

	if opts.Title != nil {
		task.Title = *opts.Title
	}
	if opts.Description != nil {
		task.Description = opts.Description
	}
	if opts.StationID != nil {
		task.StationID = opts.StationID
	}
	if opts.EstimatedTimeSecs != nil {
		task.EstimatedTimeSecs = opts.EstimatedTimeSecs
	}
	if opts.EstimatedTimeFormula != nil {
		task.EstimatedTimeFormula = opts.EstimatedTimeFormula
	}
	if opts.BatchSize != nil {
		task.BatchSize = opts.BatchSize
	}
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.Update(task); err != nil {
		return nil, fmt.Errorf("updating step: %w", err)
	}

	return task, nil
}

// AddStepDependency adds a dependency edge between two steps in a recipe's
// draft tree. fromTaskID depends on toTaskID (toTaskID blocks fromTaskID).
func (s *RecipeService) AddStepDependency(recipeID uuid.UUID, fromTaskID string, toTaskID string) error {
	version, err := s.getDraftVersion(recipeID)
	if err != nil {
		return err
	}

	rootTaskID := *version.RootTaskID
	if !isTaskInTree(fromTaskID, rootTaskID) {
		return fmt.Errorf("task %q is not in recipe %s tree", fromTaskID, recipeID)
	}
	if !isTaskInTree(toTaskID, rootTaskID) {
		return fmt.Errorf("task %q is not in recipe %s tree", toTaskID, recipeID)
	}

	dep := &models.TaskDep{
		ID:         uuid.New(),
		FromTaskID: fromTaskID,
		ToTaskID:   toTaskID,
		Type:       models.DepTypeBlocks,
	}

	if err := s.taskDepRepo.AddDep(dep); err != nil {
		return fmt.Errorf("adding dependency: %w", err)
	}

	return nil
}

// RemoveStepDependency removes a dependency edge between two steps in a
// recipe's draft tree.
func (s *RecipeService) RemoveStepDependency(recipeID uuid.UUID, fromTaskID string, toTaskID string) error {
	version, err := s.getDraftVersion(recipeID)
	if err != nil {
		return err
	}

	rootTaskID := *version.RootTaskID
	if !isTaskInTree(fromTaskID, rootTaskID) {
		return fmt.Errorf("task %q is not in recipe %s tree", fromTaskID, recipeID)
	}
	if !isTaskInTree(toTaskID, rootTaskID) {
		return fmt.Errorf("task %q is not in recipe %s tree", toTaskID, recipeID)
	}

	if err := s.taskDepRepo.RemoveDep(fromTaskID, toTaskID); err != nil {
		return fmt.Errorf("removing dependency: %w", err)
	}

	return nil
}

// PublishVersion publishes a draft recipe version by deep-cloning the draft
// task tree into a frozen snapshot, then marking the version as published.
//
// Steps:
//  1. Find the draft version and validate it has a root task
//  2. Deep-clone the draft task tree (new IDs, same structure)
//  3. Update the version's RootTaskID to the cloned root
//  4. Publish the version (sets status=published, publishedAt, archives old version,
//     updates Recipe.CurrentVersionID)
//
// The draft task tree remains in the database for future editing. The published
// version's RootTaskID points to the frozen clone.
func (s *RecipeService) PublishVersion(recipeID uuid.UUID) (*models.RecipeVersion, error) {
	// 1. Get the draft version.
	version, err := s.getDraftVersion(recipeID)
	if err != nil {
		return nil, fmt.Errorf("getting draft version: %w", err)
	}

	// 2. Deep-clone the draft task tree.
	sourceRootID := *version.RootTaskID
	clonedRootID := uuid.New().String()

	_, err = s.DeepCloneTaskTree(sourceRootID, TaskTreeCloneOptions{
		NewRootID: clonedRootID,
	})
	if err != nil {
		return nil, fmt.Errorf("cloning draft task tree: %w", err)
	}

	// 3. Update the version's RootTaskID to the cloned root.
	version.RootTaskID = &clonedRootID
	if err := s.recipeRepo.UpdateVersion(version); err != nil {
		return nil, fmt.Errorf("updating version root task: %w", err)
	}

	// 4. Publish: set status=published, archive old version, update Recipe.CurrentVersionID.
	if err := s.recipeRepo.PublishVersion(version.ID); err != nil {
		return nil, fmt.Errorf("publishing version: %w", err)
	}

	// Re-fetch the version to get the updated fields (status, publishedAt).
	published, err := s.recipeRepo.GetVersionByID(version.ID)
	if err != nil {
		return nil, fmt.Errorf("re-fetching published version: %w", err)
	}

	return published, nil
}

// CreateDraftFromPublished creates a new draft version by cloning the current
// published version's task tree. This is the task-tree equivalent of "create new
// version" — the user gets a mutable draft copy they can edit via the task API.
func (s *RecipeService) CreateDraftFromPublished(
	recipeID uuid.UUID,
	authorID uuid.UUID,
	changeSummary *string,
) (*models.RecipeVersion, error) {
	// 1. Load the recipe to find the current published version.
	recipe, err := s.recipeRepo.GetByID(recipeID)
	if err != nil {
		return nil, fmt.Errorf("loading recipe: %w", err)
	}

	if recipe.CurrentVersionID == nil {
		return nil, fmt.Errorf("recipe %s has no published version to clone from", recipeID)
	}

	// 2. Ensure no draft already exists.
	versions, err := s.recipeRepo.ListVersions(recipeID)
	if err != nil {
		return nil, fmt.Errorf("listing versions: %w", err)
	}
	for _, v := range versions {
		if v.Status == models.RecipeVersionStatusDraft {
			return nil, fmt.Errorf("recipe %s already has a draft version (ID %d)", recipeID, v.ID)
		}
	}

	// 3. Load the published version and verify it has a root task.
	publishedVersion, err := s.recipeRepo.GetVersionByID(*recipe.CurrentVersionID)
	if err != nil {
		return nil, fmt.Errorf("loading published version: %w", err)
	}
	if publishedVersion.RootTaskID == nil {
		return nil, fmt.Errorf("published version has no root task to clone")
	}

	// 4. Clone the published task tree.
	clonedRootID := uuid.New().String()

	_, err = s.DeepCloneTaskTree(*publishedVersion.RootTaskID, TaskTreeCloneOptions{
		NewRootID: clonedRootID,
		RecipeID:  &recipeID,
	})
	if err != nil {
		return nil, fmt.Errorf("cloning published task tree: %w", err)
	}

	// 5. Determine the next version number.
	nextVersion := 1
	if len(versions) > 0 {
		nextVersion = versions[0].VersionNumber + 1
	}

	// 6. Create the new draft version.
	now := time.Now()
	draft := &models.RecipeVersion{
		RecipeID:      recipeID,
		VersionNumber: nextVersion,
		Status:        models.RecipeVersionStatusDraft,
		RootTaskID:    &clonedRootID,
		ChangeSummary: changeSummary,
		AuthorID:      authorID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.recipeRepo.CreateVersion(draft); err != nil {
		return nil, fmt.Errorf("creating draft version: %w", err)
	}

	return draft, nil
}

// getDraftVersion finds the draft version for a recipe and validates it has a root task.
func (s *RecipeService) getDraftVersion(recipeID uuid.UUID) (*models.RecipeVersion, error) {
	versions, err := s.recipeRepo.ListVersions(recipeID)
	if err != nil {
		return nil, fmt.Errorf("listing versions: %w", err)
	}
	for i := range versions {
		if versions[i].Status == models.RecipeVersionStatusDraft {
			if versions[i].RootTaskID == nil {
				return nil, fmt.Errorf("draft version has no root task")
			}
			return &versions[i], nil
		}
	}
	return nil, fmt.Errorf("recipe %s has no draft version", recipeID)
}

// isTaskInTree checks if a task ID belongs to a recipe tree rooted at rootTaskID.
func isTaskInTree(taskID string, rootTaskID string) bool {
	return taskID == rootTaskID || strings.HasPrefix(taskID, rootTaskID+".")
}

// serviceSlugify converts a name to a URL-friendly slug.
func serviceSlugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash && b.Len() > 0 {
			b.WriteRune('-')
			prevDash = true
		}
	}
	result := b.String()
	return strings.TrimRight(result, "-")
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
	f, err := parser.ParseTOML([]byte(*version.Content))
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

// ---------------------------------------------------------------------------
// Roll Recipe (clone published tree → new job)
// ---------------------------------------------------------------------------

// RollRecipeOptions configures optional fields when rolling a recipe into a job.
type RollRecipeOptions struct {
	Title      *string    // Override the job root title (defaults to recipe name).
	CustomerID *uuid.UUID // Associate the job with a customer.
	DueDate    *time.Time // Set a due date on the job root.
	OrderQty   *int       // Batch expansion: number of units to produce (default 1).
}

// RollRecipe clones a published recipe version's task tree into a new job.
//
// When OrderQty is nil or 1, this is a 1:1 structural clone via DeepCloneTaskTree.
// When OrderQty > 1, batch expansion is applied: each recipe step is expanded
// based on its BatchSize (default 1), creating ticket_count = order_qty / batch_size
// tasks per step. Dependencies are wired using batch-aware patterns (1:1, fan-out,
// fan-in).
//
// The resulting tree gets:
//   - A new root with Type='job' and a generated nori-{hex} ID
//   - All children with new hierarchical IDs
//   - Dependency edges wired (cloned or batch-expanded)
//   - RecipeID and RecipeVersionID set on all tasks for traceability
//   - Status reset to 'open' on all tasks
//
// Returns the new job root task.
func (s *RecipeService) RollRecipe(
	recipeID uuid.UUID,
	spaceID uuid.UUID,
	createdByID uuid.UUID,
	opts RollRecipeOptions,
) (*models.Task, error) {
	// 1. Load recipe and validate it has a published version.
	recipe, err := s.recipeRepo.GetByID(recipeID)
	if err != nil {
		return nil, fmt.Errorf("loading recipe: %w", err)
	}
	if recipe.CurrentVersionID == nil {
		return nil, fmt.Errorf("recipe %q has no published version", recipe.Slug)
	}

	version, err := s.recipeRepo.GetVersionByID(*recipe.CurrentVersionID)
	if err != nil {
		return nil, fmt.Errorf("loading recipe version: %w", err)
	}
	if version.RootTaskID == nil {
		return nil, fmt.Errorf("published version %d has no root task tree", version.ID)
	}

	sourceRootID := *version.RootTaskID
	newRootID := generateTaskID()
	orderQty := 1
	if opts.OrderQty != nil {
		orderQty = *opts.OrderQty
	}

	var jobRoot *models.Task

	if orderQty <= 1 {
		// Simple path: 1:1 structural clone.
		clonedRoot, err := s.DeepCloneTaskTree(sourceRootID, TaskTreeCloneOptions{
			NewRootID:       newRootID,
			SpaceID:         &spaceID,
			CreatedByID:     &createdByID,
			RecipeID:        &recipeID,
			RecipeVersionID: recipe.CurrentVersionID,
			ResetStatus:     true,
		})
		if err != nil {
			return nil, fmt.Errorf("cloning recipe task tree: %w", err)
		}
		jobRoot = clonedRoot
	} else {
		// Batch expansion path: expand steps based on order_qty / batch_size.
		root, err := s.rollWithBatchExpansion(
			sourceRootID, newRootID, spaceID, createdByID,
			&recipeID, recipe.CurrentVersionID, orderQty,
		)
		if err != nil {
			return nil, fmt.Errorf("batch expansion: %w", err)
		}
		jobRoot = root
	}

	// Update root task: remap type to 'job' and apply optional overrides.
	jobRoot.Type = models.TaskTypeJob
	if opts.Title != nil {
		jobRoot.Title = *opts.Title
	}
	if opts.CustomerID != nil {
		jobRoot.CustomerID = opts.CustomerID
	}
	if opts.DueDate != nil {
		jobRoot.DueDate = opts.DueDate
	}

	if err := s.taskRepo.Update(jobRoot); err != nil {
		return nil, fmt.Errorf("updating job root task: %w", err)
	}

	return jobRoot, nil
}

// rollWithBatchExpansion loads the published tree, creates a root job task,
// expands child tasks based on batch sizes, and wires dependencies.
func (s *RecipeService) rollWithBatchExpansion(
	sourceRootID string,
	newRootID string,
	spaceID uuid.UUID,
	createdByID uuid.UUID,
	recipeID *uuid.UUID,
	recipeVersionID *int,
	orderQty int,
) (*models.Task, error) {
	// 1. Load all source tasks.
	sourceTasks, err := s.taskRepo.GetDescendants(sourceRootID)
	if err != nil {
		return nil, fmt.Errorf("loading source task tree: %w", err)
	}
	if len(sourceTasks) == 0 {
		return nil, fmt.Errorf("no tasks found for root %q", sourceRootID)
	}

	// 2. Load source dependency edges.
	taskIDs := make([]string, len(sourceTasks))
	for i, t := range sourceTasks {
		taskIDs[i] = t.ID
	}
	sourceDeps, err := s.taskDepRepo.GetDepsAmongTasks(taskIDs)
	if err != nil {
		return nil, fmt.Errorf("loading source dependencies: %w", err)
	}

	// 3. Create root job task (clone of source root).
	now := time.Now()
	var sourceRoot models.Task
	for _, t := range sourceTasks {
		if t.ID == sourceRootID {
			sourceRoot = t
			break
		}
	}

	root := &models.Task{
		ID:              newRootID,
		SpaceID:         spaceID,
		CreatedByID:     createdByID,
		RecipeID:        recipeID,
		RecipeVersionID: recipeVersionID,
		Type:            models.TaskTypeJob,
		Status:          models.TaskStatusOpen,
		Title:           sourceRoot.Title,
		Quantity:        1,
		Priority:        sourceRoot.Priority,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.taskRepo.Create(root); err != nil {
		return nil, fmt.Errorf("creating job root: %w", err)
	}

	// 4. Expand children with batch logic.
	sourceToTaskIDs := make(map[string][]string)
	rootChildren := getChildTasks(sourceRootID, sourceTasks)
	if err := expandTaskTree(
		s.taskRepo, newRootID, rootChildren, spaceID, createdByID,
		recipeID, recipeVersionID, sourceTasks, now, sourceToTaskIDs, orderQty,
	); err != nil {
		return nil, err
	}

	// 5. Wire dependencies using source edges + expansion mapping.
	if err := wireSourceDependencies(s.taskDepRepo, sourceDeps, sourceToTaskIDs); err != nil {
		return nil, fmt.Errorf("wiring dependencies: %w", err)
	}

	return root, nil
}

// expandTaskTree recursively creates child tasks from source recipe tasks,
// expanding per-piece tasks based on order quantity and batch size.
// This is the task-tree equivalent of createChildTasks (which works with formula.Step).
//
// The sourceToTaskIDs map is populated with source task ID → []new task IDs.
func expandTaskTree(
	taskRepo TaskRepositoryInterface,
	parentID string,
	sourceChildren []models.Task,
	spaceID uuid.UUID,
	createdByID uuid.UUID,
	recipeID *uuid.UUID,
	recipeVersionID *int,
	allSourceTasks []models.Task,
	now time.Time,
	sourceToTaskIDs map[string][]string,
	orderQty int,
) error {
	childSeq := 1

	for _, src := range sourceChildren {
		// nil BatchSize means "inherit orderQty" — produce 1 task with Quantity=orderQty.
		// Explicit BatchSize means per-piece expansion: ticketCount = orderQty / batchSize.
		batchSize := orderQty
		if src.BatchSize != nil {
			batchSize = *src.BatchSize
		}

		// Calculate ticket count.
		if orderQty%batchSize != 0 {
			return fmt.Errorf("task %q: order_qty %d is not evenly divisible by batch_size %d",
				src.Title, orderQty, batchSize)
		}
		ticketCount := orderQty / batchSize

		var taskIDs []string

		for n := 1; n <= ticketCount; n++ {
			childID := fmt.Sprintf("%s.%d", parentID, childSeq)
			childSeq++

			// Build title with {{n}} and {{batch_count}} substitution.
			title := src.Title
			if ticketCount > 1 {
				title = strings.ReplaceAll(title, "{{n}}", strconv.Itoa(n))
				title = strings.ReplaceAll(title, "{{batch_count}}", strconv.Itoa(ticketCount))
			}

			// Evaluate time formula. For nil batchSize (inherit), {{batch_size}}
			// resolves to the job's orderQty. For explicit batchSize, it resolves
			// to the task's batch size.
			formulaBatchSize := orderQty
			if src.BatchSize != nil {
				formulaBatchSize = batchSize
			}
			estimatedFromRecipe, err := formula.EvalTimeFormula(src.EstimatedTimeFormula, formulaBatchSize)
			if err != nil {
				return fmt.Errorf("task %q: %w", src.Title, err)
			}

			task := &models.Task{
				ID:                          childID,
				SpaceID:                     spaceID,
				ParentID:                    &parentID,
				CreatedByID:                 createdByID,
				RecipeID:                    recipeID,
				RecipeVersionID:             recipeVersionID,
				Type:                        models.TaskTypeTask,
				Status:                      models.TaskStatusOpen,
				Title:                       title,
				Description:                 src.Description,
				Quantity:                    batchSize,
				Priority:                    src.Priority,
				StationID:                   src.StationID,
				DisplayOrder:                childSeq - 1,
				EstimatedTimeFromRecipeSecs: estimatedFromRecipe,
				BatchSize:                   src.BatchSize,
				CreatedAt:                   now,
				UpdatedAt:                   now,
			}

			if err := taskRepo.Create(task); err != nil {
				return fmt.Errorf("creating task from %q (ticket %d/%d): %w",
					src.Title, n, ticketCount, err)
			}

			taskIDs = append(taskIDs, childID)
		}

		sourceToTaskIDs[src.ID] = taskIDs

		// Recursively expand children under the first task of this step.
		srcChildren := getChildTasks(src.ID, allSourceTasks)
		if len(srcChildren) > 0 {
			if err := expandTaskTree(
				taskRepo, taskIDs[0], srcChildren, spaceID, createdByID,
				recipeID, recipeVersionID, allSourceTasks, now, sourceToTaskIDs, orderQty,
			); err != nil {
				return err
			}
		}
	}

	return nil
}

// getChildTasks returns direct children of the given parent from the task list,
// sorted by DisplayOrder.
func getChildTasks(parentID string, tasks []models.Task) []models.Task {
	var children []models.Task
	for _, t := range tasks {
		if t.ParentID != nil && *t.ParentID == parentID {
			children = append(children, t)
		}
	}
	sort.Slice(children, func(i, j int) bool {
		return children[i].DisplayOrder < children[j].DisplayOrder
	})
	return children
}

// wireSourceDependencies creates TaskDep edges based on source task dependencies,
// using the sourceToTaskIDs mapping to resolve expanded task IDs and apply
// the appropriate wiring pattern (1:1, fan-out, fan-in).
func wireSourceDependencies(
	taskDepRepo TaskDepRepositoryInterface,
	sourceDeps []models.TaskDep,
	sourceToTaskIDs map[string][]string,
) error {
	for _, dep := range sourceDeps {
		downstreamIDs, ok1 := sourceToTaskIDs[dep.FromTaskID]
		upstreamIDs, ok2 := sourceToTaskIDs[dep.ToTaskID]
		if !ok1 || !ok2 {
			continue
		}

		if err := wireDependencies(taskDepRepo, downstreamIDs, upstreamIDs); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Save As Recipe (clone job tree → new recipe)
// ---------------------------------------------------------------------------

// SaveAsRecipeOptions configures optional fields when saving a job as a recipe.
type SaveAsRecipeOptions struct {
	Description *string    // Recipe description.
	CategoryID  *uuid.UUID // Recipe category.
	BackfillEstimatedFromActual bool // If true, populate EstimatedTimeSecs from ActualTimeSecs when estimated is nil.
}

// SaveAsRecipe clones a job's task tree into a brand-new recipe. This is the
// "I just built something — save this as a template" flow.
//
// Steps:
//  1. Load and validate the job root task.
//  2. Create a new Recipe record.
//  3. Deep-clone the job tree into a new root with Type='recipe', stripping
//     runtime fields (status, assigned_to, times) and optionally back-filling
//     estimated_time_secs from actual_time_secs.
//  4. Create a draft RecipeVersion pointing to the cloned root.
//  5. Return the new Recipe.
func (s *RecipeService) SaveAsRecipe(
	jobID string,
	spaceID uuid.UUID,
	createdByID uuid.UUID,
	name string,
	opts SaveAsRecipeOptions,
) (*models.Recipe, error) {
	if name == "" {
		return nil, fmt.Errorf("recipe name is required")
	}

	// 1. Load job root and validate.
	rootTask, err := s.taskRepo.GetByID(jobID)
	if err != nil {
		return nil, fmt.Errorf("loading job %q: %w", jobID, err)
	}
	if rootTask.Type != models.TaskTypeJob {
		return nil, fmt.Errorf("task %q is not a job (type=%s)", jobID, rootTask.Type)
	}

	// 2. Prepare IDs and timestamps.
	recipeID := uuid.New()
	newRootID := uuid.New().String()
	now := time.Now()

	// 3. Create Recipe record.
	recipe := &models.Recipe{
		ID:          recipeID,
		SpaceID:     spaceID,
		Slug:        serviceSlugify(name),
		CategoryID:  opts.CategoryID,
		CreatedByID: createdByID,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// 4. Deep-clone the job tree, stripping runtime fields.
	clonedRoot, err := s.DeepCloneTaskTree(jobID, TaskTreeCloneOptions{
		NewRootID:                   newRootID,
		SpaceID:                     &spaceID,
		CreatedByID:                 &createdByID,
		RecipeID:                    &recipeID,
		ResetStatus:                 true,
		ClearAssignedTo:             true,
		BackfillEstimatedFromActual: opts.BackfillEstimatedFromActual,
	})
	if err != nil {
		return nil, fmt.Errorf("cloning job task tree: %w", err)
	}

	// 5. Update root to recipe type.
	clonedRoot.Type = models.TaskTypeRecipe
	clonedRoot.Title = name
	clonedRoot.Description = opts.Description
	clonedRoot.CustomerID = nil
	clonedRoot.DueDate = nil
	if err := s.taskRepo.Update(clonedRoot); err != nil {
		return nil, fmt.Errorf("updating cloned root to recipe: %w", err)
	}

	// 6. Create draft RecipeVersion.
	version := &models.RecipeVersion{
		RecipeID:      recipeID,
		VersionNumber: 1,
		Status:        models.RecipeVersionStatusDraft,
		RootTaskID:    &newRootID,
		AuthorID:      createdByID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// 7. Persist recipe + version.
	createFn := func(recipeRepo RecipeRepositoryInterface) error {
		if err := recipeRepo.Create(recipe); err != nil {
			return fmt.Errorf("creating recipe: %w", err)
		}
		if err := recipeRepo.CreateVersion(version); err != nil {
			return fmt.Errorf("creating draft version: %w", err)
		}
		return nil
	}

	if s.db != nil {
		err = s.db.Transaction(func(tx *gorm.DB) error {
			txRecipeRepo := repositories.NewRecipeRepository(tx)
			return createFn(txRecipeRepo)
		})
	} else {
		err = createFn(s.recipeRepo)
	}

	if err != nil {
		return nil, err
	}

	recipe.CurrentVersionID = nil // Draft, not yet published.
	return recipe, nil
}

// ---------------------------------------------------------------------------
// Recipe Promote
// ---------------------------------------------------------------------------

// PromoteJobToRecipe creates a new draft recipe version by applying live edits
// from a job back to its source recipe. It diffs the job against the source
// version, applies the diff to the source TOML, and creates a new
// RecipeVersion with status=draft.
//
// This completes the feedback loop: live edits during execution become the
// next recipe version.
//
// Parameters:
//   - jobID: the root job task whose edits to promote
//   - recipeID: the recipe to create a new version for
//   - changeSummary: human-readable description of the changes
//
// Returns the newly created RecipeVersion.
func (s *RecipeService) PromoteJobToRecipe(
	jobID string,
	recipeID uuid.UUID,
	changeSummary string,
) (*models.RecipeVersion, error) {
	// 1. Load the root job to get the source recipe version.
	rootTask, err := s.taskRepo.GetByID(jobID)
	if err != nil {
		return nil, fmt.Errorf("loading job %q: %w", jobID, err)
	}
	if rootTask.Type != models.TaskTypeJob {
		return nil, fmt.Errorf("task %q is not a job (type=%s)", jobID, rootTask.Type)
	}
	if rootTask.RecipeID == nil || *rootTask.RecipeID != recipeID {
		return nil, fmt.Errorf("job %q is not linked to recipe %s", jobID, recipeID)
	}
	if rootTask.RecipeVersionID == nil {
		return nil, fmt.Errorf("job %q has no source recipe version", jobID)
	}

	sourceVersionID := *rootTask.RecipeVersionID

	// 2. Diff the job against its source recipe version.
	diff, err := s.DiffJobToRecipe(jobID, sourceVersionID)
	if err != nil {
		return nil, fmt.Errorf("diffing job against recipe: %w", err)
	}

	// 3. Load and parse the source version's TOML.
	sourceVersion, err := s.recipeRepo.GetVersionByID(sourceVersionID)
	if err != nil {
		return nil, fmt.Errorf("loading source version: %w", err)
	}

	parser := formula.NewParser()
	f, err := parser.ParseTOML([]byte(*sourceVersion.Content))
	if err != nil {
		return nil, fmt.Errorf("parsing source TOML: %w", err)
	}

	// 4. Apply the diff to the formula steps.
	applyDiffToFormula(f, diff)

	// 5. Marshal the updated formula back to TOML.
	newContent, err := marshalFormulaToTOML(f)
	if err != nil {
		return nil, fmt.Errorf("marshaling updated TOML: %w", err)
	}

	// 6. Determine the next version number.
	versions, err := s.recipeRepo.ListVersions(recipeID)
	if err != nil {
		return nil, fmt.Errorf("listing versions: %w", err)
	}
	nextVersionNumber := 1
	for _, v := range versions {
		if v.VersionNumber >= nextVersionNumber {
			nextVersionNumber = v.VersionNumber + 1
		}
	}

	// 7. Create the new draft version.
	now := time.Now()
	newVersion := &models.RecipeVersion{
		RecipeID:      recipeID,
		VersionNumber: nextVersionNumber,
		Status:        models.RecipeVersionStatusDraft,
		Content:       &newContent,
		AuthorID:      rootTask.CreatedByID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if changeSummary != "" {
		newVersion.ChangeSummary = &changeSummary
	}

	if err := s.recipeRepo.CreateVersion(newVersion); err != nil {
		return nil, fmt.Errorf("creating new version: %w", err)
	}

	return newVersion, nil
}

// applyDiffToFormula mutates a formula's steps in place based on a RecipeDiff.
// It handles modifications (title/description changes), removals, and additions.
func applyDiffToFormula(f *formula.Formula, diff *RecipeDiff) {
	// Build a step index by ID for quick lookup.
	stepByID := make(map[string]*formula.Step)
	indexStepsByID(f.Steps, stepByID)

	// Apply modifications: update title and/or description on matching steps.
	for _, mod := range diff.Modified {
		if step, ok := stepByID[mod.StepID]; ok {
			step.Title = mod.Title
			if mod.Description != "" || mod.ExpectedDescription != "" {
				step.Description = mod.Description
			}
		}
	}

	// Apply removals: remove steps by ID.
	for _, rem := range diff.Removed {
		f.Steps = removeStepByID(f.Steps, rem.StepID)
	}

	// Apply additions: append new steps at the corresponding position.
	for _, add := range diff.Added {
		newStep := &formula.Step{
			ID:          pathToStepID(add.Path),
			Title:       add.Title,
			Description: add.Description,
			Type:        "task",
		}
		f.Steps = insertStepAtPath(f.Steps, add.Path, newStep)
	}
}

// indexStepsByID recursively indexes all steps by their ID.
func indexStepsByID(steps []*formula.Step, index map[string]*formula.Step) {
	for _, step := range steps {
		index[step.ID] = step
		if len(step.Children) > 0 {
			indexStepsByID(step.Children, index)
		}
	}
}

// removeStepByID recursively removes a step with the given ID from the slice.
func removeStepByID(steps []*formula.Step, id string) []*formula.Step {
	result := make([]*formula.Step, 0, len(steps))
	for _, step := range steps {
		if step.ID == id {
			continue
		}
		if len(step.Children) > 0 {
			step.Children = removeStepByID(step.Children, id)
		}
		result = append(result, step)
	}
	return result
}

// pathToStepID converts a hierarchical path like "3" or "1.3" to a step ID
// like "added-3" or "added-1-3" (since we don't know the original step ID
// for tasks added during execution).
func pathToStepID(path string) string {
	return "added-" + strings.ReplaceAll(path, ".", "-")
}

// insertStepAtPath inserts a step at the position indicated by a hierarchical
// path. A path like "3" inserts at top-level position 3. A path like "1.3"
// inserts as child 3 of top-level step 1.
func insertStepAtPath(steps []*formula.Step, path string, newStep *formula.Step) []*formula.Step {
	parts := strings.Split(path, ".")
	if len(parts) == 1 {
		// Top-level insertion — convert path to 0-based index.
		idx, err := strconv.Atoi(parts[0])
		if err != nil {
			return append(steps, newStep)
		}
		idx-- // path is 1-based

		if idx >= len(steps) {
			return append(steps, newStep)
		}
		// Insert at position.
		result := make([]*formula.Step, 0, len(steps)+1)
		result = append(result, steps[:idx]...)
		result = append(result, newStep)
		result = append(result, steps[idx:]...)
		return result
	}

	// Nested insertion — descend into the parent step's children.
	parentIdx, err := strconv.Atoi(parts[0])
	if err != nil || parentIdx-1 >= len(steps) {
		return append(steps, newStep)
	}
	parentIdx-- // 1-based → 0-based
	childPath := strings.Join(parts[1:], ".")
	steps[parentIdx].Children = insertStepAtPath(steps[parentIdx].Children, childPath, newStep)
	return steps
}

// tomlFormula is the serialization-friendly representation of a Formula for
// TOML output. We use explicit struct tags to control field ordering and
// naming, avoiding the complexity of marshaling the full Formula type which
// includes internal-only fields.
type tomlFormula struct {
	Formula     string                 `toml:"formula"`
	Description string                 `toml:"description,omitempty"`
	Version     int                    `toml:"version"`
	Type        formula.FormulaType    `toml:"type"`
	Vars        map[string]*tomlVarDef `toml:"vars,omitempty"`
	Steps       []*tomlStep            `toml:"steps"`
	Compose     *formula.ComposeRules  `toml:"compose,omitempty"`
}

type tomlVarDef struct {
	Description string   `toml:"description,omitempty"`
	Default     *string  `toml:"default,omitempty"`
	Required    bool     `toml:"required,omitempty"`
	Enum        []string `toml:"enum,omitempty"`
	Pattern     string   `toml:"pattern,omitempty"`
	Type        string   `toml:"type,omitempty"`
}

type tomlStep struct {
	ID          string      `toml:"id"`
	Title       string      `toml:"title"`
	Description string      `toml:"description,omitempty"`
	Type        string      `toml:"type,omitempty"`
	Priority    *int        `toml:"priority,omitempty"`
	DependsOn   []string    `toml:"depends_on,omitempty"`
	Needs       []string    `toml:"needs,omitempty"`
	Condition   string      `toml:"condition,omitempty"`
	Children    []*tomlStep `toml:"children,omitempty"`
}

// marshalFormulaToTOML converts a Formula to TOML string content.
func marshalFormulaToTOML(f *formula.Formula) (string, error) {
	tf := &tomlFormula{
		Formula:     f.Formula,
		Description: f.Description,
		Version:     f.Version,
		Type:        f.Type,
		Steps:       convertStepsToTOML(f.Steps),
		Compose:     f.Compose,
	}

	if len(f.Vars) > 0 {
		tf.Vars = make(map[string]*tomlVarDef, len(f.Vars))
		for name, v := range f.Vars {
			tf.Vars[name] = &tomlVarDef{
				Description: v.Description,
				Default:     v.Default,
				Required:    v.Required,
				Enum:        v.Enum,
				Pattern:     v.Pattern,
				Type:        v.Type,
			}
		}
	}

	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(tf); err != nil {
		return "", fmt.Errorf("encoding TOML: %w", err)
	}

	return buf.String(), nil
}

// convertStepsToTOML recursively converts formula.Step to tomlStep.
func convertStepsToTOML(steps []*formula.Step) []*tomlStep {
	result := make([]*tomlStep, len(steps))
	for i, s := range steps {
		ts := &tomlStep{
			ID:          s.ID,
			Title:       s.Title,
			Description: s.Description,
			Type:        s.Type,
			Priority:    s.Priority,
			DependsOn:   s.DependsOn,
			Needs:       s.Needs,
			Condition:   s.Condition,
		}
		if len(s.Children) > 0 {
			ts.Children = convertStepsToTOML(s.Children)
		}
		result[i] = ts
	}
	return result
}
