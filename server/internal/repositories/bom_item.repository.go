package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type BOMItemRepository struct {
	db *gorm.DB
}

func NewBOMItemRepository(db *gorm.DB) *BOMItemRepository {
	return &BOMItemRepository{db: db}
}

func (r *BOMItemRepository) Create(item *models.BOMItem) error {
	return r.db.Create(item).Error
}

func (r *BOMItemRepository) GetByID(id int) (*models.BOMItem, error) {
	var item models.BOMItem
	err := r.db.First(&item, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("bom item not found")
		}
		return nil, err
	}
	return &item, nil
}

func (r *BOMItemRepository) GetByVersionID(versionID int) ([]models.BOMItem, error) {
	var items []models.BOMItem
	err := r.db.Where("sop_version_id = ?", versionID).
		Order("id ASC").
		Find(&items).Error
	return items, err
}

func (r *BOMItemRepository) GetByVersionIDWithMaterial(versionID int) ([]models.BOMItem, error) {
	var items []models.BOMItem
	err := r.db.Where("sop_version_id = ?", versionID).
		Preload("Material").
		Order("id ASC").
		Find(&items).Error
	return items, err
}

func (r *BOMItemRepository) GetByStepID(stepID int) ([]models.BOMItem, error) {
	var items []models.BOMItem
	err := r.db.Where("sop_step_id = ?", stepID).
		Order("id ASC").
		Find(&items).Error
	return items, err
}

func (r *BOMItemRepository) GetByMaterialID(materialID uuid.UUID) ([]models.BOMItem, error) {
	var items []models.BOMItem
	err := r.db.Where("material_id = ?", materialID).
		Order("id ASC").
		Find(&items).Error
	return items, err
}

func (r *BOMItemRepository) Update(item *models.BOMItem) error {
	return r.db.Save(item).Error
}

func (r *BOMItemRepository) Delete(id int) error {
	return r.db.Delete(&models.BOMItem{}, "id = ?", id).Error
}

func (r *BOMItemRepository) DeleteByVersionID(versionID int) error {
	return r.db.Where("sop_version_id = ?", versionID).Delete(&models.BOMItem{}).Error
}

func (r *BOMItemRepository) GetByVersionIDWithTx(tx *gorm.DB, versionID int) ([]models.BOMItem, error) {
	var items []models.BOMItem
	err := tx.Where("sop_version_id = ?", versionID).
		Order("id ASC").
		Find(&items).Error
	return items, err
}

func (r *BOMItemRepository) CreateBatchWithTx(tx *gorm.DB, items []models.BOMItem) error {
	if len(items) == 0 {
		return nil
	}
	return tx.Create(&items).Error
}
