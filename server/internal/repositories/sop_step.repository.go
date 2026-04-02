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
	err := r.db.Where("sop_version_id = ?", versionID).
		Order("\"order\" ASC").
		Find(&steps).Error
	return steps, err
}

func (r *SOPStepRepository) GetByVersionIDWithTx(tx *gorm.DB, versionID int) ([]models.SOPStep, error) {
	var steps []models.SOPStep
	err := tx.Where("sop_version_id = ?", versionID).
		Order("\"order\" ASC").
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
	return r.db.Where("sop_version_id = ?", versionID).Delete(&models.SOPStep{}).Error
}

func (r *SOPStepRepository) DeleteByVersionIDWithTx(tx *gorm.DB, versionID int) error {
	return tx.Where("sop_version_id = ?", versionID).Delete(&models.SOPStep{}).Error
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
	err := r.db.Where("id = ? AND sop_version_id = ?", stepID, versionID).First(&step).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("step not found in this version")
		}
		return nil, err
	}
	return &step, nil
}

// GetLastOrderByVersionID gets the last (highest) order value for a version
func (r *SOPStepRepository) GetLastOrderByVersionID(versionID int) (string, error) {
	var lastOrder string
	err := r.db.Model(&models.SOPStep{}).
		Where("sop_version_id = ?", versionID).
		Select("COALESCE(MAX(\"order\"), '')").
		Scan(&lastOrder).Error
	return lastOrder, err
}

// GetOrderBeforeAndAfter gets the order values of steps before and after a given position
// Used for reordering steps
func (r *SOPStepRepository) GetOrderBeforeAndAfter(versionID int, beforeStepID, afterStepID *int) (string, string, error) {
	var beforeOrder, afterOrder string

	if beforeStepID != nil {
		var step models.SOPStep
		if err := r.db.Where("id = ? AND sop_version_id = ?", *beforeStepID, versionID).First(&step).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return "", "", err
			}
		} else {
			beforeOrder = step.Order
		}
	}

	if afterStepID != nil {
		var step models.SOPStep
		if err := r.db.Where("id = ? AND sop_version_id = ?", *afterStepID, versionID).First(&step).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return "", "", err
			}
		} else {
			afterOrder = step.Order
		}
	}

	return beforeOrder, afterOrder, nil
}

// UpdateOrderWithTx updates a step's order value
func (r *SOPStepRepository) UpdateOrderWithTx(tx *gorm.DB, stepID int, newOrder string) error {
	return tx.Model(&models.SOPStep{}).Where("id = ?", stepID).Update("\"order\"", newOrder).Error
}
