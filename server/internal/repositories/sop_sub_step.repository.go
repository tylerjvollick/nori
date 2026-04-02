package repositories

import (
	"fmt"

	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type SOPSubStepRepository struct {
	db *gorm.DB
}

func NewSOPSubStepRepository(db *gorm.DB) *SOPSubStepRepository {
	return &SOPSubStepRepository{db: db}
}

func (r *SOPSubStepRepository) Create(subStep *models.SOPSubStep) error {
	return r.db.Create(subStep).Error
}

func (r *SOPSubStepRepository) CreateBatch(subSteps []models.SOPSubStep) error {
	if len(subSteps) == 0 {
		return nil
	}
	return r.db.Create(&subSteps).Error
}

func (r *SOPSubStepRepository) GetByID(id int) (*models.SOPSubStep, error) {
	var subStep models.SOPSubStep
	err := r.db.First(&subStep, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("SOP sub-step not found")
		}
		return nil, err
	}
	return &subStep, nil
}

func (r *SOPSubStepRepository) GetByStepID(stepID int) ([]models.SOPSubStep, error) {
	var subSteps []models.SOPSubStep
	err := r.db.Where("sop_step_id = ?", stepID).
		Order("display_order ASC").
		Find(&subSteps).Error
	return subSteps, err
}

func (r *SOPSubStepRepository) GetByStepIDWithTx(tx *gorm.DB, stepID int) ([]models.SOPSubStep, error) {
	var subSteps []models.SOPSubStep
	err := tx.Where("sop_step_id = ?", stepID).
		Order("display_order ASC").
		Find(&subSteps).Error
	return subSteps, err
}

func (r *SOPSubStepRepository) Update(subStep *models.SOPSubStep) error {
	return r.db.Save(subStep).Error
}

func (r *SOPSubStepRepository) UpdateWithTx(tx *gorm.DB, subStep *models.SOPSubStep) error {
	return tx.Save(subStep).Error
}

func (r *SOPSubStepRepository) Delete(id int) error {
	return r.db.Delete(&models.SOPSubStep{}, id).Error
}

func (r *SOPSubStepRepository) DeleteWithTx(tx *gorm.DB, id int) error {
	return tx.Delete(&models.SOPSubStep{}, id).Error
}

func (r *SOPSubStepRepository) DeleteByStepID(stepID int) error {
	return r.db.Where("sop_step_id = ?", stepID).Delete(&models.SOPSubStep{}).Error
}

func (r *SOPSubStepRepository) DeleteByStepIDWithTx(tx *gorm.DB, stepID int) error {
	return tx.Where("sop_step_id = ?", stepID).Delete(&models.SOPSubStep{}).Error
}

func (r *SOPSubStepRepository) CreateBatchWithTx(tx *gorm.DB, subSteps []models.SOPSubStep) error {
	if len(subSteps) == 0 {
		return nil
	}
	return tx.Create(&subSteps).Error
}

// Reorder updates the display_order of all sub-steps for a given step.
// The ids slice defines the new order — ids[0] gets display_order 0, etc.
func (r *SOPSubStepRepository) Reorder(stepID int, ids []int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			err := tx.Model(&models.SOPSubStep{}).
				Where("id = ? AND sop_step_id = ?", id, stepID).
				Update("display_order", i).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// GetMaxDisplayOrder returns the highest display_order for sub-steps in a given step.
func (r *SOPSubStepRepository) GetMaxDisplayOrder(stepID int) (int, error) {
	var maxOrder int
	err := r.db.Model(&models.SOPSubStep{}).
		Where("sop_step_id = ?", stepID).
		Select("COALESCE(MAX(display_order), -1)").
		Scan(&maxOrder).Error
	return maxOrder, err
}
