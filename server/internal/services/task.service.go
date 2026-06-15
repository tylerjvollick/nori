package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
)

// ErrMaxDepthExceeded is returned when a task operation would create nesting
// deeper than 2 levels (root → child). Tasks may not be nested under a task
// that already has a parent.
var ErrMaxDepthExceeded = errors.New("tasks cannot be nested more than 2 levels deep (root → child)")

// TaskRepositoryInterface defines the methods needed from a task repository.
type TaskRepositoryInterface interface {
	Create(task *models.Task) error
	GetByID(id string) (*models.Task, error)
	List(filter repositories.TaskFilter) ([]models.Task, int64, error)
	Update(task *models.Task) error
	Delete(id string) error
	GetChildren(parentID string) ([]models.Task, error)
	GetRoot(taskID string) (*models.Task, error)
	GetDescendants(idPrefix string) ([]models.Task, error)
}

// TaskDepRepositoryInterface defines the methods needed from a task dep repository.
type TaskDepRepositoryInterface interface {
	AddDep(dep *models.TaskDep) error
	RemoveDep(fromID, toID string) error
	GetBlockers(taskID string) ([]models.TaskDep, error)
	GetDependents(taskID string) ([]models.TaskDep, error)
	GetAllForTask(taskID string) ([]models.TaskDep, error)
	GetDepsAmongTasks(taskIDs []string) ([]models.TaskDep, error)
}

// TimeEventRepositoryInterface defines the methods needed to create time events.
type TimeEventRepositoryInterface interface {
	Create(event *models.TimeEvent) error
}

// TimeEntryStopperInterface allows CompleteTask to auto-stop running timers.
type TimeEntryStopperInterface interface {
	StopRunningTimer(taskID string) error
}

type TaskService struct {
	taskRepo         TaskRepositoryInterface
	taskDepRepo      TaskDepRepositoryInterface
	timeEventRepo    TimeEventRepositoryInterface
	timeEntryStopper TimeEntryStopperInterface
}

func NewTaskService(taskRepo TaskRepositoryInterface, taskDepRepo TaskDepRepositoryInterface, timeEventRepo TimeEventRepositoryInterface) *TaskService {
	return &TaskService{taskRepo: taskRepo, taskDepRepo: taskDepRepo, timeEventRepo: timeEventRepo}
}

// SetTimeEntryStopper injects the time entry stopper after construction to
// avoid circular dependencies between TaskService and TimeEntryService.
func (s *TaskService) SetTimeEntryStopper(stopper TimeEntryStopperInterface) {
	s.timeEntryStopper = stopper
}

// CreateTask creates a new task in the given space.
func (s *TaskService) CreateTask(spaceID uuid.UUID, createdByID uuid.UUID, dto *dtos.CreateTaskRequest) (*models.Task, error) {
	if dto.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	// Generate the task ID. Root tasks (jobs) get a short hierarchical prefix
	// (task-xxxx). Children derive a {parentID}.{seq} ID so descendant tree
	// queries (LIKE 'parent.%') continue to work.
	id := generateTaskID()

	// Enforce max nesting depth of 2 levels. If ParentID is set, the parent
	// must be a root task (i.e., its own ParentID must be nil).
	if dto.ParentID != nil {
		parent, err := s.taskRepo.GetByID(*dto.ParentID)
		if err != nil {
			return nil, fmt.Errorf("parent task %q not found: %w", *dto.ParentID, err)
		}
		if parent.ParentID != nil {
			return nil, ErrMaxDepthExceeded
		}
		children, err := s.taskRepo.GetChildren(*dto.ParentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get children of task %q: %w", *dto.ParentID, err)
		}
		id = fmt.Sprintf("%s.%d", *dto.ParentID, len(children)+1)
	}

	// Tasks start as open unless explicitly created in the backlog
	// (confirmed but not yet scheduled).
	status := models.TaskStatusOpen
	if dto.Status != nil {
		switch *dto.Status {
		case models.TaskStatusOpen, models.TaskStatusBacklog:
			status = *dto.Status
		default:
			return nil, fmt.Errorf("invalid initial status %q: must be %q or %q",
				*dto.Status, models.TaskStatusOpen, models.TaskStatusBacklog)
		}
	}

	task := &models.Task{
		ID:          id,
		SpaceID:     spaceID,
		CreatedByID: createdByID,
		Title:       dto.Title,
		Description: dto.Description,
		Type:        dto.Type,
		ParentID:    dto.ParentID,
		StationID:   dto.StationID,
		Status:      status,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if dto.Priority != nil {
		task.Priority = *dto.Priority
	}

	if err := s.taskRepo.Create(task); err != nil {
		return nil, err
	}

	return task, nil
}

// GetTaskByID retrieves a task by its ID.
func (s *TaskService) GetTaskByID(id string) (*models.Task, error) {
	return s.taskRepo.GetByID(id)
}

// GetTaskTree retrieves a task and all its descendants in a single query,
// then assembles them into a nested tree structure.
func (s *TaskService) GetTaskTree(id string) (*models.Task, []models.Task, error) {
	// Fetch the root task first to verify it exists.
	root, err := s.taskRepo.GetByID(id)
	if err != nil {
		return nil, nil, err
	}

	// Fetch all descendants (including root) with a single LIKE query.
	descendants, err := s.taskRepo.GetDescendants(id)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch descendants of task %q: %w", id, err)
	}

	return root, descendants, nil
}

