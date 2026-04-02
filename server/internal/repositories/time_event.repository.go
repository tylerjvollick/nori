package repositories

import (
	"time"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

// TimeEventFilters holds optional filter parameters for querying time events.
type TimeEventFilters struct {
	UserID       *uuid.UUID
	TicketID     *uuid.UUID
	TicketStepID *uuid.UUID
	StationID    *uuid.UUID
	EventType    *models.TimeEventType
	Source       *models.TimeEventSource
	From         *time.Time
	To           *time.Time
}

type TimeEventRepository struct {
	db *gorm.DB
}

func NewTimeEventRepository(db *gorm.DB) *TimeEventRepository {
	return &TimeEventRepository{db: db}
}

// Create inserts a new time event. TimeEvents are append-only; there is no
// Update or Delete method by design.
func (r *TimeEventRepository) Create(event *models.TimeEvent) error {
	return r.db.Create(event).Error
}

// GetBySpaceID returns time events for a space, ordered by timestamp descending.
// Accepts optional filters to narrow results.
func (r *TimeEventRepository) GetBySpaceID(spaceID uuid.UUID, filters *TimeEventFilters) ([]models.TimeEvent, error) {
	var events []models.TimeEvent
	q := r.db.Where("space_id = ?", spaceID)
	q = applyTimeEventFilters(q, filters)
	err := q.Order(`"timestamp" DESC`).Find(&events).Error
	return events, err
}

// GetByTicketID returns all time events for a ticket, ordered by timestamp descending.
func (r *TimeEventRepository) GetByTicketID(ticketID uuid.UUID) ([]models.TimeEvent, error) {
	var events []models.TimeEvent
	err := r.db.Where("ticket_id = ?", ticketID).
		Order(`"timestamp" DESC`).
		Find(&events).Error
	return events, err
}

// GetByUserID returns all time events for a user, ordered by timestamp descending.
func (r *TimeEventRepository) GetByUserID(userID uuid.UUID) ([]models.TimeEvent, error) {
	var events []models.TimeEvent
	err := r.db.Where("user_id = ?", userID).
		Order(`"timestamp" DESC`).
		Find(&events).Error
	return events, err
}

// GetByStationID returns all time events for a station, ordered by timestamp descending.
func (r *TimeEventRepository) GetByStationID(stationID uuid.UUID) ([]models.TimeEvent, error) {
	var events []models.TimeEvent
	err := r.db.Where("station_id = ?", stationID).
		Order(`"timestamp" DESC`).
		Find(&events).Error
	return events, err
}

// applyTimeEventFilters adds optional WHERE clauses to a query.
func applyTimeEventFilters(q *gorm.DB, f *TimeEventFilters) *gorm.DB {
	if f == nil {
		return q
	}
	if f.UserID != nil {
		q = q.Where("user_id = ?", *f.UserID)
	}
	if f.TicketID != nil {
		q = q.Where("ticket_id = ?", *f.TicketID)
	}
	if f.TicketStepID != nil {
		q = q.Where("ticket_step_id = ?", *f.TicketStepID)
	}
	if f.StationID != nil {
		q = q.Where("station_id = ?", *f.StationID)
	}
	if f.EventType != nil {
		q = q.Where("event_type = ?", *f.EventType)
	}
	if f.Source != nil {
		q = q.Where("source = ?", *f.Source)
	}
	if f.From != nil {
		q = q.Where(`"timestamp" >= ?`, *f.From)
	}
	if f.To != nil {
		q = q.Where(`"timestamp" <= ?`, *f.To)
	}
	return q
}
