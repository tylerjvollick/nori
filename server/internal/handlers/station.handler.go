package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
)

// StationRepoInterface defines the repository methods needed by StationHandler.
type StationRepoInterface interface {
	GetBySpaceID(spaceID uuid.UUID) ([]models.Station, error)
	GetWIPCounts(spaceID uuid.UUID) (map[uuid.UUID]int, error)
}

// stationResponse is the JSON representation of a station with WIP count.
type stationResponse struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	DisplayOrder int       `json:"displayOrder"`
	WIPLimit     int       `json:"wipLimit"`
	WIPCount     int       `json:"wipCount"`
	BufferSize   int       `json:"bufferSize"`
	IsActive     bool      `json:"isActive"`
}

// StationHandler handles HTTP requests for stations.
type StationHandler struct {
	stationRepo StationRepoInterface
}

// NewStationHandler creates a new StationHandler.
func NewStationHandler(stationRepo StationRepoInterface) *StationHandler {
	return &StationHandler{stationRepo: stationRepo}
}

// RegisterStationRoutes registers station API routes on the Fiber app.
func (h *StationHandler) RegisterStationRoutes(app *fiber.App, middlewares ...fiber.Handler) {
	group := app.Group("/api/v1/stations", middlewares...)

	group.Get("", h.ListStations)
}

// ListStations returns all stations for the active space, ordered by display_order.
func (h *StationHandler) ListStations(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	if authDTO.ActiveSpaceID == nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "X-Space-ID header is required",
		})
	}

	stations, err := h.stationRepo.GetBySpaceID(*authDTO.ActiveSpaceID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list stations",
		})
	}

	wipCounts, err := h.stationRepo.GetWIPCounts(*authDTO.ActiveSpaceID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get WIP counts",
		})
	}

	resp := make([]stationResponse, len(stations))
	for i, s := range stations {
		resp[i] = stationResponse{
			ID:           s.ID,
			Name:         s.Name,
			Description:  s.Description,
			DisplayOrder: s.DisplayOrder,
			WIPLimit:     s.WIPLimit,
			WIPCount:     wipCounts[s.ID],
			BufferSize:   s.BufferSize,
			IsActive:     s.IsActive,
		}
	}

	return c.JSON(resp)
}
