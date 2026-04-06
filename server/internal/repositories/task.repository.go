package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

// TaskFilter defines the optional filters for listing tasks.
type TaskFilter struct {
	SpaceID    *uuid.UUID
	Status     *models.TaskStatus
	Type       *models.TaskType
	StationID  *uuid.UUID
	AssigneeID *uuid.UUID
	ParentID   *string // use pointer to distinguish "no filter" from "filter for root tasks (nil parent)"
	Offset     int
	Limit      int
}

type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// WithTx returns a new TaskRepository that uses the given transaction handle.
func (r *TaskRepository) WithTx(tx *gorm.DB) *TaskRepository {
	return &TaskRepository{db: tx}
}

func (r *TaskRepository) Create(task *models.Task) error {
	return r.db.Create(task).Error
}

func (r *TaskRepository) GetByID(id string) (*models.Task, error) {
	var task models.Task
	err := r.db.First(&task, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("task not found")
		}
		return nil, err
	}
	return &task, nil
}

// List returns tasks matching the given filter along with the total count
// (before pagination). Offset and Limit on the filter control pagination.
func (r *TaskRepository) List(filter TaskFilter) ([]models.Task, int64, error) {
	query := r.db.Model(&models.Task{})

	if filter.SpaceID != nil {
		query = query.Where("space_id = ?", *filter.SpaceID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.Type != nil {
		query = query.Where("type = ?", *filter.Type)
	}
	if filter.StationID != nil {
		query = query.Where("station_id = ?", *filter.StationID)
	}
	if filter.AssigneeID != nil {
		query = query.Where("assigned_to_id = ?", *filter.AssigneeID)
	}
	if filter.ParentID != nil {
		query = query.Where("parent_id = ?", *filter.ParentID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tasks []models.Task
	q := query.Order("display_order ASC, created_at ASC")
	if filter.Offset > 0 {
		q = q.Offset(filter.Offset)
	}
	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	}
	if err := q.Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

func (r *TaskRepository) Update(task *models.Task) error {
	return r.db.Save(task).Error
}

func (r *TaskRepository) Delete(id string) error {
	result := r.db.Delete(&models.Task{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("task not found")
	}
	return nil
}

// GetChildren returns the direct children of a task, ordered by display_order.
func (r *TaskRepository) GetChildren(parentID string) ([]models.Task, error) {
	var tasks []models.Task
	err := r.db.Where("parent_id = ?", parentID).
		Order("display_order ASC").
		Find(&tasks).Error
	return tasks, err
}

// GetRoot walks up the ParentID chain from the given task and returns the
// root task (the one with no parent). Returns an error if any task in the
// chain is not found. Includes a safety limit to prevent infinite loops.
func (r *TaskRepository) GetRoot(taskID string) (*models.Task, error) {
	const maxDepth = 100

	current, err := r.GetByID(taskID)
	if err != nil {
		return nil, err
	}

	for i := 0; i < maxDepth; i++ {
		if current.ParentID == nil {
			return current, nil
		}
		current, err = r.GetByID(*current.ParentID)
		if err != nil {
			return nil, fmt.Errorf("broken parent chain at %q: %w", taskID, err)
		}
	}

	return nil, fmt.Errorf("parent chain exceeds maximum depth of %d from task %q", maxDepth, taskID)
}
