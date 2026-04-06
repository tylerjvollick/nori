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
