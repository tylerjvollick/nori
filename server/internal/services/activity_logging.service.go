package services

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
)

// ActivityLoggingService is the central service for logging activity entries
// on tickets. All other services call this instead of creating ActivityEntry
// records directly. Description generation is handled here so that the format
// is consistent across the codebase. Failures are always logged but never
// propagated — activity logging is secondary to the operations it decorates.
type ActivityLoggingService struct {
	activityRepo *repositories.ActivityEntryRepository
}

func NewActivityLoggingService(
	activityRepo *repositories.ActivityEntryRepository,
) *ActivityLoggingService {
	return &ActivityLoggingService{
		activityRepo: activityRepo,
	}
}

// --- Ticket-level activities ---

// LogTicketCreated logs a ticket_created activity entry.
func (s *ActivityLoggingService) LogTicketCreated(ticketID, userID uuid.UUID, ticketNumber string, hasSOP bool, defaultStatusName string) {
	desc := ticketCreatedDescription(ticketNumber, hasSOP)
	s.log(&models.ActivityEntry{
		TicketID:    ticketID,
		UserID:      userID,
		EntryType:   models.ActivityEntryTypeTicketCreated,
		Description: desc,
		NewValue:    &defaultStatusName,
	})
}

// LogStatusChange logs a status_change activity entry with old and new values.
func (s *ActivityLoggingService) LogStatusChange(ticketID, userID uuid.UUID, oldStatusName, newStatusName string) {
	s.log(&models.ActivityEntry{
		TicketID:    ticketID,
		UserID:      userID,
		EntryType:   models.ActivityEntryTypeStatusChange,
		Description: statusChangeDesc(oldStatusName, newStatusName),
		OldValue:    &oldStatusName,
		NewValue:    &newStatusName,
	})
}

// LogAssignmentChange logs an assignment_change activity entry.
func (s *ActivityLoggingService) LogAssignmentChange(ticketID, userID uuid.UUID, oldAssignee, newAssignee string) {
	s.log(&models.ActivityEntry{
		TicketID:    ticketID,
		UserID:      userID,
		EntryType:   models.ActivityEntryTypeAssignmentChange,
		Description: assignmentChangeDescription(oldAssignee, newAssignee),
		OldValue:    &oldAssignee,
		NewValue:    &newAssignee,
	})
}

// LogComment logs a comment activity entry.
func (s *ActivityLoggingService) LogComment(ticketID, userID uuid.UUID, commentText string) {
	s.log(&models.ActivityEntry{
		TicketID:    ticketID,
		UserID:      userID,
		EntryType:   models.ActivityEntryTypeComment,
		Description: commentDescription(commentText),
	})
}

// LogLinkAdded logs a link_added activity entry.
func (s *ActivityLoggingService) LogLinkAdded(ticketID, userID uuid.UUID, linkedTicketID uuid.UUID, linkType string) {
	s.log(&models.ActivityEntry{
		TicketID:       ticketID,
		UserID:         userID,
		EntryType:      models.ActivityEntryTypeLinkAdded,
		Description:    linkAddedDescription(linkType),
		LinkedTicketID: &linkedTicketID,
	})
}

// LogSOPEdited logs an sop_edited activity entry.
func (s *ActivityLoggingService) LogSOPEdited(ticketID, userID uuid.UUID, description string) {
	s.log(&models.ActivityEntry{
		TicketID:    ticketID,
		UserID:      userID,
		EntryType:   models.ActivityEntryTypeSOPEdited,
		Description: description,
	})
}

// LogCostLogged logs a cost_logged activity entry.
func (s *ActivityLoggingService) LogCostLogged(ticketID, userID uuid.UUID, costType string, amountCents int) {
	s.log(&models.ActivityEntry{
		TicketID:    ticketID,
		UserID:      userID,
		EntryType:   models.ActivityEntryTypeCostLogged,
		Description: costLoggedDescription(costType, amountCents),
	})
}

// --- Step-level activities ---

// LogStepStarted logs a step_started activity entry.
func (s *ActivityLoggingService) LogStepStarted(ticketID, userID, stepID uuid.UUID, stepTitle string) {
	s.logStep(ticketID, userID, stepID, models.ActivityEntryTypeStepStarted,
		stepActivityDesc("started", stepTitle))
}

// LogStepPaused logs a step_paused activity entry.
func (s *ActivityLoggingService) LogStepPaused(ticketID, userID, stepID uuid.UUID, stepTitle string) {
	s.logStep(ticketID, userID, stepID, models.ActivityEntryTypeStepPaused,
		stepActivityDesc("paused", stepTitle))
}

