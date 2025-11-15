package repositories

import (
	"fmt"

	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type SOPStepPhotoRepository struct {
	db *gorm.DB
}

func NewSOPStepPhotoRepository(db *gorm.DB) *SOPStepPhotoRepository {
	return &SOPStepPhotoRepository{db: db}
}

func (r *SOPStepPhotoRepository) Create(photo *models.SOPStepPhoto) error {
	return r.db.Create(photo).Error
}

func (r *SOPStepPhotoRepository) GetByID(id int) (*models.SOPStepPhoto, error) {
	var photo models.SOPStepPhoto
	err := r.db.First(&photo, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("photo not found")
		}
		return nil, err
	}
	return &photo, nil
}

func (r *SOPStepPhotoRepository) GetByUUID(uuid string) (*models.SOPStepPhoto, error) {
	var photo models.SOPStepPhoto
	err := r.db.Where("uuid = ?", uuid).First(&photo).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("photo not found")
		}
		return nil, err
	}
	return &photo, nil
}

func (r *SOPStepPhotoRepository) GetByStepID(stepID int) ([]models.SOPStepPhoto, error) {
	var photos []models.SOPStepPhoto
	err := r.db.Where("sop_step_id = ?", stepID).
		Order("\"order\" ASC").
		Find(&photos).Error
	return photos, err
}

func (r *SOPStepPhotoRepository) Update(photo *models.SOPStepPhoto) error {
	return r.db.Save(photo).Error
}

func (r *SOPStepPhotoRepository) UpdateWithTx(tx *gorm.DB, photo *models.SOPStepPhoto) error {
	return tx.Save(photo).Error
}

func (r *SOPStepPhotoRepository) Delete(id int) error {
	return r.db.Delete(&models.SOPStepPhoto{}, id).Error
}

func (r *SOPStepPhotoRepository) DeleteWithTx(tx *gorm.DB, id int) error {
	return tx.Delete(&models.SOPStepPhoto{}, id).Error
}

func (r *SOPStepPhotoRepository) DeleteByStepID(stepID int) error {
	return r.db.Where("sop_step_id = ?", stepID).Delete(&models.SOPStepPhoto{}).Error
}

// GetLastOrderByStepID gets the last (highest) order value for a step
func (r *SOPStepPhotoRepository) GetLastOrderByStepID(stepID int) (string, error) {
	var lastOrder string
	err := r.db.Model(&models.SOPStepPhoto{}).
		Where("sop_step_id = ?", stepID).
		Select("COALESCE(MAX(\"order\"), '')").
		Scan(&lastOrder).Error
	return lastOrder, err
}

// GetOrderBeforeAndAfter gets the order values of photos before and after a given position
func (r *SOPStepPhotoRepository) GetOrderBeforeAndAfter(stepID int, beforePhotoID, afterPhotoID *int) (string, string, error) {
	var beforeOrder, afterOrder string

	if beforePhotoID != nil {
		var photo models.SOPStepPhoto
		if err := r.db.Where("id = ? AND sop_step_id = ?", *beforePhotoID, stepID).First(&photo).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return "", "", err
			}
		} else {
			beforeOrder = photo.Order
		}
	}

	if afterPhotoID != nil {
		var photo models.SOPStepPhoto
		if err := r.db.Where("id = ? AND sop_step_id = ?", *afterPhotoID, stepID).First(&photo).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return "", "", err
			}
		} else {
			afterOrder = photo.Order
		}
	}

	return beforeOrder, afterOrder, nil
}

// UpdateOrderWithTx updates a photo's order value
func (r *SOPStepPhotoRepository) UpdateOrderWithTx(tx *gorm.DB, photoID int, newOrder string) error {
	return tx.Model(&models.SOPStepPhoto{}).Where("id = ?", photoID).Update("\"order\"", newOrder).Error
}
