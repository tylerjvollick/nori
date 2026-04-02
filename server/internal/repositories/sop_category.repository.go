package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type SOPCategoryRepository struct {
	db *gorm.DB
}

func NewSOPCategoryRepository(db *gorm.DB) *SOPCategoryRepository {
	return &SOPCategoryRepository{db: db}
}

func (r *SOPCategoryRepository) Create(category *models.SOPCategory) error {
	return r.db.Create(category).Error
}

func (r *SOPCategoryRepository) GetByID(id uuid.UUID) (*models.SOPCategory, error) {
	var category models.SOPCategory
	err := r.db.First(&category, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("sop category not found")
		}
		return nil, err
	}
	return &category, nil
}

// GetBySpaceID returns all categories for a space, ordered by display_order.
// Use BuildTree to assemble the flat list into a hierarchy.
func (r *SOPCategoryRepository) GetBySpaceID(spaceID uuid.UUID) ([]models.SOPCategory, error) {
	var categories []models.SOPCategory
	err := r.db.Where("space_id = ?", spaceID).
		Order("display_order ASC, name ASC").
		Find(&categories).Error
	return categories, err
}

// GetByParentID returns child categories under a given parent (or root if parentID is nil).
func (r *SOPCategoryRepository) GetByParentID(spaceID uuid.UUID, parentID *uuid.UUID) ([]models.SOPCategory, error) {
	var categories []models.SOPCategory
	query := r.db.Where("space_id = ?", spaceID)
	if parentID == nil {
		query = query.Where("parent_category_id IS NULL")
	} else {
		query = query.Where("parent_category_id = ?", *parentID)
	}
	err := query.Order("display_order ASC, name ASC").
		Find(&categories).Error
	return categories, err
}

func (r *SOPCategoryRepository) Update(category *models.SOPCategory) error {
	return r.db.Save(category).Error
}

func (r *SOPCategoryRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.SOPCategory{}, "id = ?", id).Error
}

// MoveCategory updates a category's parent and display order.
func (r *SOPCategoryRepository) MoveCategory(id uuid.UUID, newParentID *uuid.UUID, displayOrder int) error {
	updates := map[string]interface{}{
		"parent_category_id": newParentID,
		"display_order":      displayOrder,
	}
	return r.db.Model(&models.SOPCategory{}).Where("id = ?", id).Updates(updates).Error
}

// GetMaxDisplayOrder returns the highest display_order among siblings (children of parentID).
func (r *SOPCategoryRepository) GetMaxDisplayOrder(spaceID uuid.UUID, parentID *uuid.UUID) (int, error) {
	var maxOrder *int
	query := r.db.Model(&models.SOPCategory{}).Where("space_id = ?", spaceID)
	if parentID == nil {
		query = query.Where("parent_category_id IS NULL")
	} else {
		query = query.Where("parent_category_id = ?", *parentID)
	}
	err := query.Select("MAX(display_order)").Scan(&maxOrder).Error
	if err != nil {
		return 0, err
	}
	if maxOrder == nil {
		return 0, nil
	}
	return *maxOrder, nil
}

// BuildTree assembles a flat list of categories into a tree structure.
// Root categories (ParentCategoryID == nil) are returned as top-level items
// with their Children populated recursively.
func BuildTree(categories []models.SOPCategory) []models.SOPCategory {
	byParent := make(map[uuid.UUID][]models.SOPCategory)
	var roots []models.SOPCategory

	for _, c := range categories {
		if c.ParentCategoryID == nil {
			roots = append(roots, c)
		} else {
			byParent[*c.ParentCategoryID] = append(byParent[*c.ParentCategoryID], c)
		}
	}

	for i := range roots {
		populateChildren(&roots[i], byParent)
	}
	return roots
}

func populateChildren(category *models.SOPCategory, byParent map[uuid.UUID][]models.SOPCategory) {
	children, ok := byParent[category.ID]
	if !ok {
		return
	}
	category.Children = children
	for i := range category.Children {
		populateChildren(&category.Children[i], byParent)
	}
}
