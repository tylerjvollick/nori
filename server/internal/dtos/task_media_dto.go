package dtos

import (
	"time"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
)

// TaskMediaResponse represents a single media attachment in API responses.
type TaskMediaResponse struct {
	ID           uuid.UUID  `json:"id"`
	TaskID       string     `json:"taskId"`
	URL          string     `json:"url"`
	FileName     string     `json:"fileName"`
	MimeType     string     `json:"mimeType"`
	FileSize     int64      `json:"fileSize"`
	Duration     *int       `json:"duration,omitempty"`
	DisplayOrder int        `json:"displayOrder"`
	CapturedByID *uuid.UUID `json:"capturedById,omitempty"`
	CreatedAt    string     `json:"createdAt"`
}

// TaskMediaListResponse represents the ordered list of media for a task.
type TaskMediaListResponse struct {
	Items []TaskMediaResponse `json:"items"`
	Total int                 `json:"total"`
}

// TaskMediaResponseFromModel converts a models.TaskMedia to a TaskMediaResponse DTO.
func TaskMediaResponseFromModel(m *models.TaskMedia) TaskMediaResponse {
	return TaskMediaResponse{
		ID:           m.ID,
		TaskID:       m.TaskID,
		URL:          m.FilePath,
		FileName:     m.FileName,
		MimeType:     m.MimeType,
		FileSize:     m.FileSize,
		Duration:     m.Duration,
		DisplayOrder: m.DisplayOrder,
		CapturedByID: m.CapturedByID,
		CreatedAt:    m.CreatedAt.Format(time.RFC3339),
	}
}
