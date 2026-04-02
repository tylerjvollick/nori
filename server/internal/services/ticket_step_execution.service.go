package services

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"gorm.io/gorm"
)

// TicketStepExecutionService manages the lifecycle of individual ticket steps:
// starting, pausing, resuming, completing, and skipping. Each transition
// creates a TimeEvent (append-only time log) and an ActivityEntry (chronological
// ticket activity log) via the central ActivityLoggingService. ActualTimeSeconds
// accumulates active working time, excluding pauses.
type TicketStepExecutionService struct {
	db                *gorm.DB
	ticketRepo        *repositories.TicketRepository
	ticketStepRepo    *repositories.TicketStepRepository
	ticketSubStepRepo *repositories.TicketSubStepRepository
	timeEventRepo     *repositories.TimeEventRepository
	activitySvc       *ActivityLoggingService
}

func NewTicketStepExecutionService(
	db *gorm.DB,
	ticketRepo *repositories.TicketRepository,
	ticketStepRepo *repositories.TicketStepRepository,
	ticketSubStepRepo *repositories.TicketSubStepRepository,
	timeEventRepo *repositories.TimeEventRepository,
	activitySvc *ActivityLoggingService,
) *TicketStepExecutionService {
	return &TicketStepExecutionService{
		db:                db,
		ticketRepo:        ticketRepo,
		ticketStepRepo:    ticketStepRepo,
		ticketSubStepRepo: ticketSubStepRepo,
		timeEventRepo:     timeEventRepo,
		activitySvc:       activitySvc,
	}
}

// StartStepInput contains the parameters for starting a ticket step.
type StartStepInput struct {
	StepID  uuid.UUID
	UserID  uuid.UUID
	SpaceID uuid.UUID
	Source  models.TimeEventSource
}

// StartStep transitions a ticket step from pending to in_progress. It records
// StartedAt, creates a check_in TimeEvent, and logs a step_started ActivityEntry.
func (s *TicketStepExecutionService) StartStep(input StartStepInput) (*models.TicketStep, error) {
	step, err := s.ticketStepRepo.GetByID(input.StepID)
	if err != nil {
		return nil, fmt.Errorf("step not found: %w", err)
	}

	if step.Status != models.TicketStepStatusPending {
		return nil, fmt.Errorf("cannot start step: current status is %q, expected %q", step.Status, models.TicketStepStatusPending)
	}

	now := time.Now()
	step.Status = models.TicketStepStatusInProgress
	step.StartedAt = &now
	step.PausedAt = nil

	if err := s.ticketStepRepo.Update(step); err != nil {
		return nil, fmt.Errorf("failed to update step: %w", err)
	}

	// Create check_in TimeEvent
	s.createTimeEvent(input.SpaceID, input.UserID, step, models.TimeEventTypeCheckIn, input.Source, now, nil)

	// Log activity
	s.activitySvc.LogStepStarted(step.TicketID, input.UserID, step.ID, step.Title)

	return step, nil
}

// PauseStepInput contains the parameters for pausing a ticket step.
type PauseStepInput struct {
	StepID         uuid.UUID
	UserID         uuid.UUID
	SpaceID        uuid.UUID
	Source         models.TimeEventSource
	Notes          *string
	LinkedTicketID *uuid.UUID // For interruptions — links to the cause (e.g., maintenance ticket)
}

// PauseStep transitions a step from in_progress to paused. It records PausedAt,
// accumulates active time into ActualTimeSeconds, creates a pause TimeEvent,
// and logs a step_paused ActivityEntry. If LinkedTicketID is provided, an
// additional interruption ActivityEntry is created.
func (s *TicketStepExecutionService) PauseStep(input PauseStepInput) (*models.TicketStep, error) {
	step, err := s.ticketStepRepo.GetByID(input.StepID)
	if err != nil {
		return nil, fmt.Errorf("step not found: %w", err)
	}

	if step.Status != models.TicketStepStatusInProgress {
		return nil, fmt.Errorf("cannot pause step: current status is %q, expected %q", step.Status, models.TicketStepStatusInProgress)
	}

	now := time.Now()

	// Accumulate active time since the step was last started or resumed
	activeStart := s.lastActiveStart(step)
	if activeStart != nil {
		elapsed := int(now.Sub(*activeStart).Seconds())
		step.ActualTimeSeconds += elapsed
	}

	step.Status = models.TicketStepStatusPaused
	step.PausedAt = &now

	if err := s.ticketStepRepo.Update(step); err != nil {
		return nil, fmt.Errorf("failed to update step: %w", err)
	}

	// Create pause TimeEvent
	s.createTimeEvent(input.SpaceID, input.UserID, step, models.TimeEventTypePause, input.Source, now, input.Notes)

	// Log step_paused activity
	s.activitySvc.LogStepPaused(step.TicketID, input.UserID, step.ID, step.Title)

	// If this pause is due to an interruption (linked ticket), log that too
	if input.LinkedTicketID != nil {
		s.activitySvc.LogInterruption(step.TicketID, input.UserID, step.ID, step.Title, *input.LinkedTicketID, nil)
	}

	return step, nil
}