// ListTasks returns tasks matching the given filter.
func (s *TaskService) ListTasks(filter repositories.TaskFilter) ([]models.Task, int64, error) {
	return s.taskRepo.List(filter)
}

// UpdateTask updates an existing task.
func (s *TaskService) UpdateTask(id string, dto *dtos.UpdateTaskRequest) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if dto.Title != nil {
		task.Title = *dto.Title
	}
	if dto.Description != nil {
		task.Description = dto.Description
	}
	if dto.StationID != nil {
		task.StationID = dto.StationID
	}
	if dto.Priority != nil {
		task.Priority = *dto.Priority
	}
	if dto.Status != nil {
		task.Status = *dto.Status
	}
	if dto.EstimatedTimeFormula != nil {
		task.EstimatedTimeFormula = dto.EstimatedTimeFormula
	}
	if dto.ClearBatchSize {
		task.BatchSize = nil
	} else if dto.BatchSize != nil {
		task.BatchSize = dto.BatchSize
	}
	if dto.AssignedToID != nil {
		if *dto.AssignedToID == "" {
			task.AssignedToID = nil // unassign
		} else {
			parsed, err := uuid.Parse(*dto.AssignedToID)
			if err != nil {
				return nil, fmt.Errorf("invalid assignedToId: %w", err)
			}
			task.AssignedToID = &parsed
		}
	}

	task.UpdatedAt = time.Now()

	if err := s.taskRepo.Update(task); err != nil {
		return nil, err
	}

	return task, nil
}

// DeleteTask deletes a task by its ID.
func (s *TaskService) DeleteTask(id string) error {
	return s.taskRepo.Delete(id)
}

// SetTaskStatus transitions a task to the given status.
// Valid transitions: backlog -> open/in_progress; open -> in_progress;
// any non-terminal except backlog -> done; any non-terminal -> skipped/cancelled.
// Sets StartedAt on first transition to in_progress.
func (s *TaskService) SetTaskStatus(taskID string, userID uuid.UUID, newStatus models.TaskStatus) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, err
	}

	if err := validateStatusTransition(task.Status, newStatus); err != nil {
		return nil, fmt.Errorf("task %q: %w", taskID, err)
	}

	now := time.Now()
	task.Status = newStatus
	task.UpdatedAt = now

	if newStatus == models.TaskStatusInProgress && task.StartedAt == nil {
		task.StartedAt = &now
	}
	if isTerminalStatus(newStatus) {
		task.CompletedAt = &now
	}

	if err := s.taskRepo.Update(task); err != nil {
		return nil, err
	}

	// Emit time events for status changes.
	switch newStatus {
	case models.TaskStatusInProgress:
		s.emitTimeEvent(task, userID, models.TimeEventTypeCheckIn, now)
	case models.TaskStatusDone, models.TaskStatusSkipped, models.TaskStatusCancelled:
		s.emitTimeEvent(task, userID, models.TimeEventTypeCheckOut, now)
		// Auto-complete parent if all siblings are terminal.
		if task.ParentID != nil {
			_ = s.maybeCompleteParent(*task.ParentID)
		}
	}

	return task, nil
}

