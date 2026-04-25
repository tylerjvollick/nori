package dtos

import (
	"time"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
)

// CreateRecipeRequest represents the request body for creating a recipe.
type CreateRecipeRequest struct {
	Name        string     `json:"name" binding:"required"`
	Description *string    `json:"description,omitempty"`
	CategoryID  *uuid.UUID `json:"categoryId,omitempty"`
}

// UpdateRecipeRequest represents the request body for updating a recipe.
type UpdateRecipeRequest struct {
	CategoryID *uuid.UUID `json:"categoryId,omitempty"`
	IsActive   *bool      `json:"isActive,omitempty"`
}

// CreateRecipeVersionRequest represents the request body for creating a recipe version.
// For TOML-based recipes, Content is required. For task-tree recipes, Content is
// omitted and the draft is created by cloning from the published version.
type CreateRecipeVersionRequest struct {
	Content       *string `json:"content,omitempty"`
	ChangeSummary *string `json:"changeSummary,omitempty"`
}

// PourRecipeRequest represents the request body for pouring a recipe into a task graph.
type PourRecipeRequest struct {
	Vars    map[string]string `json:"vars,omitempty"`
	OrderID *uuid.UUID        `json:"orderId,omitempty"`
}

// RollRecipeRequest represents the request body for rolling a recipe into a job.
type RollRecipeRequest struct {
	OrderQty   *int       `json:"order_qty,omitempty"`
	CustomerID *uuid.UUID `json:"customer_id,omitempty"`
	DueDate    *time.Time `json:"due_date,omitempty"`
	Title      *string    `json:"title,omitempty"`
}

// SaveAsRecipeRequest represents the request body for saving a job as a recipe.
type SaveAsRecipeRequest struct {
	Name                        string  `json:"name" binding:"required"`
	Description                 *string `json:"description,omitempty"`
	BackfillEstimatedFromActual *bool   `json:"backfillEstimatedFromActual,omitempty"`
}

// RecipeResponse represents a single recipe in API responses.
type RecipeResponse struct {
	ID               uuid.UUID              `json:"id"`
	SpaceID          uuid.UUID              `json:"spaceId"`
	Name             string                 `json:"name"`
	Slug             string                 `json:"slug"`
	Description      *string                `json:"description,omitempty"`
	CategoryID       *uuid.UUID             `json:"categoryId,omitempty"`
	CurrentVersionID *int                   `json:"currentVersionId,omitempty"`
	ExtendsRecipeID  *uuid.UUID             `json:"extendsRecipeId,omitempty"`
	CreatedByID      uuid.UUID              `json:"createdById"`
	IsActive         bool                   `json:"isActive"`
	CreatedAt        string                 `json:"createdAt"`
	UpdatedAt        string                 `json:"updatedAt"`
	CurrentVersion   *RecipeVersionResponse `json:"currentVersion,omitempty"`
}

// RecipeListResponse represents a paginated list of recipes.
type RecipeListResponse struct {
	Items  []RecipeResponse `json:"items"`
	Total  int64            `json:"total"`
	Offset int              `json:"offset"`
	Limit  int              `json:"limit"`
}

// RecipeVersionResponse represents a single recipe version in API responses.
type RecipeVersionResponse struct {
	ID            int                        `json:"id"`
	RecipeID      uuid.UUID                  `json:"recipeId"`
	VersionNumber int                        `json:"versionNumber"`
	Status        models.RecipeVersionStatus `json:"status"`
	Content       *string                    `json:"content,omitempty"`
	RootTaskID    *string                     `json:"rootTaskId,omitempty"`
	ChangeSummary *string                    `json:"changeSummary,omitempty"`
	AuthorID      uuid.UUID                  `json:"authorId"`
	PublishedAt   *time.Time                 `json:"publishedAt,omitempty"`
	TaskTree      *TaskTreeResponse          `json:"taskTree,omitempty"`
	CreatedAt     string                     `json:"createdAt"`
	UpdatedAt     string                     `json:"updatedAt"`
}

// RecipeResponseFromModel converts a models.Recipe to a RecipeResponse DTO.
// Name and Description are derived from the current version's root task.
// If no current version or root task exists, Name falls back to the recipe slug.
func RecipeResponseFromModel(r *models.Recipe) RecipeResponse {
	resp := RecipeResponse{
		ID:               r.ID,
		SpaceID:          r.SpaceID,
		Name:             r.Slug,
		Slug:             r.Slug,
		CategoryID:       r.CategoryID,
		CurrentVersionID: r.CurrentVersionID,
		ExtendsRecipeID:  r.ExtendsRecipeID,
		CreatedByID:      r.CreatedByID,
		IsActive:         r.IsActive,
		CreatedAt:        r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if r.CurrentVersion != nil {
		cv := RecipeVersionResponseFromModel(r.CurrentVersion)
		resp.CurrentVersion = &cv

		if r.CurrentVersion.RootTask != nil {
			resp.Name = r.CurrentVersion.RootTask.Title
			resp.Description = r.CurrentVersion.RootTask.Description
		}
	}

	return resp
}

// RecipeVersionResponseWithTree converts a RecipeVersion and its task tree into a
// RecipeVersionResponse with the TaskTree field populated.
func RecipeVersionResponseWithTree(v *models.RecipeVersion, tasks []models.Task) RecipeVersionResponse {
	resp := RecipeVersionResponseFromModel(v)
	if v.RootTaskID != nil && len(tasks) > 0 {
		// Find the root task in the tasks slice.
		rootID := *v.RootTaskID
		for i := range tasks {
			if tasks[i].ID == rootID {
				tree := BuildTaskTree(&tasks[i], tasks)
				resp.TaskTree = &tree
				break
			}
		}
	}
	return resp
}

// RecipeVersionResponseFromModel converts a models.RecipeVersion to a RecipeVersionResponse DTO.
func RecipeVersionResponseFromModel(v *models.RecipeVersion) RecipeVersionResponse {
	return RecipeVersionResponse{
		ID:            v.ID,
		RecipeID:      v.RecipeID,
		VersionNumber: v.VersionNumber,
		Status:        v.Status,
		Content:       v.Content,
		RootTaskID:    v.RootTaskID,
		ChangeSummary: v.ChangeSummary,
		AuthorID:      v.AuthorID,
		PublishedAt:   v.PublishedAt,
		CreatedAt:     v.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     v.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
