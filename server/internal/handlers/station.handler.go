package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
)

// StationRepoInterface defines the repository methods needed by StationHandler.
type StationRepoInterface interface {
	Create(station *models.Station) error
	GetBySpaceID(spaceID uuid.UUID) ([]models.Station, error)
	GetMaxDisplayOrder(spaceID uuid.UUID) (int, error)
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

// createStationRequest is the JSON body for POST /api/v1/stations.
type createStationRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	WIPLimit    *int    `json:"wipLimit,omitempty"`
}

// RegisterStationRoutes registers station API routes on the Fiber app.
func (h *StationHandler) RegisterStationRoutes(app *fiber.App, middlewares ...fiber.Handler) {
	group := app.Group("/api/v1/stations", middlewares...)

	group.Get("", h.ListStations)
	group.Post("", h.CreateStation)
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

// CreateStation creates a new station in the active space (admin only).
func (h *StationHandler) CreateStation(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	// Admin only
	if !isAdmin(authDTO) {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "admin access required",
		})
	}

	if authDTO.ActiveSpaceID == nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "X-Space-ID header is required",
		})
	}

	var req createStationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "name is required",
		})
	}

	// Auto-assign displayOrder = max existing + 1
	maxOrder, err := h.stationRepo.GetMaxDisplayOrder(*authDTO.ActiveSpaceID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to determine display order",
		})
	}

	wipLimit := 1
	if req.WIPLimit != nil {
		wipLimit = *req.WIPLimit
	}

	station := &models.Station{
		ID:           uuid.New(),
		SpaceID:      *authDTO.ActiveSpaceID,
		Name:         req.Name,
		Description:  req.Description,
		DisplayOrder: maxOrder + 1,
		WIPLimit:     wipLimit,
		IsActive:     true,
	}

	if err := h.stationRepo.Create(station); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create station",
		})
	}

	return c.Status(http.StatusCreated).JSON(stationResponse{
		ID:           station.ID,
		Name:         station.Name,
		Description:  station.Description,
		DisplayOrder: station.DisplayOrder,
		WIPLimit:     station.WIPLimit,
		WIPCount:     0,
		BufferSize:   station.BufferSize,
		IsActive:     station.IsActive,
	})
}
