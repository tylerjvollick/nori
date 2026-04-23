package repositories

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

// RecipeFilter defines the optional filters for listing recipes.
type RecipeFilter struct {
	SpaceID    *uuid.UUID
	CategoryID *uuid.UUID
	Slug       string
	IsActive   *bool
	Offset     int
	Limit      int
}

type RecipeRepository struct {
	db *gorm.DB
}

func NewRecipeRepository(db *gorm.DB) *RecipeRepository {
	return &RecipeRepository{db: db}
}

// ---------------------------------------------------------------------------
// Recipe CRUD
// ---------------------------------------------------------------------------

func (r *RecipeRepository) Create(recipe *models.Recipe) error {
	return r.db.Create(recipe).Error
}

func (r *RecipeRepository) GetByID(id uuid.UUID) (*models.Recipe, error) {
	var recipe models.Recipe
	err := r.db.Preload("CurrentVersion").First(&recipe, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("recipe not found")
		}
		return nil, err
	}
	return &recipe, nil
}

func (r *RecipeRepository) GetBySlug(spaceID uuid.UUID, slug string) (*models.Recipe, error) {
	var recipe models.Recipe
	err := r.db.Preload("CurrentVersion").Where("space_id = ? AND slug = ?", spaceID, slug).First(&recipe).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("recipe not found")
		}
		return nil, err
	}
	return &recipe, nil
}

// List returns recipes matching the given filter along with the total count
// (before pagination). Offset and Limit on the filter control pagination.
func (r *RecipeRepository) List(filter RecipeFilter) ([]models.Recipe, int64, error) {
	query := r.db.Model(&models.Recipe{})

	if filter.SpaceID != nil {
		query = query.Where("space_id = ?", *filter.SpaceID)
	}
	if filter.CategoryID != nil {
		query = query.Where("category_id = ?", *filter.CategoryID)
	}
	if filter.Slug != "" {
		query = query.Where("slug = ?", filter.Slug)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var recipes []models.Recipe
	q := query.Preload("CurrentVersion").Order("name ASC")
	if filter.Offset > 0 {
		q = q.Offset(filter.Offset)
	}
	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	}
	if err := q.Find(&recipes).Error; err != nil {
		return nil, 0, err
	}

	return recipes, total, nil
}

func (r *RecipeRepository) Update(recipe *models.Recipe) error {
	return r.db.Save(recipe).Error
}

func (r *RecipeRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Recipe{}, "id = ?", id).Error
}

// ---------------------------------------------------------------------------
// RecipeVersion CRUD
// ---------------------------------------------------------------------------

func (r *RecipeRepository) CreateVersion(version *models.RecipeVersion) error {
	return r.db.Create(version).Error
}

func (r *RecipeRepository) UpdateVersion(version *models.RecipeVersion) error {
	return r.db.Save(version).Error
}

func (r *RecipeRepository) GetVersionByID(id int) (*models.RecipeVersion, error) {
	var version models.RecipeVersion
	err := r.db.First(&version, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("recipe version not found")
		}
		return nil, err
	}
	return &version, nil
}

// GetVersionWithTaskTree returns a recipe version with its root task preloaded.
// The full task tree (descendants) is loaded separately via the hierarchical ID
// prefix pattern and returned as a flat slice. Callers that need the tree
// structure can rebuild it from ParentID references.
func (r *RecipeRepository) GetVersionWithTaskTree(id int) (*models.RecipeVersion, []models.Task, error) {
	version, err := r.GetVersionByID(id)
	if err != nil {
		return nil, nil, err
	}

	if version.RootTaskID == nil {
		return version, nil, nil
	}

	// Load the root task.
	var rootTask models.Task
	if err := r.db.First(&rootTask, "id = ?", *version.RootTaskID).Error; err != nil {
		return nil, nil, fmt.Errorf("loading root task: %w", err)
	}
	version.RootTask = &rootTask

	// Load the full descendant tree using the hierarchical ID prefix.
	var tasks []models.Task
	if err := r.db.Where("id = ? OR id LIKE ?", rootTask.ID, rootTask.ID+".%").
		Order("display_order ASC, created_at ASC").
		Find(&tasks).Error; err != nil {
		return nil, nil, fmt.Errorf("loading task tree: %w", err)
	}

	return version, tasks, nil
}

// ListVersions returns all versions for a recipe, ordered by version number descending.
func (r *RecipeRepository) ListVersions(recipeID uuid.UUID) ([]models.RecipeVersion, error) {
	var versions []models.RecipeVersion
	err := r.db.Where("recipe_id = ?", recipeID).
		Order("version_number DESC").
		Find(&versions).Error
	return versions, err
}

// PublishVersion sets the version status to "published", records the published
// timestamp, archives the previously published version (if any), and updates
// the parent Recipe's CurrentVersionID — all within a single transaction.
func (r *RecipeRepository) PublishVersion(versionID int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var version models.RecipeVersion
		if err := tx.First(&version, "id = ?", versionID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("recipe version not found")
			}
			return err
		}

		// Archive the previously published version, if one exists.
		if err := tx.Model(&models.RecipeVersion{}).
			Where("recipe_id = ? AND status = ? AND id != ?",
				version.RecipeID,
				models.RecipeVersionStatusPublished,
				versionID).
			Update("status", models.RecipeVersionStatusArchived).Error; err != nil {
			return err
		}

		now := time.Now()
		version.Status = models.RecipeVersionStatusPublished
		version.PublishedAt = &now
		if err := tx.Save(&version).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Recipe{}).
			Where("id = ?", version.RecipeID).
			Update("current_version_id", version.ID).Error; err != nil {
			return err
		}

		return nil
	})
}

// ArchiveVersion sets the version status to "archived". If the archived
// version is the recipe's current version, the recipe's CurrentVersionID
// is cleared.
func (r *RecipeRepository) ArchiveVersion(versionID int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var version models.RecipeVersion
		if err := tx.First(&version, "id = ?", versionID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("recipe version not found")
			}
			return err
		}

		version.Status = models.RecipeVersionStatusArchived
		if err := tx.Save(&version).Error; err != nil {
			return err
		}

		// If this was the current version, clear the pointer.
		if err := tx.Model(&models.Recipe{}).
			Where("id = ? AND current_version_id = ?", version.RecipeID, version.ID).
			Update("current_version_id", nil).Error; err != nil {
			return err
		}

		return nil
	})
}