// validateStatusTransition checks if transitioning from current to next is valid.
func validateStatusTransition(current, next models.TaskStatus) error {
	if isTerminalStatus(current) {
		return fmt.Errorf("cannot change status: already in terminal status %q", current)
	}
	if current == next {
		return nil // no-op is fine
	}
	switch next {
	case models.TaskStatusBacklog:
		// Can revert to backlog (unschedule) from open only.
		if current != models.TaskStatusOpen {
			return fmt.Errorf("cannot change from %q to %q", current, next)
		}
	case models.TaskStatusOpen:
		// Schedule from backlog, or revert to open from in_progress.
		if current != models.TaskStatusBacklog && current != models.TaskStatusInProgress {
			return fmt.Errorf("cannot change from %q to %q", current, next)
		}
	case models.TaskStatusInProgress:
		// Start from open, or directly from backlog (shortcut).
		if current != models.TaskStatusOpen && current != models.TaskStatusBacklog {
			return fmt.Errorf("cannot change from %q to %q", current, next)
		}
	case models.TaskStatusDone:
		// Done is allowed from any non-terminal status except backlog —
		// backlog work was never scheduled, so it cannot be completed.
		if current == models.TaskStatusBacklog {
			return fmt.Errorf("cannot change from %q to %q: schedule (open) or start (in_progress) the task first", current, next)
		}
	case models.TaskStatusSkipped, models.TaskStatusCancelled:
		// Skipped/cancelled are allowed from any non-terminal status.
	default:
		return fmt.Errorf("unknown status %q", next)
	}
	return nil
}

// StartTask is a convenience that sets status to in_progress.
// Kept for backward compatibility with CLI and existing callers.
func (s *TaskService) StartTask(taskID string, userID uuid.UUID) (*models.Task, error) {
	return s.SetTaskStatus(taskID, userID, models.TaskStatusInProgress)
}

// UnresolvedBlocker describes a blocker that is not yet in a terminal status.
type UnresolvedBlocker struct {
	ID     string
	Title  string
	Status models.TaskStatus
}

// CompleteTaskResult holds the completed task and optional navigation hint.
type CompleteTaskResult struct {
	Task               *models.Task
	NextTaskID         *string
	NextTaskTitle      *string
	UnresolvedBlockers []UnresolvedBlocker
}

// CompleteTask marks a task as done.
// The task must be in a non-terminal status. Unresolved blockers are returned
// in the result (not as an error) so the frontend can prompt cascade completion.
// If actualTimeSecs is non-nil, the task's ActualTimeSecs is overridden.
// If all sibling tasks under the same parent are now done, the parent is
// auto-completed as well.
// Returns the completed task plus the next downstream task ID (if any).
func (s *TaskService) CompleteTask(taskID string, userID uuid.UUID, actualTimeSecs *int) (*CompleteTaskResult, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, err
	}

	// Validate status: must not already be terminal.
	if isTerminalStatus(task.Status) {
		return nil, fmt.Errorf("task %q cannot be completed: already in terminal status %q", taskID, task.Status)
	}

	// Collect unresolved blockers (informational, not blocking).
	var unresolvedBlockers []UnresolvedBlocker
	deps, err := s.taskDepRepo.GetBlockers(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to check dependencies for task %q: %w", taskID, err)
	}
	for _, dep := range deps {
		if dep.Type != models.DepTypeBlocks {
			continue
		}
		blocker, err := s.taskRepo.GetByID(dep.ToTaskID)
		if err != nil {
			return nil, fmt.Errorf("failed to look up blocking task %q: %w", dep.ToTaskID, err)
		}
		if !isTerminalStatus(blocker.Status) {
			unresolvedBlockers = append(unresolvedBlockers, UnresolvedBlocker{
				ID:     blocker.ID,
				Title:  blocker.Title,
				Status: blocker.Status,
			})
		}
	}

	// Auto-stop any running timer before completing.
	if s.timeEntryStopper != nil {
		if err := s.timeEntryStopper.StopRunningTimer(taskID); err != nil {
			return nil, fmt.Errorf("stopping running timer: %w", err)
		}
	}

	// Apply time correction if provided.
	if actualTimeSecs != nil {
		task.ActualTimeSecs = *actualTimeSecs
	}

	// Mark task as done.
	now := time.Now()
	task.Status = models.TaskStatusDone
	task.CompletedAt = &now
	task.UpdatedAt = now

	if err := s.taskRepo.Update(task); err != nil {
		return nil, err
	}

	s.emitTimeEvent(task, userID, models.TimeEventTypeCheckOut, now)

	// Auto-complete parent if all siblings are done.
	if task.ParentID != nil {
		if err := s.maybeCompleteParent(*task.ParentID); err != nil {
			_ = err
		}
	}

	// Find the next downstream task (first task blocked by this one).
	result := &CompleteTaskResult{
		Task:               task,
		UnresolvedBlockers: unresolvedBlockers,
	}
	nextID, nextTitle := s.findNextTask(taskID)
	result.NextTaskID = nextID
	result.NextTaskTitle = nextTitle

	return result, nil
}

