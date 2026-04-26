package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
)

// TimeEntryRepoInterface defines the methods needed from the time entry repository.
type TimeEntryRepoInterface interface {
	Create(entry *models.TimeEntry) error
	GetByID(id uuid.UUID) (*models.TimeEntry, error)
	Update(entry *models.TimeEntry) error
	Delete(id uuid.UUID) error
	GetByTaskID(taskID string) ([]models.TimeEntry, error)
	GetRunningEntry(taskID string) (*models.TimeEntry, error)
	GetCompletedTasksWithNoEntries(spaceID uuid.UUID) ([]models.Task, error)
}

type TimeEntryService struct {
	timeEntryRepo TimeEntryRepoInterface
	taskRepo      TaskRepositoryInterface
}

func NewTimeEntryService(timeEntryRepo TimeEntryRepoInterface, taskRepo TaskRepositoryInterface) *TimeEntryService {
	return &TimeEntryService{timeEntryRepo: timeEntryRepo, taskRepo: taskRepo}
}

// StartTimer creates a new running time entry on a task.
// Returns 409-style error if a timer is already running.
func (s *TimeEntryService) StartTimer(taskID string, spaceID uuid.UUID, userID uuid.UUID) (*models.TimeEntry, error) {
	// Verify task exists.
	if _, err := s.taskRepo.GetByID(taskID); err != nil {
		return nil, fmt.Errorf("task %q not found: %w", taskID, err)
	}

	// Check for running timer.
	running, err := s.timeEntryRepo.GetRunningEntry(taskID)
	if err != nil {
		return nil, fmt.Errorf("checking running timer: %w", err)
	}
	if running != nil {
		return nil, ErrTimerAlreadyRunning
	}

	now := time.Now()
	entry := &models.TimeEntry{
		ID:         uuid.New(),
		TaskID:     taskID,
		SpaceID:    spaceID,
		LoggedByID: userID,
		StartedAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.timeEntryRepo.Create(entry); err != nil {
		return nil, fmt.Errorf("creating time entry: %w", err)
	}

	return entry, nil
}

// PauseTimer stops the running timer on a task.
// Returns 409-style error if no timer is running.
func (s *TimeEntryService) PauseTimer(taskID string) (*models.TimeEntry, error) {
	running, err := s.timeEntryRepo.GetRunningEntry(taskID)
	if err != nil {
		return nil, fmt.Errorf("checking running timer: %w", err)
	}
	if running == nil {
		return nil, ErrNoTimerRunning
	}

	now := time.Now()
	running.EndedAt = &now
	dur := int(now.Sub(running.StartedAt).Seconds())
	running.DurationSecs = &dur
	running.UpdatedAt = now

	if err := s.timeEntryRepo.Update(running); err != nil {
		return nil, fmt.Errorf("pausing time entry: %w", err)
	}

	return running, nil
}

// StopRunningTimer pauses any running timer on a task without error if none is running.
// Used by CompleteTask to auto-stop timers.
func (s *TimeEntryService) StopRunningTimer(taskID string) error {
	running, err := s.timeEntryRepo.GetRunningEntry(taskID)
	if err != nil {
		return fmt.Errorf("checking running timer: %w", err)
	}
	if running == nil {
		return nil // no timer running — not an error
	}

	now := time.Now()
	running.EndedAt = &now
	dur := int(now.Sub(running.StartedAt).Seconds())
	running.DurationSecs = &dur
	running.UpdatedAt = now

	return s.timeEntryRepo.Update(running)
}

// CreateManualEntry creates a time entry with explicit start/end times.
func (s *TimeEntryService) CreateManualEntry(taskID string, spaceID uuid.UUID, userID uuid.UUID, dto *dtos.CreateTimeEntryRequest) (*models.TimeEntry, error) {
	if _, err := s.taskRepo.GetByID(taskID); err != nil {
		return nil, fmt.Errorf("task %q not found: %w", taskID, err)
	}

	now := time.Now()
	entry := &models.TimeEntry{
		ID:         uuid.New(),
		TaskID:     taskID,
		SpaceID:    spaceID,
		LoggedByID: userID,
		StartedAt:  dto.StartedAt,
		EndedAt:    dto.EndedAt,
		Notes:      dto.Notes,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if dto.EndedAt != nil {
		dur := int(dto.EndedAt.Sub(dto.StartedAt).Seconds())
		entry.DurationSecs = &dur
	}

	if err := s.timeEntryRepo.Create(entry); err != nil {
		return nil, fmt.Errorf("creating time entry: %w", err)
	}

	return entry, nil
}

// UpdateEntry updates fields on an existing time entry.
func (s *TimeEntryService) UpdateEntry(id uuid.UUID, dto *dtos.UpdateTimeEntryRequest) (*models.TimeEntry, error) {
	entry, err := s.timeEntryRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("time entry %q not found: %w", id, err)
	}

	if dto.StartedAt != nil {
		entry.StartedAt = *dto.StartedAt
	}
	if dto.EndedAt != nil {
		entry.EndedAt = dto.EndedAt
	}
	if dto.DurationSecs != nil {
		entry.DurationSecs = dto.DurationSecs
	}
	if dto.Notes != nil {
		entry.Notes = dto.Notes
	}
	entry.UpdatedAt = time.Now()

	if err := s.timeEntryRepo.Update(entry); err != nil {
		return nil, fmt.Errorf("updating time entry: %w", err)
	}

	return entry, nil
}

// DeleteEntry removes a time entry.
func (s *TimeEntryService) DeleteEntry(id uuid.UUID) error {
	return s.timeEntryRepo.Delete(id)
}

// ListEntries returns all time entries for a task with a total duration.
func (s *TimeEntryService) ListEntries(taskID string) ([]models.TimeEntry, int, error) {
	entries, err := s.timeEntryRepo.GetByTaskID(taskID)
	if err != nil {
		return nil, 0, fmt.Errorf("listing time entries: %w", err)
	}

	total := 0
	for _, e := range entries {
		if e.DurationSecs != nil {
			total += *e.DurationSecs
		}
	}

	return entries, total, nil
}

// GetUnloggedTasks returns completed tasks with zero time entries.
func (s *TimeEntryService) GetUnloggedTasks(spaceID uuid.UUID) ([]models.Task, error) {
	return s.timeEntryRepo.GetCompletedTasksWithNoEntries(spaceID)
}

// Sentinel errors for conflict responses.
var (
	ErrTimerAlreadyRunning = fmt.Errorf("a timer is already running on this task")
	ErrNoTimerRunning      = fmt.Errorf("no timer is running on this task")
)
