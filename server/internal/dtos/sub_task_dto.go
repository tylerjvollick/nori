package dtos

import (
	"time"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
)

// CreateSubTaskRequest represents the request body for creating a sub-task.
type CreateSubTaskRequest struct {
	Title        string  `json:"title"`
	Description  *string `json:"description,omitempty"`
	DisplayOrder *int    `json:"displayOrder,omitempty"`
}

// UpdateSubTaskRequest represents the request body for updating a sub-task.
type UpdateSubTaskRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
}

// ReorderSubTasksRequest represents the request body for reordering sub-tasks.
type ReorderSubTasksRequest struct {
	// IDs is the ordered list of sub-task UUIDs. All sub-task IDs for the
	// parent task must be present; missing IDs are rejected.
	IDs []uuid.UUID `json:"ids"`
}

// SubTaskResponse represents a single sub-task in API responses.
type SubTaskResponse struct {
	ID           uuid.UUID  `json:"id"`
	TaskID       string     `json:"taskId"`
	Title        string     `json:"title"`
	Description  *string    `json:"description,omitempty"`
	DisplayOrder int        `json:"displayOrder"`
	CreatedAt    string     `json:"createdAt"`
	UpdatedAt    string     `json:"updatedAt"`
}

// SubTaskListResponse represents the ordered list of sub-tasks for a task.
type SubTaskListResponse struct {
	Items []SubTaskResponse `json:"items"`
	Total int               `json:"total"`
}

// SubTaskResponseFromModel converts a models.SubTask to a SubTaskResponse DTO.
func SubTaskResponseFromModel(s *models.SubTask) SubTaskResponse {
	return SubTaskResponse{
		ID:           s.ID,
		TaskID:       s.TaskID,
		Title:        s.Title,
		Description:  s.Description,
		DisplayOrder: s.DisplayOrder,
		CreatedAt:    s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    s.UpdatedAt.Format(time.RFC3339),
	}
}
