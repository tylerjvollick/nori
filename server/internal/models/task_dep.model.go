package models

import (
	"time"

	"github.com/google/uuid"
)

// DepType represents the kind of dependency between two tasks.
type DepType string

const (
	DepTypeBlocks   DepType = "blocks"
	DepTypeWaitsFor DepType = "waits_for"
	DepTypeRelated  DepType = "related"
)

// TaskDep represents a directed dependency edge between two tasks.
// FromTaskID is the task that has the dependency; ToTaskID is the task
// being depended on. For example, if Task A blocks Task B, then
// FromTaskID=B, ToTaskID=A, Type=blocks.
type TaskDep struct {
	ID         uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	FromTaskID string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_task_dep_from_to" json:"fromTaskId"`
	ToTaskID   string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_task_dep_from_to" json:"toTaskId"`
	Type       DepType   `gorm:"type:varchar(50);not null" json:"type"`
	CreatedAt  time.Time `gorm:"default:now()" json:"createdAt"`

	// Relations
	FromTask *Task `gorm:"foreignKey:FromTaskID" json:"fromTask,omitempty"`
	ToTask   *Task `gorm:"foreignKey:ToTaskID" json:"toTask,omitempty"`
}

func (TaskDep) TableName() string {
	return "task_dep"
}
