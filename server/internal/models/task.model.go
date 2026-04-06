package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TaskType represents the kind of work a task represents.
type TaskType string

const (
	TaskTypeJob       TaskType = "job"
	TaskTypeTask      TaskType = "task"
	TaskTypeMilestone TaskType = "milestone"
	TaskTypeGate      TaskType = "gate"
)

// TaskStatus represents the current state of a task.
type TaskStatus string

const (
	TaskStatusOpen      TaskStatus = "open"
	TaskStatusActive    TaskStatus = "active"
	TaskStatusPaused    TaskStatus = "paused"
	TaskStatusDone      TaskStatus = "done"
	TaskStatusSkipped   TaskStatus = "skipped"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// JSONB is a helper type for storing arbitrary JSON data in PostgreSQL jsonb columns.
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface for GORM.
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for GORM.
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprintf("JSONB.Scan: expected []byte, got %T", value))
	}
	return json.Unmarshal(bytes, j)
}

// Task is the universal work item in Nori. A root task with no parent is a
// Job. Children are Tasks. Uses hierarchical string IDs (e.g., "nori-a3f8.1.2").
type Task struct {
	ID              string     `gorm:"type:varchar(255);primaryKey" json:"id"`
	SpaceID         uuid.UUID  `gorm:"type:uuid;not null" json:"spaceId"`
	ParentID        *string    `gorm:"type:varchar(255)" json:"parentId,omitempty"`
	RecipeID        *uuid.UUID `gorm:"type:uuid" json:"recipeId,omitempty"`
	RecipeVersionID *int       `json:"recipeVersionId,omitempty"`
	StationID       *uuid.UUID `gorm:"type:uuid" json:"stationId,omitempty"`
	CustomerID      *uuid.UUID `gorm:"type:uuid" json:"customerId,omitempty"`
	AssignedToID    *uuid.UUID `gorm:"type:uuid" json:"assignedToId,omitempty"`
	CreatedByID     uuid.UUID  `gorm:"type:uuid;not null" json:"createdById"`
	Type            TaskType   `gorm:"type:varchar(50);not null;default:'task'" json:"type"`
	Status          TaskStatus `gorm:"type:varchar(50);not null;default:'open'" json:"status"`
	Title           string     `gorm:"not null" json:"title"`
	Description     *string    `json:"description,omitempty"`
	Quantity        int        `gorm:"not null;default:1" json:"quantity"`
	Priority        int        `gorm:"not null;default:0" json:"priority"`
	DisplayOrder    int        `gorm:"not null;default:0" json:"displayOrder"`
	DueDate         *time.Time `json:"dueDate,omitempty"`
	StartedAt       *time.Time `json:"startedAt,omitempty"`
	PausedAt        *time.Time `json:"pausedAt,omitempty"`
	CompletedAt     *time.Time `json:"completedAt,omitempty"`
	ActualTimeSecs  int        `gorm:"not null;default:0" json:"actualTimeSeconds"`
	DeviationNotes  *string    `json:"deviationNotes,omitempty"`
	Metadata        JSONB      `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt       time.Time  `gorm:"default:now()" json:"createdAt"`
	UpdatedAt       time.Time  `gorm:"default:now()" json:"updatedAt"`

	// Relations
	Space      *Space    `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
	Parent     *Task     `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children   []Task    `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Station    *Station  `gorm:"foreignKey:StationID" json:"station,omitempty"`
	Customer   *Customer `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	AssignedTo *User     `gorm:"foreignKey:AssignedToID" json:"assignedTo,omitempty"`
	CreatedBy  *User     `gorm:"foreignKey:CreatedByID" json:"createdBy,omitempty"`
	Tags       []Tag     `gorm:"many2many:task_tag;joinForeignKey:TaskID;joinReferences:TagID" json:"tags,omitempty"`
}

func (Task) TableName() string {
	return "task"
}
