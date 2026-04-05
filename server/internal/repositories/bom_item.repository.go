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

func (r *BOMItemRepository) GetByMaterialID(materialID uuid.UUID) ([]models.BOMItem, error) {
	var items []models.BOMItem
	err := r.db.Where("material_id = ?", materialID).
		Order("id ASC").
		Find(&items).Error
	return items, err
}

// GetByRecipeVersionID returns all BOM items for a recipe version, ordered by ID.
func (r *BOMItemRepository) GetByRecipeVersionID(recipeVersionID int) ([]models.BOMItem, error) {
	var items []models.BOMItem
	err := r.db.Where("recipe_version_id = ?", recipeVersionID).
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
