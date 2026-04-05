package models

import (
	"time"

	"github.com/google/uuid"
)

// ActivityEntryType represents the kind of activity logged on a task.
type ActivityEntryType string

const (
	ActivityEntryTypeStatusChange     ActivityEntryType = "status_change"
	ActivityEntryTypeStepStarted      ActivityEntryType = "step_started"
	ActivityEntryTypeStepCompleted    ActivityEntryType = "step_completed"
	ActivityEntryTypeStepPaused       ActivityEntryType = "step_paused"
	ActivityEntryTypeStepResumed      ActivityEntryType = "step_resumed"
	ActivityEntryTypeStepSkipped      ActivityEntryType = "step_skipped"
	ActivityEntryTypeComment          ActivityEntryType = "comment"
	ActivityEntryTypeInterruption     ActivityEntryType = "interruption"
	ActivityEntryTypeAssignmentChange ActivityEntryType = "assignment_change"
	ActivityEntryTypeLinkAdded        ActivityEntryType = "link_added"
	ActivityEntryTypeRecipeEdited     ActivityEntryType = "recipe_edited"
	ActivityEntryTypeCostLogged       ActivityEntryType = "cost_logged"
	ActivityEntryTypeTaskCreated      ActivityEntryType = "task_created"
)

// ActivityEntry records a chronological log entry on a task. This is the
// "activity tab" — every status change, step transition, comment, and
// interruption creates an ActivityEntry.
type ActivityEntry struct {
	ID              uuid.UUID         `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TaskID          string            `gorm:"type:varchar(255);not null" json:"taskId"`
	UserID          uuid.UUID         `gorm:"type:uuid;not null" json:"userId"`
	EntryType       ActivityEntryType `gorm:"type:varchar(30);not null" json:"entryType"`
	Description     string            `gorm:"type:text;not null" json:"description"`
	LinkedTaskID    *string           `gorm:"type:varchar(255)" json:"linkedTaskId,omitempty"`
	StepTaskID      *string           `gorm:"type:varchar(255)" json:"stepTaskId,omitempty"`
	DurationSeconds *int              `json:"durationSeconds,omitempty"`
	OldValue        *string           `gorm:"type:text" json:"oldValue,omitempty"`
	NewValue        *string           `gorm:"type:text" json:"newValue,omitempty"`
	Metadata        JSONB             `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt       time.Time         `gorm:"default:now()" json:"createdAt"`

	// Relations
	Task       *Task `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	User       *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	LinkedTask *Task `gorm:"foreignKey:LinkedTaskID" json:"linkedTask,omitempty"`
	StepTask   *Task `gorm:"foreignKey:StepTaskID" json:"stepTask,omitempty"`
}

func (ActivityEntry) TableName() string {
	return "activity_entry"
}
