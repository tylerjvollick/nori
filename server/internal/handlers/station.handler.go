package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
)

// StationRepoInterface defines the repository methods needed by StationHandler.
type StationRepoInterface interface {
	Create(station *models.Station) error
	GetByID(id uuid.UUID) (*models.Station, error)
	GetBySpaceID(spaceID uuid.UUID) ([]models.Station, error)
	Update(station *models.Station) error
	Delete(id uuid.UUID) error
	GetMaxDisplayOrder(spaceID uuid.UUID) (int, error)
	GetWIPCounts(spaceID uuid.UUID) (map[uuid.UUID]int, error)
}

// StationHandler handles HTTP requests for stations.
type StationHandler struct {
	stationRepo StationRepoInterface
}

// NewStationHandler creates a new StationHandler.
func NewStationHandler(stationRepo StationRepoInterface) *StationHandler {
	return &StationHandler{stationRepo: stationRepo}
}

// RegisterStationRoutes registers station API routes under a space-scoped router.
// Expects the router to already have :spaceId in its path and RequireSpace middleware applied.
func (h *StationHandler) RegisterStationRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	group := router.Group("/stations", middlewares...)

	group.Get("", h.ListStations)
	group.Post("", h.CreateStation)
	group.Get("/:id", h.GetStation)
	group.Put("/:id", h.UpdateStation)
	group.Delete("/:id", h.DeleteStation)
}

// ListStations returns all stations for the space, ordered by display_order.
func (h *StationHandler) ListStations(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	stations, err := h.stationRepo.GetBySpaceID(spaceID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list stations",
		})
	}

	wipCounts, err := h.stationRepo.GetWIPCounts(spaceID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get WIP counts",
		})
	}

	resp := make([]dtos.StationResponse, len(stations))
	for i, s := range stations {
		resp[i] = dtos.StationResponseFromModel(&s, wipCounts[s.ID])
	}

	return c.JSON(resp)
}

// CreateStation creates a new station in the space (admin only).
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

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	var req dtos.CreateStationRequest
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
	maxOrder, err := h.stationRepo.GetMaxDisplayOrder(spaceID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to determine display order",
		})
	}

	wipLimit := 1
	if req.WIPLimit != nil {
		wipLimit = *req.WIPLimit
	}

	bufferSize := 0
	if req.BufferSize != nil {
		bufferSize = *req.BufferSize
	}

	station := &models.Station{
		ID:           uuid.New(),
		SpaceID:      spaceID,
		Name:         req.Name,
		Description:  req.Description,
		DisplayOrder: maxOrder + 1,
		WIPLimit:     wipLimit,
		BufferSize:   bufferSize,
		IsActive:     true,
	}

	if err := h.stationRepo.Create(station); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create station",
		})
	}

	return c.Status(http.StatusCreated).JSON(dtos.StationResponseFromModel(station, 0))
}

// GetStation returns a single station by ID.
func (h *StationHandler) GetStation(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid station ID",
		})
	}

	station, err := h.stationRepo.GetByID(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "station not found",
		})
	}

	// Verify the station belongs to the requested space.
	if station.SpaceID != spaceID {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "station not found",
		})
	}

	wipCounts, err := h.stationRepo.GetWIPCounts(spaceID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get WIP counts",
		})
	}

	return c.JSON(dtos.StationResponseFromModel(station, wipCounts[station.ID]))
}

// UpdateStation updates station fields (admin only).
func (h *StationHandler) UpdateStation(c *fiber.Ctx) error {
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

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid station ID",
		})
	}

	station, err := h.stationRepo.GetByID(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "station not found",
		})
	}

	// Verify the station belongs to the requested space.
	if station.SpaceID != spaceID {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "station not found",
		})
	}

	var req dtos.UpdateStationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Name != nil {
		station.Name = *req.Name
	}
	if req.Description != nil {
		station.Description = req.Description
	}
	if req.WIPLimit != nil {
		station.WIPLimit = *req.WIPLimit
	}
	if req.BufferSize != nil {
		station.BufferSize = *req.BufferSize
	}
	if req.CostsHour.Set {
		if req.CostsHour.Valid && req.CostsHour.Value.IsNegative() {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "costsHour must be >= 0",
			})
		}
		station.CostsHour = req.CostsHour.Ptr()
	}
	if req.IsActive != nil {
		station.IsActive = *req.IsActive
	}

	if err := h.stationRepo.Update(station); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update station",
		})
	}

	wipCounts, err := h.stationRepo.GetWIPCounts(spaceID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get WIP counts",
		})
	}

	return c.JSON(dtos.StationResponseFromModel(station, wipCounts[station.ID]))
}

// DeleteStation deletes a station (admin only).
func (h *StationHandler) DeleteStation(c *fiber.Ctx) error {
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

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid station ID",
		})
	}

	station, err := h.stationRepo.GetByID(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "station not found",
		})
	}

	// Verify the station belongs to the requested space.
	if station.SpaceID != spaceID {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "station not found",
		})
	}

	if err := h.stationRepo.Delete(id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete station",
		})
	}

	return c.Status(http.StatusNoContent).Send(nil)
}
