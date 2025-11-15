package repositories

import (
	"fmt"

	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type SOPStepMediaRepository struct {
	db *gorm.DB
}

func NewSOPStepMediaRepository(db *gorm.DB) *SOPStepMediaRepository {
	return &SOPStepMediaRepository{db: db}
}

func (r *SOPStepMediaRepository) Create(media *models.SOPStepMedia) error {
	return r.db.Create(media).Error
}

func (r *SOPStepMediaRepository) GetByID(id int) (*models.SOPStepMedia, error) {
	var media models.SOPStepMedia
	err := r.db.First(&media, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("media not found")
		}
		return nil, err
	}
	return &media, nil
}

func (r *SOPStepMediaRepository) GetByUUID(uuid string) (*models.SOPStepMedia, error) {
	var media models.SOPStepMedia
	err := r.db.Where("uuid = ?", uuid).First(&media).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("media not found")
		}
		return nil, err
	}
	return &media, nil
}

func (r *SOPStepMediaRepository) GetByStepID(stepID int) ([]models.SOPStepMedia, error) {
	var mediaItems []models.SOPStepMedia
	err := r.db.Where("sop_step_id = ?", stepID).
		Order("\"order\" ASC").
		Find(&mediaItems).Error
	return mediaItems, err
}

func (r *SOPStepMediaRepository) Update(media *models.SOPStepMedia) error {
	return r.db.Save(media).Error
}

func (r *SOPStepMediaRepository) UpdateWithTx(tx *gorm.DB, media *models.SOPStepMedia) error {
	return tx.Save(media).Error
}

func (r *SOPStepMediaRepository) Delete(id int) error {
	return r.db.Delete(&models.SOPStepMedia{}, id).Error
}

func (r *SOPStepMediaRepository) DeleteWithTx(tx *gorm.DB, id int) error {
	return tx.Delete(&models.SOPStepMedia{}, id).Error
}

func (r *SOPStepMediaRepository) DeleteByStepID(stepID int) error {
	return r.db.Where("sop_step_id = ?", stepID).Delete(&models.SOPStepMedia{}).Error
}

// GetLastOrderByStepID gets the last (highest) order value for a step
func (r *SOPStepMediaRepository) GetLastOrderByStepID(stepID int) (string, error) {
	var lastOrder string
	err := r.db.Model(&models.SOPStepMedia{}).
		Where("sop_step_id = ?", stepID).
		Select("COALESCE(MAX(\"order\"), '')").
		Scan(&lastOrder).Error
	return lastOrder, err
}

// GetOrderBeforeAndAfter gets the order values of media before and after a given position
func (r *SOPStepMediaRepository) GetOrderBeforeAndAfter(stepID int, beforeMediaID, afterMediaID *int) (string, string, error) {
	var beforeOrder, afterOrder string

	if beforeMediaID != nil {
		var media models.SOPStepMedia
		if err := r.db.Where("id = ? AND sop_step_id = ?", *beforeMediaID, stepID).First(&media).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return "", "", err
			}
		} else {
			beforeOrder = media.Order
		}
	}

	if afterMediaID != nil {
		var media models.SOPStepMedia
		if err := r.db.Where("id = ? AND sop_step_id = ?", *afterMediaID, stepID).First(&media).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return "", "", err
			}
		} else {
			afterOrder = media.Order
		}
	}

	return beforeOrder, afterOrder, nil
}

// UpdateOrderWithTx updates a media's order value
func (r *SOPStepMediaRepository) UpdateOrderWithTx(tx *gorm.DB, mediaID int, newOrder string) error {
	return tx.Model(&models.SOPStepMedia{}).Where("id = ?", mediaID).Update("\"order\"", newOrder).Error
}
