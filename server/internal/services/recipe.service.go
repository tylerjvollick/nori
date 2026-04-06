package services

import (
	"bytes"
	"fmt"
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
	GetByID(id uuid.UUID) (*models.Recipe, error)
	GetVersionByID(id int) (*models.RecipeVersion, error)
	CreateVersion(version *models.RecipeVersion) error
	ListVersions(recipeID uuid.UUID) ([]models.RecipeVersion, error)
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
	//    resolve loop counts → conditions → control flow → batch sizes
	steps := f.Steps

	// Resolve template expressions in loop count fields (e.g., "{{batch_size}}" → 6).
	if err := formula.ResolveLoopCounts(steps, allVars); err != nil {
		return nil, fmt.Errorf("resolving loop counts: %w", err)
	}

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
		batchSize := 1
		if step.BatchSize != nil {
			batchSize = *step.BatchSize
		}

		// Calculate ticket count.
		if orderQty%batchSize != 0 {
			return fmt.Errorf("step %q: order_qty %d is not evenly divisible by batch_size %d", step.ID, orderQty, batchSize)
		}
		ticketCount := orderQty / batchSize

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
	f, err := parser.ParseTOML([]byte(sourceVersion.Content))
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
		Content:       newContent,
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