// ResumeStepInput contains the parameters for resuming a paused ticket step.
type ResumeStepInput struct {
	StepID  uuid.UUID
	UserID  uuid.UUID
	SpaceID uuid.UUID
	Source  models.TimeEventSource
}

// ResumeStep transitions a step from paused back to in_progress. It clears
// PausedAt, creates a resume TimeEvent, and logs a step_resumed ActivityEntry.
func (s *TicketStepExecutionService) ResumeStep(input ResumeStepInput) (*models.TicketStep, error) {
	step, err := s.ticketStepRepo.GetByID(input.StepID)
	if err != nil {
		return nil, fmt.Errorf("step not found: %w", err)
	}

	if step.Status != models.TicketStepStatusPaused {
		return nil, fmt.Errorf("cannot resume step: current status is %q, expected %q", step.Status, models.TicketStepStatusPaused)
	}

	now := time.Now()
	step.Status = models.TicketStepStatusInProgress
	step.PausedAt = nil

	if err := s.ticketStepRepo.Update(step); err != nil {
		return nil, fmt.Errorf("failed to update step: %w", err)
	}

	// Create resume TimeEvent
	s.createTimeEvent(input.SpaceID, input.UserID, step, models.TimeEventTypeResume, input.Source, now, nil)

	// Log activity
	s.activitySvc.LogStepResumed(step.TicketID, input.UserID, step.ID, step.Title)

	return step, nil
}

// CompleteStepInput contains the parameters for completing a ticket step.
type CompleteStepInput struct {
	StepID         uuid.UUID
	UserID         uuid.UUID
	SpaceID        uuid.UUID
	Source         models.TimeEventSource
	DeviationNotes *string
	AutoStartNext  bool // Whether to automatically start the next pending step
}

// CompleteStep transitions a step to completed. It records CompletedAt,
// accumulates final active time into ActualTimeSeconds, creates a check_out
// TimeEvent, and logs a step_completed ActivityEntry. If AutoStartNext is true,
// the next pending step (by DisplayOrder) is automatically started.
func (s *TicketStepExecutionService) CompleteStep(input CompleteStepInput) (*models.TicketStep, *models.TicketStep, error) {
	step, err := s.ticketStepRepo.GetByID(input.StepID)
	if err != nil {
		return nil, nil, fmt.Errorf("step not found: %w", err)
	}

	// Allow completing from in_progress or paused
	if step.Status != models.TicketStepStatusInProgress && step.Status != models.TicketStepStatusPaused {
		return nil, nil, fmt.Errorf("cannot complete step: current status is %q, expected %q or %q",
			step.Status, models.TicketStepStatusInProgress, models.TicketStepStatusPaused)
	}

	now := time.Now()

	// Accumulate active time if the step was in_progress (not paused)
	if step.Status == models.TicketStepStatusInProgress {
		activeStart := s.lastActiveStart(step)
		if activeStart != nil {
			elapsed := int(now.Sub(*activeStart).Seconds())
			step.ActualTimeSeconds += elapsed
		}
	}

	step.Status = models.TicketStepStatusCompleted
	step.CompletedAt = &now
	step.PausedAt = nil
	if input.DeviationNotes != nil {
		step.DeviationNotes = input.DeviationNotes
	}

	if err := s.ticketStepRepo.Update(step); err != nil {
		return nil, nil, fmt.Errorf("failed to update step: %w", err)
	}

	// Create check_out TimeEvent
	s.createTimeEvent(input.SpaceID, input.UserID, step, models.TimeEventTypeCheckOut, input.Source, now, nil)

	// Log activity
	s.activitySvc.LogStepCompleted(step.TicketID, input.UserID, step.ID, step.Title, step.ActualTimeSeconds)

	// Auto-start next step if requested
	var nextStep *models.TicketStep
	if input.AutoStartNext {
		nextStep, err = s.autoStartNextStep(step.TicketID, input.UserID, input.SpaceID, input.Source)
		if err != nil {
			log.Println("Failed to auto-start next step:", err)
		}
	}

	return step, nextStep, nil
}

