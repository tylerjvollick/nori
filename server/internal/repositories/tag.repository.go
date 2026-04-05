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

// AddTagToTask creates an association between a task and a tag.
func (r *TagRepository) AddTagToTask(taskID string, tagID uuid.UUID) error {
	return r.db.Create(&models.TaskTag{TaskID: taskID, TagID: tagID}).Error
}

// RemoveTagFromTask removes the association between a task and a tag.
func (r *TagRepository) RemoveTagFromTask(taskID string, tagID uuid.UUID) error {
	return r.db.Delete(&models.TaskTag{}, "task_id = ? AND tag_id = ?", taskID, tagID).Error
}

// GetTagsByTaskID returns all tags associated with a task.
func (r *TagRepository) GetTagsByTaskID(taskID string) ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Joins("JOIN task_tag ON task_tag.tag_id = tag.id").
		Where("task_tag.task_id = ?", taskID).
		Order("tag.name ASC").
		Find(&tags).Error
	return tags, err
}

// AddTagToRecipe creates an association between a recipe and a tag.
func (r *TagRepository) AddTagToRecipe(recipeID uuid.UUID, tagID uuid.UUID) error {
	return r.db.Create(&models.RecipeTag{RecipeID: recipeID, TagID: tagID}).Error
}

// RemoveTagFromRecipe removes the association between a recipe and a tag.
func (r *TagRepository) RemoveTagFromRecipe(recipeID uuid.UUID, tagID uuid.UUID) error {
	return r.db.Delete(&models.RecipeTag{}, "recipe_id = ? AND tag_id = ?", recipeID, tagID).Error
}

// GetTagsByRecipeID returns all tags associated with a recipe.
func (r *TagRepository) GetTagsByRecipeID(recipeID uuid.UUID) ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Joins("JOIN recipe_tag ON recipe_tag.tag_id = tag.id").
		Where("recipe_tag.recipe_id = ?", recipeID).
		Order("tag.name ASC").
		Find(&tags).Error
	return tags, err
}
