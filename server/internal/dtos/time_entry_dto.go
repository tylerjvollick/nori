package dtos

import (
	"time"

	"github.com/google/uuid"

	"github.com/tylerjvollick/nori/internal/models"
)

// CreateTimeEntryRequest is used for manual time entry creation.
type CreateTimeEntryRequest struct {
	StartedAt   time.Time  `json:"startedAt"`
	EndedAt     *time.Time `json:"endedAt,omitempty"`
	ElapsedSecs *int       `json:"elapsedSecs,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
}

// UpdateTimeEntryRequest allows editing an existing time entry.
type UpdateTimeEntryRequest struct {
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	EndedAt     *time.Time `json:"endedAt,omitempty"`
	ElapsedSecs *int       `json:"elapsedSecs,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
}

// TimeEntryResponse represents a time entry in API responses.
type TimeEntryResponse struct {
	ID          uuid.UUID  `json:"id"`
	TaskID      string     `json:"taskId"`
	SpaceID     uuid.UUID  `json:"spaceId"`
	LoggedByID  uuid.UUID  `json:"loggedById"`
	StartedAt   string     `json:"startedAt"`
	EndedAt     *string    `json:"endedAt,omitempty"`
	ElapsedSecs int        `json:"elapsedSecs"`
	IsPaused    bool       `json:"isPaused"`
	PausedAt    *string    `json:"pausedAt,omitempty"`
	ResumedAt   *string    `json:"resumedAt,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
	CreatedAt   string     `json:"createdAt"`
	UpdatedAt   string     `json:"updatedAt"`
}

// TimeEntryListResponse wraps a list of entries with a computed total.
type TimeEntryListResponse struct {
	Items             []TimeEntryResponse `json:"items"`
	TotalDurationSecs int                 `json:"totalDurationSecs"`
}

func TimeEntryResponseFromModel(e *models.TimeEntry) TimeEntryResponse {
	resp := TimeEntryResponse{
		ID:          e.ID,
		TaskID:      e.TaskID,
		SpaceID:     e.SpaceID,
		LoggedByID:  e.LoggedByID,
		ElapsedSecs: e.ElapsedSecs,
		IsPaused:    e.IsPaused,
		Notes:       e.Notes,
		StartedAt:   e.StartedAt.Format(time.RFC3339),
		CreatedAt:   e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   e.UpdatedAt.Format(time.RFC3339),
	}
	if e.EndedAt != nil {
		s := e.EndedAt.Format(time.RFC3339)
		resp.EndedAt = &s
	}
	if e.PausedAt != nil {
		s := e.PausedAt.Format(time.RFC3339)
		resp.PausedAt = &s
	}
	if e.ResumedAt != nil {
		s := e.ResumedAt.Format(time.RFC3339)
		resp.ResumedAt = &s
	}
	return resp
}
