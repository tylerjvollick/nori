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
// Returns 409-style error if a timer is already running (not paused).
func (s *TimeEntryService) StartTimer(taskID string, spaceID uuid.UUID, userID uuid.UUID) (*models.TimeEntry, error) {
	if _, err := s.taskRepo.GetByID(taskID); err != nil {
		return nil, fmt.Errorf("task %q not found: %w", taskID, err)
	}

	// Check for active (non-ended) timer.
	running, err := s.timeEntryRepo.GetRunningEntry(taskID)
	if err != nil {
		return nil, fmt.Errorf("checking running timer: %w", err)
	}
	if running != nil {
		return nil, ErrTimerAlreadyRunning
	}

	now := time.Now()
	entry := &models.TimeEntry{
		ID:          uuid.New(),
		TaskID:      taskID,
		SpaceID:     spaceID,
		LoggedByID:  userID,
		StartedAt:   now,
		ElapsedSecs: 0,
		IsPaused:    false,
		ResumedAt:   &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.timeEntryRepo.Create(entry); err != nil {
		return nil, fmt.Errorf("creating time entry: %w", err)
	}

	return entry, nil
}

// PauseTimer pauses the running timer on a task, accumulating elapsed time.
// Returns 409-style error if no timer is running.
func (s *TimeEntryService) PauseTimer(taskID string) (*models.TimeEntry, error) {
	running, err := s.timeEntryRepo.GetRunningEntry(taskID)
	if err != nil {
		return nil, fmt.Errorf("checking running timer: %w", err)
	}
	if running == nil {
		return nil, ErrNoTimerRunning
	}
	if running.IsPaused {
		return running, nil // already paused, no-op
	}

	now := time.Now()
	// Accumulate elapsed: add time since last resume (or start).
	activeStart := running.StartedAt
	if running.ResumedAt != nil {
		activeStart = *running.ResumedAt
	}
	running.ElapsedSecs += int(now.Sub(activeStart).Seconds())
	running.IsPaused = true
	running.PausedAt = &now
	running.UpdatedAt = now

	if err := s.timeEntryRepo.Update(running); err != nil {
		return nil, fmt.Errorf("pausing time entry: %w", err)
	}

	return running, nil
}

// ResumeTimer resumes a paused timer.
func (s *TimeEntryService) ResumeTimer(taskID string) (*models.TimeEntry, error) {
	running, err := s.timeEntryRepo.GetRunningEntry(taskID)
	if err != nil {
		return nil, fmt.Errorf("checking running timer: %w", err)
	}
	if running == nil {
		return nil, ErrNoTimerRunning
	}
	if !running.IsPaused {
		return running, nil // already running, no-op
	}

	now := time.Now()
	running.IsPaused = false
	running.PausedAt = nil
	running.ResumedAt = &now
	running.UpdatedAt = now

	if err := s.timeEntryRepo.Update(running); err != nil {
		return nil, fmt.Errorf("resuming time entry: %w", err)
	}

	return running, nil
}

// StopTimer finalizes the running timer (accumulates elapsed if running, sets ended_at).
func (s *TimeEntryService) StopTimer(taskID string) (*models.TimeEntry, error) {
	running, err := s.timeEntryRepo.GetRunningEntry(taskID)
	if err != nil {
		return nil, fmt.Errorf("checking running timer: %w", err)
	}
	if running == nil {
		return nil, ErrNoTimerRunning
	}

	now := time.Now()

	// If timer is actively running (not paused), accumulate remaining time.
	if !running.IsPaused {
		activeStart := running.StartedAt
		if running.ResumedAt != nil {
			activeStart = *running.ResumedAt
		}
		running.ElapsedSecs += int(now.Sub(activeStart).Seconds())
	}

	running.EndedAt = &now
	running.IsPaused = false
	running.PausedAt = nil
	running.UpdatedAt = now

	if err := s.timeEntryRepo.Update(running); err != nil {
		return nil, fmt.Errorf("stopping time entry: %w", err)
	}

	return running, nil
}

// StopRunningTimer stops any running timer on a task without error if none is running.
// Used by CompleteTask to auto-stop timers.
func (s *TimeEntryService) StopRunningTimer(taskID string) error {
	running, err := s.timeEntryRepo.GetRunningEntry(taskID)
	if err != nil {
		return fmt.Errorf("checking running timer: %w", err)
	}
	if running == nil {
		return nil
	}

	_, err = s.StopTimer(taskID)
	return err
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

	// Calculate elapsed from start/end if both provided.
	if dto.EndedAt != nil {
		entry.ElapsedSecs = int(dto.EndedAt.Sub(dto.StartedAt).Seconds())
	}
	// Allow explicit override.
	if dto.ElapsedSecs != nil {
		entry.ElapsedSecs = *dto.ElapsedSecs
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
	if dto.ElapsedSecs != nil {
		entry.ElapsedSecs = *dto.ElapsedSecs
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

// ListEntries returns all time entries for a task with a total elapsed duration.
func (s *TimeEntryService) ListEntries(taskID string) ([]models.TimeEntry, int, error) {
	entries, err := s.timeEntryRepo.GetByTaskID(taskID)
	if err != nil {
		return nil, 0, fmt.Errorf("listing time entries: %w", err)
	}

	total := 0
	now := time.Now()
	for _, e := range entries {
		total += e.ElapsedSecs
		// If entry is still running (not ended, not paused), add live delta.
		if e.EndedAt == nil && !e.IsPaused {
			activeStart := e.StartedAt
			if e.ResumedAt != nil {
				activeStart = *e.ResumedAt
			}
			total += int(now.Sub(activeStart).Seconds())
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
