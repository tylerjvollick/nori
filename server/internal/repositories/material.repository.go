package repositories

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type MaterialRepository struct {
	db *gorm.DB
}

func NewMaterialRepository(db *gorm.DB) *MaterialRepository {
	return &MaterialRepository{db: db}
}

func (r *MaterialRepository) Create(material *models.Material) error {
	return r.db.Create(material).Error
}

func (r *MaterialRepository) GetByID(id uuid.UUID) (*models.Material, error) {
	var material models.Material
	err := r.db.First(&material, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("material not found")
		}
		return nil, err
	}
	return &material, nil
}

// List returns all non-deleted materials for a space ordered by name.
// When search is non-empty, results are filtered to names containing the
// search string (case-insensitive).
func (r *MaterialRepository) List(spaceID uuid.UUID, search string) ([]models.Material, error) {
	query := r.db.Where("space_id = ?", spaceID)
	if search != "" {
		query = query.Where("name ILIKE ?", "%"+escapeLikePattern(search)+"%")
	}

	var materials []models.Material
	err := query.Order("name ASC").Find(&materials).Error
	return materials, err
}

// escapeLikePattern escapes LIKE/ILIKE wildcard characters in user input so
// they are matched literally.
func escapeLikePattern(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(s)
}

func (r *MaterialRepository) GetBySpaceID(spaceID uuid.UUID) ([]models.Material, error) {
	var materials []models.Material
	err := r.db.Where("space_id = ?", spaceID).
		Order("name ASC").
		Find(&materials).Error
	return materials, err
}

func (r *MaterialRepository) GetBySpaceIDAndCategory(spaceID uuid.UUID, category models.MaterialCategory) ([]models.Material, error) {
	var materials []models.Material
	err := r.db.Where("space_id = ? AND category = ?", spaceID, category).
		Order("name ASC").
		Find(&materials).Error
	return materials, err
}

func (r *MaterialRepository) GetActiveBySpaceID(spaceID uuid.UUID) ([]models.Material, error) {
	var materials []models.Material
	err := r.db.Where("space_id = ? AND is_active = true", spaceID).
		Order("name ASC").
		Find(&materials).Error
	return materials, err
}

func (r *MaterialRepository) GetBelowReorderThreshold(spaceID uuid.UUID) ([]models.Material, error) {
	var materials []models.Material
	err := r.db.Where("space_id = ? AND is_active = true AND current_stock <= reorder_threshold", spaceID).
		Order("name ASC").
		Find(&materials).Error
	return materials, err
}

func (r *MaterialRepository) Update(material *models.Material) error {
	return r.db.Save(material).Error
}

func (r *MaterialRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Material{}, "id = ?", id).Error
}
