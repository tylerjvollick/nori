package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
)

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
}

type TaskService struct {
	taskRepo    TaskRepositoryInterface
	taskDepRepo TaskDepRepositoryInterface
}

func NewTaskService(taskRepo TaskRepositoryInterface, taskDepRepo TaskDepRepositoryInterface) *TaskService {
	return &TaskService{taskRepo: taskRepo, taskDepRepo: taskDepRepo}
}

// CreateTask creates a new task in the given space.
func (s *TaskService) CreateTask(spaceID uuid.UUID, createdByID uuid.UUID, dto *dtos.CreateTaskRequest) (*models.Task, error) {
	if dto.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	task := &models.Task{
		ID:          uuid.New().String(), // placeholder ID generation; real hierarchical IDs come later
		SpaceID:     spaceID,
		CreatedByID: createdByID,
		Title:       dto.Title,
		Description: dto.Description,
		Type:        dto.Type,
		ParentID:    dto.ParentID,
		StationID:   dto.StationID,
		Status:      models.TaskStatusOpen,
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

// ClaimTask assigns a task to a user and sets it to active status.
// Returns an error if the task is not in "open" status or is already claimed.
func (s *TaskService) ClaimTask(taskID string, userID uuid.UUID) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, err
	}

	if task.Status != models.TaskStatusOpen {
		return nil, fmt.Errorf("task %q cannot be claimed: status is %q, must be %q", taskID, task.Status, models.TaskStatusOpen)
	}

	if task.AssignedToID != nil {
		return nil, fmt.Errorf("task %q is already assigned to user %s", taskID, task.AssignedToID.String())
	}

	now := time.Now()
	task.AssignedToID = &userID
	task.Status = models.TaskStatusActive
	task.StartedAt = &now
	task.UpdatedAt = now

	if err := s.taskRepo.Update(task); err != nil {
		return nil, err
	}

	return task, nil
}

// CompleteTask marks a task as done. Only the assigned user can complete it,
// and the task must be in "active" status. All blocking dependencies must be
// resolved (in a terminal status: done, skipped, or cancelled).
// If all sibling tasks under the same parent are now done, the parent is
// auto-completed as well.
func (s *TaskService) CompleteTask(taskID string, userID uuid.UUID) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, err
	}

	// Validate status: must be active (in_progress).
	if task.Status != models.TaskStatusActive {
		return nil, fmt.Errorf("task %q cannot be completed: status is %q, must be %q", taskID, task.Status, models.TaskStatusActive)
	}

	// Validate assignee: must be assigned to the requesting user.
	if task.AssignedToID == nil || *task.AssignedToID != userID {
		return nil, fmt.Errorf("task %q is not assigned to user %s", taskID, userID.String())
	}

	// Check blocking dependencies are resolved.
	// GetDependents returns deps where from_task_id = taskID (i.e., deps that this task has).
	// We filter for type "blocks" and check each blocker's status.
	deps, err := s.taskDepRepo.GetDependents(taskID)
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
			return nil, fmt.Errorf("task %q cannot be completed: blocked by task %q (status %q)", taskID, blocker.ID, blocker.Status)
		}
	}

	// Mark task as done.
	now := time.Now()
	task.Status = models.TaskStatusDone
	task.CompletedAt = &now
	task.UpdatedAt = now

	if err := s.taskRepo.Update(task); err != nil {
		return nil, err
	}

	// Auto-complete parent if all siblings are done.
	if task.ParentID != nil {
		if err := s.maybeCompleteParent(*task.ParentID); err != nil {
			// Log but don't fail the completion — the task itself is done.
			// In a production system we'd log this; for now we silently ignore.
			_ = err
		}
	}

	return task, nil
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
		Priority:    parent.Priority,
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

// ResumeTask resumes a paused task back to active status. Only the assigned user can resume it.
// Clears the PausedAt timestamp on resume.
func (s *TaskService) ResumeTask(taskID string, userID uuid.UUID) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, err
	}

	if task.Status != models.TaskStatusPaused {
		return nil, fmt.Errorf("task %q cannot be resumed: status is %q, must be %q", taskID, task.Status, models.TaskStatusPaused)
	}

	if task.AssignedToID == nil || *task.AssignedToID != userID {
		return nil, fmt.Errorf("task %q is not assigned to user %s", taskID, userID.String())
	}

	now := time.Now()
	task.Status = models.TaskStatusActive
	task.PausedAt = nil
	task.UpdatedAt = now

	if err := s.taskRepo.Update(task); err != nil {
		return nil, err
	}

	return task, nil
}

// SkipTask marks a task as skipped. Only the assigned user can skip it (if assigned).
// Skipping a task triggers downstream readiness checks via maybeCompleteParent.
// The task must not already be in a terminal status (done, skipped, cancelled).
func (s *TaskService) SkipTask(taskID string, userID uuid.UUID) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, err
	}

	if isTerminalStatus(task.Status) {
		return nil, fmt.Errorf("task %q cannot be skipped: status is %q (already terminal)", taskID, task.Status)
	}

	// If the task is assigned, only the assignee can skip it.
	if task.AssignedToID != nil && *task.AssignedToID != userID {
		return nil, fmt.Errorf("task %q is not assigned to user %s", taskID, userID.String())
	}

	now := time.Now()
	task.Status = models.TaskStatusSkipped
	task.CompletedAt = &now
	task.UpdatedAt = now

	if err := s.taskRepo.Update(task); err != nil {
		return nil, err
	}

	// Skipped is a terminal status — check if parent can be auto-completed.
	if task.ParentID != nil {
		if err := s.maybeCompleteParent(*task.ParentID); err != nil {
			_ = err
		}
	}

	return task, nil
}

// PauseTask pauses an active task. Only the assigned user can pause it.
// Returns an error if the task is not in "active" status or is not assigned to the user.
func (s *TaskService) PauseTask(taskID string, userID uuid.UUID) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, err
	}

	if task.Status != models.TaskStatusActive {
		return nil, fmt.Errorf("task %q cannot be paused: status is %q, must be %q", taskID, task.Status, models.TaskStatusActive)
	}

	if task.AssignedToID == nil || *task.AssignedToID != userID {
		return nil, fmt.Errorf("task %q is not assigned to user %s", taskID, userID.String())
	}

	now := time.Now()
	task.Status = models.TaskStatusPaused
	task.PausedAt = &now
	task.UpdatedAt = now

	if err := s.taskRepo.Update(task); err != nil {
		return nil, err
	}

	return task, nil
}
