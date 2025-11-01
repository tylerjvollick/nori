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

func (r *SOPStepRepository) DeleteByVersionIDWithTx(tx *gorm.DB, versionID int) error {
	return tx.Where("sop_template_version_id = ?", versionID).Delete(&models.SOPStep{}).Error
}

func (r *SOPStepRepository) CreateBatchWithTx(tx *gorm.DB, steps []models.SOPStep) error {
	if len(steps) == 0 {
		return nil
	}
	return tx.Create(&steps).Error
}

func (r *SOPStepRepository) UpdateWithTx(tx *gorm.DB, step *models.SOPStep) error {
	return tx.Save(step).Error
}

func (r *SOPStepRepository) DeleteWithTx(tx *gorm.DB, id int) error {
	return tx.Delete(&models.SOPStep{}, id).Error
}

// GetByIDAndVersionID gets a step by ID and verifies it belongs to the specified version
func (r *SOPStepRepository) GetByIDAndVersionID(stepID int, versionID int) (*models.SOPStep, error) {
	var step models.SOPStep
	err := r.db.Where("id = ? AND sop_template_version_id = ?", stepID, versionID).First(&step).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("step not found in this version")
		}
		return nil, err
	}
	return &step, nil
}

// GetMaxStepNumber gets the highest step number for a version
func (r *SOPStepRepository) GetMaxStepNumber(versionID int) (int, error) {
	var maxStepNumber int
	err := r.db.Model(&models.SOPStep{}).
		Where("sop_template_version_id = ?", versionID).
		Select("COALESCE(MAX(step_number), 0)").
		Scan(&maxStepNumber).Error
	return maxStepNumber, err
}

// UpdateStepNumbersWithTx updates step numbers in bulk (for reordering)
func (r *SOPStepRepository) UpdateStepNumbersWithTx(tx *gorm.DB, updates map[int]int) error {
	// updates is a map of stepID -> newStepNumber
	for stepID, newStepNumber := range updates {
		if err := tx.Model(&models.SOPStep{}).Where("id = ?", stepID).Update("step_number", newStepNumber).Error; err != nil {
			return err
		}
	}
	return nil
}
