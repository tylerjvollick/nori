package repositories

import (
	"errors"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

// ErrTaskMaterialDuplicate is returned when a material is added to a task that
// already has it (violates the composite unique constraint).
var ErrTaskMaterialDuplicate = errors.New("material already attached to task")

type TaskMaterialRepository struct {
	db *gorm.DB
}

func NewTaskMaterialRepository(db *gorm.DB) *TaskMaterialRepository {
	return &TaskMaterialRepository{db: db}
}

// Create inserts a task_material. A duplicate (task_id, material_id) is
// surfaced as ErrTaskMaterialDuplicate.
func (r *TaskMaterialRepository) Create(tm *models.TaskMaterial) error {
	err := r.db.Create(tm).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrTaskMaterialDuplicate
	}
	return err
}

// GetByID loads a single task_material with its Material preloaded.
func (r *TaskMaterialRepository) GetByID(id uuid.UUID) (*models.TaskMaterial, error) {
	var tm models.TaskMaterial
	if err := r.db.Preload("Material").First(&tm, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &tm, nil
}

// ListByTaskID returns the materials attached to a task, with Material
// preloaded, ordered by creation time.
func (r *TaskMaterialRepository) ListByTaskID(taskID string) ([]models.TaskMaterial, error) {
	var materials []models.TaskMaterial
	err := r.db.Preload("Material").
		Where("task_id = ?", taskID).
		Order("created_at ASC").
		Find(&materials).Error
	return materials, err
}

// ListByTaskIDs returns the materials attached to any of the given tasks.
// Used when cloning a task tree.
func (r *TaskMaterialRepository) ListByTaskIDs(taskIDs []string) ([]models.TaskMaterial, error) {
	if len(taskIDs) == 0 {
		return nil, nil
	}
	var materials []models.TaskMaterial
	err := r.db.Where("task_id IN ?", taskIDs).
		Order("created_at ASC").
		Find(&materials).Error
	return materials, err
}

func (r *TaskMaterialRepository) Update(tm *models.TaskMaterial) error {
	return r.db.Save(tm).Error
}

func (r *TaskMaterialRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.TaskMaterial{}, "id = ?", id).Error
}