// LogStepResumed logs a step_resumed activity entry.
func (s *ActivityLoggingService) LogStepResumed(ticketID, userID, stepID uuid.UUID, stepTitle string) {
	s.logStep(ticketID, userID, stepID, models.ActivityEntryTypeStepResumed,
		stepActivityDesc("resumed", stepTitle))
}

// LogStepCompleted logs a step_completed activity entry with actual time.
func (s *ActivityLoggingService) LogStepCompleted(ticketID, userID, stepID uuid.UUID, stepTitle string, actualTimeSeconds int) {
	actualMinutes := actualTimeSeconds / 60
	desc := fmt.Sprintf("Step %q completed (%d min active time)", stepTitle, actualMinutes)
	s.logStep(ticketID, userID, stepID, models.ActivityEntryTypeStepCompleted, desc)
}

// LogStepSkipped logs a step_skipped activity entry with an optional reason.
func (s *ActivityLoggingService) LogStepSkipped(ticketID, userID, stepID uuid.UUID, stepTitle string, reason *string) {
	desc := fmt.Sprintf("Step %q skipped", stepTitle)
	if reason != nil && *reason != "" {
		desc = fmt.Sprintf("Step %q skipped: %s", stepTitle, *reason)
	}
	s.logStep(ticketID, userID, stepID, models.ActivityEntryTypeStepSkipped, desc)
}

// --- Interruption ---

// LogInterruption logs an interruption activity entry with a linked ticket
// and optional duration.
func (s *ActivityLoggingService) LogInterruption(ticketID, userID, stepID uuid.UUID, stepTitle string, linkedTicketID uuid.UUID, durationSeconds *int) {
	entry := &models.ActivityEntry{
		TicketID:        ticketID,
		UserID:          userID,
		EntryType:       models.ActivityEntryTypeInterruption,
		Description:     interruptionDescription(stepTitle, durationSeconds),
		TicketStepID:    &stepID,
		LinkedTicketID:  &linkedTicketID,
		DurationSeconds: durationSeconds,
	}
	s.log(entry)
}

// --- Internal helpers ---

// log persists an activity entry. Failures are logged but never returned.
func (s *ActivityLoggingService) log(entry *models.ActivityEntry) {
	if err := s.activityRepo.Create(entry); err != nil {
		log.Printf("Failed to log %s activity for ticket %s: %v", entry.EntryType, entry.TicketID, err)
	}
}

// logStep creates and persists a step-scoped activity entry.
func (s *ActivityLoggingService) logStep(ticketID, userID, stepID uuid.UUID, entryType models.ActivityEntryType, description string) {
	s.log(&models.ActivityEntry{
		TicketID:     ticketID,
		UserID:       userID,
		EntryType:    entryType,
		Description:  description,
		TicketStepID: &stepID,
	})
}

// --- Description generators (exported for testing) ---

func ticketCreatedDescription(ticketNumber string, hasSOP bool) string {
	desc := fmt.Sprintf("Ticket %s created", ticketNumber)
	if hasSOP {
		desc += " with linked SOP"
	}
	return desc
}

func statusChangeDesc(oldName, newName string) string {
	return fmt.Sprintf("Status changed from %q to %q", oldName, newName)
}

func stepActivityDesc(action, stepTitle string) string {
	return fmt.Sprintf("Step %q %s", stepTitle, action)
}

func assignmentChangeDescription(oldAssignee, newAssignee string) string {
	if oldAssignee == "" {
		return fmt.Sprintf("Assigned to %s", newAssignee)
	}
	if newAssignee == "" {
		return fmt.Sprintf("Unassigned from %s", oldAssignee)
	}
	return fmt.Sprintf("Reassigned from %s to %s", oldAssignee, newAssignee)
}

func commentDescription(commentText string) string {
	if len(commentText) > 100 {
		return commentText[:100] + "..."
	}
	return commentText
}

func linkAddedDescription(linkType string) string {
	return fmt.Sprintf("Link added: %s", linkType)
}

func costLoggedDescription(costType string, amountCents int) string {
	dollars := float64(amountCents) / 100.0
	return fmt.Sprintf("%s cost logged: $%.2f", costType, dollars)
}

func interruptionDescription(stepTitle string, durationSeconds *int) string {
	desc := fmt.Sprintf("Step %q paused for interruption", stepTitle)
	if durationSeconds != nil && *durationSeconds > 0 {
		minutes := *durationSeconds / 60
		desc = fmt.Sprintf("Step %q paused for interruption (%d min)", stepTitle, minutes)
	}
	return desc
}
