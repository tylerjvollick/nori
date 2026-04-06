package dtos

import (
	"time"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
)

// TaskDepResponse is the JSON representation of a task dependency edge.
type TaskDepResponse struct {
	ID         uuid.UUID      `json:"id"`
	FromTaskID string         `json:"fromTaskId"`
	ToTaskID   string         `json:"toTaskId"`
	Type       models.DepType `json:"type"`
	CreatedAt  time.Time      `json:"createdAt"`
}

// TaskDepResponseFromModel converts a models.TaskDep to a TaskDepResponse DTO.
func TaskDepResponseFromModel(d *models.TaskDep) TaskDepResponse {
	return TaskDepResponse{
		ID:         d.ID,
		FromTaskID: d.FromTaskID,
		ToTaskID:   d.ToTaskID,
		Type:       d.Type,
		CreatedAt:  d.CreatedAt,
	}
}

// TaskDepsResponse wraps the two-direction dependency query result.
type TaskDepsResponse struct {
	Blockers   []TaskDepResponse `json:"blockers"`
	Dependents []TaskDepResponse `json:"dependents"`
}

// AddDepRequest is the JSON body for POST /api/v1/tasks/:id/deps.
type AddDepRequest struct {
	TargetTaskID string         `json:"targetTaskId"`
	Type         models.DepType `json:"type"`
}
