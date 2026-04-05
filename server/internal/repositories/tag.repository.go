package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type TagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) Create(tag *models.Tag) error {
	return r.db.Create(tag).Error
}

func (r *TagRepository) GetByID(id uuid.UUID) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.First(&tag, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tag not found")
		}
		return nil, err
	}
	return &tag, nil
}

func (r *TagRepository) GetBySpaceID(spaceID uuid.UUID) ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Where("space_id = ?", spaceID).
		Order("name ASC").
		Find(&tags).Error
	return tags, err
}

func (r *TagRepository) GetByName(spaceID uuid.UUID, name string) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.Where("space_id = ? AND name = ?", spaceID, name).First(&tag).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tag not found")
		}
		return nil, err
	}
	return &tag, nil
}

func (r *TagRepository) Update(tag *models.Tag) error {
	return r.db.Save(tag).Error
}

func (r *TagRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Tag{}, "id = ?", id).Error
}