// SkipStepInput contains the parameters for skipping a ticket step.
type SkipStepInput struct {
	StepID uuid.UUID
	UserID uuid.UUID
	Reason *string // Optional reason for skipping
}

// SkipStep transitions a step to skipped. It logs a step_skipped ActivityEntry
// with an optional reason. Only pending or paused steps can be skipped.
func (s *TicketStepExecutionService) SkipStep(input SkipStepInput) (*models.TicketStep, error) {
	step, err := s.ticketStepRepo.GetByID(input.StepID)
	if err != nil {
		return nil, fmt.Errorf("step not found: %w", err)
	}

	// Allow skipping from pending, in_progress, or paused
	if step.Status == models.TicketStepStatusCompleted || step.Status == models.TicketStepStatusSkipped {
		return nil, fmt.Errorf("cannot skip step: current status is %q", step.Status)
	}

	step.Status = models.TicketStepStatusSkipped
	step.PausedAt = nil

	if err := s.ticketStepRepo.Update(step); err != nil {
		return nil, fmt.Errorf("failed to update step: %w", err)
	}

	// Log activity
	s.activitySvc.LogStepSkipped(step.TicketID, input.UserID, step.ID, step.Title, input.Reason)

	return step, nil
}

// autoStartNextStep finds the next pending step (by DisplayOrder) for the same
// ticket and starts it. Returns nil if there are no more pending steps.
func (s *TicketStepExecutionService) autoStartNextStep(
	ticketID uuid.UUID,
	userID uuid.UUID,
	spaceID uuid.UUID,
	source models.TimeEventSource,
) (*models.TicketStep, error) {
	steps, err := s.ticketStepRepo.GetByTicketID(ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket steps: %w", err)
	}

	for _, step := range steps {
		if step.Status == models.TicketStepStatusPending {
			return s.StartStep(StartStepInput{
				StepID:  step.ID,
				UserID:  userID,
				SpaceID: spaceID,
				Source:  source,
			})
		}
	}

	return nil, nil // No more pending steps
}

// lastActiveStart returns the timestamp from which active time should be
// measured. If the step was resumed from a pause, this is when it was last
// started/resumed (approximated by the absence of PausedAt while in_progress).
// For simplicity, we use StartedAt for the first active period. On resume,
// PausedAt is cleared so we need the resume timestamp — which is the current
// time at the point of resume. Since we clear PausedAt on resume, we track
// the "last active start" as: if PausedAt is nil and status is in_progress,
// the step is actively being worked. The active period started at the later
// of StartedAt or the most recent resume (which we approximate via UpdatedAt).
func (s *TicketStepExecutionService) lastActiveStart(step *models.TicketStep) *time.Time {
	if step.StartedAt == nil {
		return nil
	}
	// If the step has never been paused, active time starts at StartedAt.
	// If it was resumed (PausedAt cleared), UpdatedAt reflects the resume time.
	// We use UpdatedAt as the safe approximation of "last active start"
	// because it's updated on every state transition including resume.
	// However, for a freshly started step that's never been paused,
	// UpdatedAt == StartedAt (approximately), so this is correct either way.
	if step.UpdatedAt.After(*step.StartedAt) {
		return &step.UpdatedAt
	}
	return step.StartedAt
}

// createTimeEvent is a helper that creates a TimeEvent. Failures are logged
// but do not propagate — time event creation is secondary to the step transition.
func (s *TicketStepExecutionService) createTimeEvent(
	spaceID uuid.UUID,
	userID uuid.UUID,
	step *models.TicketStep,
	eventType models.TimeEventType,
	source models.TimeEventSource,
	timestamp time.Time,
	notes *string,
) {
	event := &models.TimeEvent{
		SpaceID:      spaceID,
		UserID:       userID,
		TicketID:     &step.TicketID,
		TicketStepID: &step.ID,
		StationID:    step.StationID,
		EventType:    eventType,
		Source:       source,
		Timestamp:    timestamp,
		Notes:        notes,
	}
	if err := s.timeEventRepo.Create(event); err != nil {
		log.Println("Failed to create time event:", err)
	}
}

// stepActivityDescription generates a human-readable description for a step
// transition. Delegates to the central ActivityLoggingService generator.
// Kept for backward compatibility with existing tests.
func stepActivityDescription(action, stepTitle string) string {
	return stepActivityDesc(action, stepTitle)
}
