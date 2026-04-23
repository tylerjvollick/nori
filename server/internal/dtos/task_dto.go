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
	Title        *string            `json:"title,omitempty"`
	Description  *string            `json:"description,omitempty"`
	StationID    *uuid.UUID         `json:"stationId,omitempty"`
	Priority     *int               `json:"priority,omitempty"`
	Status       *models.TaskStatus `json:"status,omitempty"`
	AssignedToID *string            `json:"assignedToId,omitempty"` // UUID string or empty string to unassign
}

// AddChildTaskRequest represents the request body for adding a child task.
type AddChildTaskRequest struct {
	Title       string          `json:"title"`
	Description *string         `json:"description,omitempty"`
	Type        models.TaskType `json:"type,omitempty"`
	StationID   *uuid.UUID      `json:"stationId,omitempty"`
	AfterID     *string         `json:"afterId,omitempty"` // Task ID to add a dependency on
}

// CompleteTaskRequest represents the optional request body for completing a task.
// All fields are optional — if no body is sent, defaults apply.
type CompleteTaskRequest struct {
	ActualTimeSecs *int `json:"actualTimeSeconds,omitempty"` // Override the logged time (in seconds)
}

// CompleteTaskResponse extends the standard task response with navigation hints.
type CompleteTaskResponse struct {
	TaskResponse
	NextTaskId    *string `json:"nextTaskId,omitempty"`    // First downstream task unblocked by this completion
	NextTaskTitle *string `json:"nextTaskTitle,omitempty"` // Title of the next task for display
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
	Quantity        int               `json:"quantity"`
	Priority        int               `json:"priority"`
	DisplayOrder    int               `json:"displayOrder"`
	DueDate         *time.Time        `json:"dueDate,omitempty"`
	StartedAt       *time.Time        `json:"startedAt,omitempty"`
	PausedAt        *time.Time        `json:"pausedAt,omitempty"`
	CompletedAt     *time.Time        `json:"completedAt,omitempty"`
	ActualTimeSecs     int               `json:"actualTimeSeconds"`
	EstimatedTimeSecs  *int              `json:"estimatedTimeSeconds,omitempty"`
	DeviationNotes     *string           `json:"deviationNotes,omitempty"`
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

// TaskTreeResponse extends TaskResponse with a nested children array for
// the recursive task tree endpoint.
type TaskTreeResponse struct {
	TaskResponse
	Children []TaskTreeResponse `json:"children"`
}

// BuildTaskTree takes a root task and all its descendants (flat list) and
// assembles them into a nested TaskTreeResponse. The descendants slice should
// include the root task itself. This is O(n) where n = len(descendants).
func BuildTaskTree(root *models.Task, descendants []models.Task) TaskTreeResponse {
	// Build a map of parentID -> children
	childrenMap := make(map[string][]models.Task)
	for i := range descendants {
		t := &descendants[i]
		if t.ParentID != nil {
			childrenMap[*t.ParentID] = append(childrenMap[*t.ParentID], *t)
		}
	}

	// Recursively build the tree from the root
	return buildTreeNode(root, childrenMap)
}

// buildTreeNode recursively converts a task and its descendants into a
// TaskTreeResponse using the precomputed children map.
func buildTreeNode(task *models.Task, childrenMap map[string][]models.Task) TaskTreeResponse {
	node := TaskTreeResponse{
		TaskResponse: TaskResponseFromModel(task),
		Children:     []TaskTreeResponse{}, // always non-nil for consistent JSON
	}

	children := childrenMap[task.ID]
	for i := range children {
		node.Children = append(node.Children, buildTreeNode(&children[i], childrenMap))
	}

	return node
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
		Quantity:        t.Quantity,
		Priority:        t.Priority,
		DisplayOrder:    t.DisplayOrder,
		DueDate:         t.DueDate,
		StartedAt:       t.StartedAt,
		PausedAt:        t.PausedAt,
		CompletedAt:     t.CompletedAt,
		ActualTimeSecs:     t.ActualTimeSecs,
		EstimatedTimeSecs:  t.EstimatedTimeSecs,
		DeviationNotes:     t.DeviationNotes,
		Metadata:        t.Metadata,
		CreatedAt:       t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