// findNextTask finds the first downstream task that is blocked by the given task.
// It returns the task ID and title, or nil if no downstream task exists.
// The "next" task is the first non-terminal task that has a "blocks" dependency
// on the completed task, sorted by priority then creation date.
func (s *TaskService) findNextTask(completedTaskID string) (*string, *string) {
	// GetDependents returns deps where to_task_id = completedTaskID.
	// In the dep model: FromTaskID is the task that depends, ToTaskID is the blocker.
	// So these are deps where completedTaskID is the blocker — the FromTaskID tasks
	// are the ones that were waiting on it.
	blockerDeps, err := s.taskDepRepo.GetDependents(completedTaskID)
	if err != nil {
		return nil, nil
	}

	// Collect candidate downstream tasks (non-terminal, with "blocks" type).
	type candidate struct {
		id       string
		title    string
		priority int
	}
	var candidates []candidate

	for _, dep := range blockerDeps {
		if dep.Type != models.DepTypeBlocks {
			continue
		}
		downstream, err := s.taskRepo.GetByID(dep.FromTaskID)
		if err != nil {
			continue
		}
		if !isTerminalStatus(downstream.Status) {
			candidates = append(candidates, candidate{
				id:       downstream.ID,
				title:    downstream.Title,
				priority: downstream.Priority,
			})
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// Pick the highest-priority (lowest number) candidate.
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.priority < best.priority {
			best = c
		}
	}

	return &best.id, &best.title
}

// maybeCompleteParent checks if all children of the given parent task are in
// a terminal status, and if so, auto-completes the parent.
func (s *TaskService) maybeCompleteParent(parentID string) error {
	parent, err := s.taskRepo.GetByID(parentID)
	if err != nil {
		return fmt.Errorf("failed to look up parent task %q: %w", parentID, err)
	}

	// Only auto-complete if parent is still in an active-ish state.
	if parent.Status == models.TaskStatusDone || parent.Status == models.TaskStatusCancelled || parent.Status == models.TaskStatusSkipped {
		return nil
	}

	children, err := s.taskRepo.GetChildren(parentID)
	if err != nil {
		return fmt.Errorf("failed to get children of task %q: %w", parentID, err)
	}

	if len(children) == 0 {
		return nil
	}

	for _, child := range children {
		if !isTerminalStatus(child.Status) {
			return nil // at least one child is not done
		}
	}

	// All children are in terminal status — auto-complete parent.
	now := time.Now()
	parent.Status = models.TaskStatusDone
	parent.CompletedAt = &now
	parent.UpdatedAt = now

	return s.taskRepo.Update(parent)
}

// emitTimeEvent creates a TimeEvent with source=system for automatic task transitions.
// Errors are logged but not propagated — the task transition itself is the critical path.
func (s *TaskService) emitTimeEvent(task *models.Task, userID uuid.UUID, eventType models.TimeEventType, ts time.Time) {
	if s.timeEventRepo == nil {
		return
	}
	event := &models.TimeEvent{
		SpaceID:   task.SpaceID,
		UserID:    userID,
		TaskID:    &task.ID,
		StationID: task.StationID,
		EventType: eventType,
		Source:    models.TimeEventSourceSystem,
		Timestamp: ts,
	}
	// Best-effort: don't fail the transition if event creation fails.
	_ = s.timeEventRepo.Create(event)
}

// isTerminalStatus returns true if the status indicates the task is finished.
func isTerminalStatus(status models.TaskStatus) bool {
	return status == models.TaskStatusDone ||
		status == models.TaskStatusSkipped ||
		status == models.TaskStatusCancelled
}

// AddChildTask creates a new child task under an existing parent task.
// It generates a hierarchical ID by appending the next sequence number to the
// parent's ID (e.g., "job-abc.1", "job-abc.2"). This supports first-time
// capture mode where operators add steps during execution.
func (s *TaskService) AddChildTask(parentID string, dto *dtos.AddChildTaskRequest, userID uuid.UUID) (*models.Task, error) {
	if dto.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	// Validate parent exists.
	parent, err := s.taskRepo.GetByID(parentID)
	if err != nil {
		return nil, fmt.Errorf("parent task %q not found: %w", parentID, err)
	}

	// Enforce max nesting depth of 2 levels. If the parent already has a
	// parent, adding a child would create depth > 2.
	if parent.ParentID != nil {
		return nil, ErrMaxDepthExceeded
	}

	// Get existing children to determine the next sequence number.
	children, err := s.taskRepo.GetChildren(parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get children of task %q: %w", parentID, err)
	}

	nextSeq := len(children) + 1
	childID := fmt.Sprintf("%s.%d", parentID, nextSeq)

	taskType := models.TaskTypeTask
	if dto.Type != "" {
		taskType = dto.Type
	}

	now := time.Now()
	child := &models.Task{
		ID:          childID,
		SpaceID:     parent.SpaceID,
		ParentID:    &parentID,
		CreatedByID: userID,
		Type:        taskType,
		Status:      models.TaskStatusOpen,
		Title:       dto.Title,
		Description: dto.Description,
		StationID:   dto.StationID,
		Priority:    2, // Default to P2 (Medium); callers can update after creation
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.taskRepo.Create(child); err != nil {
		return nil, fmt.Errorf("failed to create child task: %w", err)
	}

	// If an after dependency was specified, add a "blocks" dependency.
	if dto.AfterID != nil && *dto.AfterID != "" {
		// Validate the dependency target exists.
		if _, err := s.taskRepo.GetByID(*dto.AfterID); err != nil {
			return nil, fmt.Errorf("dependency target task %q not found: %w", *dto.AfterID, err)
		}

		dep := &models.TaskDep{
			FromTaskID: childID,
			ToTaskID:   *dto.AfterID,
			Type:       models.DepTypeBlocks,
			CreatedAt:  now,
		}
		if err := s.taskDepRepo.AddDep(dep); err != nil {
			return nil, fmt.Errorf("child task created but dependency creation failed: %w", err)
		}
	}

	return child, nil
}

// AddNote appends a deviation note to an existing task. If the task already
// has notes, the new text is appended with a newline separator.
func (s *TaskService) AddNote(taskID string, text string) (*models.Task, error) {
	if text == "" {
		return nil, fmt.Errorf("note text is required")
	}

	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, fmt.Errorf("task %q not found: %w", taskID, err)
	}

	if task.DeviationNotes != nil && *task.DeviationNotes != "" {
		combined := *task.DeviationNotes + "\n" + text
		task.DeviationNotes = &combined
	} else {
		task.DeviationNotes = &text
	}
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.Update(task); err != nil {
		return nil, err
	}

	return task, nil
}

// SkipTask marks a task as skipped. Convenience wrapper for SetTaskStatus.
func (s *TaskService) SkipTask(taskID string, userID uuid.UUID) (*models.Task, error) {
	return s.SetTaskStatus(taskID, userID, models.TaskStatusSkipped)
}
