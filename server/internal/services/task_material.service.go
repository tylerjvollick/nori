package services

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
)

// ErrTaskMaterialNotFound is returned when a task_material does not exist or
// is not reachable in the requested space.
var ErrTaskMaterialNotFound = errors.New("task material not found")

// ErrTaskMaterialDuplicate is re-exported for handler error mapping.
var ErrTaskMaterialDuplicate = repositories.ErrTaskMaterialDuplicate

// TaskMaterialValidationError indicates invalid input. Handlers map it to 400.
type TaskMaterialValidationError struct {
	Message string
}

func (e *TaskMaterialValidationError) Error() string { return e.Message }

// TaskMaterialRepositoryInterface is the repository surface used by the service.
type TaskMaterialRepositoryInterface interface {
	Create(tm *models.TaskMaterial) error
	GetByID(id uuid.UUID) (*models.TaskMaterial, error)
	ListByTaskID(taskID string) ([]models.TaskMaterial, error)
	Update(tm *models.TaskMaterial) error
	Delete(id uuid.UUID) error
}

// taskLookupRepo is the subset of the task repository used for space scoping.
type taskLookupRepo interface {
	GetByID(id string) (*models.Task, error)
}

// materialLookupRepo is the subset of the material repository used to validate
// that a material exists in the space.
type materialLookupRepo interface {
	GetByID(id uuid.UUID) (*models.Material, error)
}

// TaskMaterialService manages materials attached to recipe/job tasks.
type TaskMaterialService struct {
	repo         TaskMaterialRepositoryInterface
	taskRepo     taskLookupRepo
	materialRepo materialLookupRepo
}

func NewTaskMaterialService(
	repo TaskMaterialRepositoryInterface,
	taskRepo taskLookupRepo,
	materialRepo materialLookupRepo,
) *TaskMaterialService {
	return &TaskMaterialService{repo: repo, taskRepo: taskRepo, materialRepo: materialRepo}
}

// requireTaskInSpace verifies the task exists and belongs to the given space.
func (s *TaskMaterialService) requireTaskInSpace(spaceID uuid.UUID, taskID string) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, ErrTaskMaterialNotFound
	}
	if task.SpaceID != spaceID {
		return nil, ErrTaskMaterialNotFound
	}
	return task, nil
}

// Add attaches a material to a task.
func (s *TaskMaterialService) Add(spaceID uuid.UUID, taskID string, req *dtos.CreateTaskMaterialRequest) (*models.TaskMaterial, error) {
	if _, err := s.requireTaskInSpace(spaceID, taskID); err != nil {
		return nil, err
	}

	if req.QuantityPerUnit.IsZero() || req.QuantityPerUnit.IsNegative() {
		return nil, &TaskMaterialValidationError{Message: "quantityPerUnit must be > 0"}
	}

	// Material must exist in the same space.
	material, err := s.materialRepo.GetByID(req.MaterialID)
	if err != nil || material.SpaceID != spaceID {
		return nil, &TaskMaterialValidationError{Message: "material not found in space"}
	}

	tm := &models.TaskMaterial{
		TaskID:          taskID,
		MaterialID:      req.MaterialID,
		QuantityPerUnit: req.QuantityPerUnit,
		Notes:           req.Notes,
	}
	if err := s.repo.Create(tm); err != nil {
		if errors.Is(err, ErrTaskMaterialDuplicate) {
			return nil, ErrTaskMaterialDuplicate
		}
		return nil, fmt.Errorf("creating task material: %w", err)
	}

	// Reload with Material preloaded for the response.
	return s.repo.GetByID(tm.ID)
}

// List returns the materials attached to a task.
func (s *TaskMaterialService) List(spaceID uuid.UUID, taskID string) ([]models.TaskMaterial, error) {
	if _, err := s.requireTaskInSpace(spaceID, taskID); err != nil {
		return nil, err
	}
	return s.repo.ListByTaskID(taskID)
}

// Update changes the quantity and/or notes on a task material.
func (s *TaskMaterialService) Update(spaceID uuid.UUID, taskID string, id uuid.UUID, req *dtos.UpdateTaskMaterialRequest) (*models.TaskMaterial, error) {
	if _, err := s.requireTaskInSpace(spaceID, taskID); err != nil {
		return nil, err
	}

	tm, err := s.repo.GetByID(id)
	if err != nil || tm.TaskID != taskID {
		return nil, ErrTaskMaterialNotFound
	}

	if req.QuantityPerUnit != nil {
		if req.QuantityPerUnit.IsZero() || req.QuantityPerUnit.IsNegative() {
			return nil, &TaskMaterialValidationError{Message: "quantityPerUnit must be > 0"}
		}
		tm.QuantityPerUnit = *req.QuantityPerUnit
	}
	if req.Notes != nil {
		tm.Notes = req.Notes
	}

	if err := s.repo.Update(tm); err != nil {
		return nil, fmt.Errorf("updating task material: %w", err)
	}
	return s.repo.GetByID(tm.ID)
}

// Remove detaches a material from a task.
func (s *TaskMaterialService) Remove(spaceID uuid.UUID, taskID string, id uuid.UUID) error {
	if _, err := s.requireTaskInSpace(spaceID, taskID); err != nil {
		return err
	}

	tm, err := s.repo.GetByID(id)
	if err != nil || tm.TaskID != taskID {
		return ErrTaskMaterialNotFound
	}

	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("deleting task material: %w", err)
	}
	return nil
}
