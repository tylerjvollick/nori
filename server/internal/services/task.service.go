package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
)

type TaskService struct {
	taskRepo *repositories.TaskRepository
}

func NewTaskService(taskRepo *repositories.TaskRepository) *TaskService {
	return &TaskService{taskRepo: taskRepo}
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
