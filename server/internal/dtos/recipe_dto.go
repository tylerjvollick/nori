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
	Name        *string    `json:"name,omitempty"`
	Description *string    `json:"description,omitempty"`
	CategoryID  *uuid.UUID `json:"categoryId,omitempty"`
	IsActive    *bool      `json:"isActive,omitempty"`
}

// CreateRecipeVersionRequest represents the request body for creating a recipe version.
type CreateRecipeVersionRequest struct {
	Content       string  `json:"content" binding:"required"`
	ChangeSummary *string `json:"changeSummary,omitempty"`
}

// PourRecipeRequest represents the request body for pouring a recipe into a task graph.
type PourRecipeRequest struct {
	Vars    map[string]string `json:"vars,omitempty"`
	OrderID *uuid.UUID        `json:"orderId,omitempty"`
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
	RootTaskID    *uuid.UUID                 `json:"rootTaskId,omitempty"`
	ChangeSummary *string                    `json:"changeSummary,omitempty"`
	AuthorID      uuid.UUID                  `json:"authorId"`
	PublishedAt   *time.Time                 `json:"publishedAt,omitempty"`
	CreatedAt     string                     `json:"createdAt"`
	UpdatedAt     string                     `json:"updatedAt"`
}

// RecipeResponseFromModel converts a models.Recipe to a RecipeResponse DTO.
func RecipeResponseFromModel(r *models.Recipe) RecipeResponse {
	resp := RecipeResponse{
		ID:               r.ID,
		SpaceID:          r.SpaceID,
		Name:             r.Name,
		Slug:             r.Slug,
		Description:      r.Description,
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
