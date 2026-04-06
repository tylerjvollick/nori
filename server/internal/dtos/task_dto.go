package dtos

import (
	"time"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
)

// CreateTaskRequest represents the request body for creating a task.
type CreateTaskRequest struct {
	Title       string          `json:"title" binding:"required"`
	Description *string         `json:"description,omitempty"`
	Type        models.TaskType `json:"type" binding:"required"`
	ParentID    *string         `json:"parentId,omitempty"`
	StationID   *uuid.UUID      `json:"stationId,omitempty"`
	Priority    *int            `json:"priority,omitempty"`
}

// UpdateTaskRequest represents the request body for updating a task.
type UpdateTaskRequest struct {
	Title       *string            `json:"title,omitempty"`
	Description *string            `json:"description,omitempty"`
	StationID   *uuid.UUID         `json:"stationId,omitempty"`
	Priority    *int               `json:"priority,omitempty"`
	Status      *models.TaskStatus `json:"status,omitempty"`
}

// AddChildTaskRequest represents the request body for adding a child task.
type AddChildTaskRequest struct {
	Title       string          `json:"title"`
	Description *string         `json:"description,omitempty"`
	Type        models.TaskType `json:"type,omitempty"`
	StationID   *uuid.UUID      `json:"stationId,omitempty"`
	AfterID     *string         `json:"afterId,omitempty"` // Task ID to add a dependency on
}

// AddNoteRequest represents the request body for adding a deviation note.
type AddNoteRequest struct {
	Text string `json:"text"`
}

// TaskResponse represents a single task in API responses.
type TaskResponse struct {
	ID              string            `json:"id"`
	SpaceID         uuid.UUID         `json:"spaceId"`
	ParentID        *string           `json:"parentId,omitempty"`
	RecipeID        *uuid.UUID        `json:"recipeId,omitempty"`
	RecipeVersionID *int              `json:"recipeVersionId,omitempty"`
	StationID       *uuid.UUID        `json:"stationId,omitempty"`
	CustomerID      *uuid.UUID        `json:"customerId,omitempty"`
	AssignedToID    *uuid.UUID        `json:"assignedToId,omitempty"`
	CreatedByID     uuid.UUID         `json:"createdById"`
	Type            models.TaskType   `json:"type"`
	Status          models.TaskStatus `json:"status"`
	Title           string            `json:"title"`
	Description     *string           `json:"description,omitempty"`
	Priority        int               `json:"priority"`
	DisplayOrder    int               `json:"displayOrder"`
	DueDate         *time.Time        `json:"dueDate,omitempty"`
	StartedAt       *time.Time        `json:"startedAt,omitempty"`
	PausedAt        *time.Time        `json:"pausedAt,omitempty"`
	CompletedAt     *time.Time        `json:"completedAt,omitempty"`
	ActualTimeSecs  int               `json:"actualTimeSeconds"`
	DeviationNotes  *string           `json:"deviationNotes,omitempty"`
	Metadata        models.JSONB      `json:"metadata,omitempty"`
	CreatedAt       string            `json:"createdAt"`
	UpdatedAt       string            `json:"updatedAt"`
}

// TaskListResponse represents a paginated list of tasks.
type TaskListResponse struct {
	Items  []TaskResponse `json:"items"`
	Total  int64          `json:"total"`
	Offset int            `json:"offset"`
	Limit  int            `json:"limit"`
}

// TaskResponseFromModel converts a models.Task to a TaskResponse DTO.
func TaskResponseFromModel(t *models.Task) TaskResponse {
	return TaskResponse{
		ID:              t.ID,
		SpaceID:         t.SpaceID,
		ParentID:        t.ParentID,
		RecipeID:        t.RecipeID,
		RecipeVersionID: t.RecipeVersionID,
		StationID:       t.StationID,
		CustomerID:      t.CustomerID,
		AssignedToID:    t.AssignedToID,
		CreatedByID:     t.CreatedByID,
		Type:            t.Type,
		Status:          t.Status,
		Title:           t.Title,
		Description:     t.Description,
		Priority:        t.Priority,
		DisplayOrder:    t.DisplayOrder,
		DueDate:         t.DueDate,
		StartedAt:       t.StartedAt,
		PausedAt:        t.PausedAt,
		CompletedAt:     t.CompletedAt,
		ActualTimeSecs:  t.ActualTimeSecs,
		DeviationNotes:  t.DeviationNotes,
		Metadata:        t.Metadata,
		CreatedAt:       t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
