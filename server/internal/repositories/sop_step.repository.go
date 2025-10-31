package repositories

import (
	"fmt"

	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type SOPStepRepository struct {
	db *gorm.DB
}

func NewSOPStepRepository(db *gorm.DB) *SOPStepRepository {
	return &SOPStepRepository{db: db}
}

func (r *SOPStepRepository) Create(step *models.SOPStep) error {
	return r.db.Create(step).Error
}

func (r *SOPStepRepository) CreateBatch(steps []models.SOPStep) error {
	if len(steps) == 0 {
		return nil
	}
	return r.db.Create(&steps).Error
}

func (r *SOPStepRepository) GetByID(id int) (*models.SOPStep, error) {
	var step models.SOPStep
	err := r.db.First(&step, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("SOP step not found")
		}
		return nil, err
	}
	return &step, nil
}

func (r *SOPStepRepository) GetByVersionID(versionID int) ([]models.SOPStep, error) {
	var steps []models.SOPStep
	err := r.db.Where("sop_template_version_id = ?", versionID).
		Order("step_number ASC").
		Find(&steps).Error
	return steps, err
}

func (r *SOPStepRepository) Update(step *models.SOPStep) error {
	return r.db.Save(step).Error
}

func (r *SOPStepRepository) Delete(id int) error {
	return r.db.Delete(&models.SOPStep{}, id).Error
}

func (r *SOPStepRepository) DeleteByVersionID(versionID int) error {
	return r.db.Where("sop_template_version_id = ?", versionID).Delete(&models.SOPStep{}).Error
}
