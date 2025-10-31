package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/auth"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/services"
)

type SpaceHandler struct {
	spaceService *services.SpaceService
}

func NewSpaceHandler(spaceService *services.SpaceService) *SpaceHandler {
	return &SpaceHandler{spaceService: spaceService}
}

func (h *SpaceHandler) RegisterSpaceRoutes(app *fiber.App) {
	group := app.Group("/api/spaces", auth.AuthMiddleware())

	group.Post("", h.CreateSpace)
	group.Get("", h.GetSpaces)
	group.Get("/recent", h.GetRecentSpaces)
	group.Get("/:id", h.GetSpaceByID)
	group.Put("/:id", h.UpdateSpace)
	group.Delete("/:id", h.DeleteSpace)
	group.Post("/:id/visit", h.RecordSpaceVisit)
}

// CreateSpace creates a new space
func (h *SpaceHandler) CreateSpace(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	var dto dtos.CreateSpaceDTO
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	space, err := h.spaceService.CreateSpace(authDTO.AccountID, &dto)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := &dtos.SpaceResponseDTO{
		ID:        space.ID,
		Name:      space.Name,
		AccountID: space.AccountID,
		IsDefault: space.IsDefault,
		CreatedAt: space.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: space.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return c.Status(http.StatusCreated).JSON(response)
}

// GetSpaces retrieves all spaces for the user's account
func (h *SpaceHandler) GetSpaces(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	spaces, err := h.spaceService.GetSpacesByAccountID(authDTO.AccountID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := []dtos.SpaceResponseDTO{}
	for _, space := range spaces {
		response = append(response, dtos.SpaceResponseDTO{
			ID:        space.ID,
			Name:      space.Name,
			AccountID: space.AccountID,
			IsDefault: space.IsDefault,
			CreatedAt: space.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: space.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// GetRecentSpaces retrieves the user's recently visited spaces
func (h *SpaceHandler) GetRecentSpaces(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	spaces, err := h.spaceService.GetRecentSpaces(authDTO.User.ID, authDTO.AccountID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := []dtos.SpaceResponseDTO{}
	for _, space := range spaces {
		response = append(response, dtos.SpaceResponseDTO{
			ID:        space.ID,
			Name:      space.Name,
			AccountID: space.AccountID,
			IsDefault: space.IsDefault,
			CreatedAt: space.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: space.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// GetSpaceByID retrieves a single space by ID
func (h *SpaceHandler) GetSpaceByID(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	spaceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid space ID",
		})
	}

	space, err := h.spaceService.GetSpaceByID(spaceID, authDTO.AccountID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := &dtos.SpaceResponseDTO{
		ID:        space.ID,
		Name:      space.Name,
		AccountID: space.AccountID,
		IsDefault: space.IsDefault,
		CreatedAt: space.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: space.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return c.Status(http.StatusOK).JSON(response)
}

// UpdateSpace updates a space
func (h *SpaceHandler) UpdateSpace(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	spaceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid space ID",
		})
	}

	var dto dtos.UpdateSpaceDTO
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	space, err := h.spaceService.UpdateSpace(spaceID, authDTO.AccountID, &dto)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := &dtos.SpaceResponseDTO{
		ID:        space.ID,
		Name:      space.Name,
		AccountID: space.AccountID,
		IsDefault: space.IsDefault,
		CreatedAt: space.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: space.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return c.Status(http.StatusOK).JSON(response)
}

// DeleteSpace deletes a space
func (h *SpaceHandler) DeleteSpace(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	spaceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid space ID",
		})
	}

	err = h.spaceService.DeleteSpace(spaceID, authDTO.AccountID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusNoContent).Send(nil)
}

// RecordSpaceVisit records a visit to a space
func (h *SpaceHandler) RecordSpaceVisit(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	spaceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid space ID",
		})
	}

	// Verify the space exists and belongs to the user's account
	_, err = h.spaceService.GetSpaceByID(spaceID, authDTO.AccountID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	err = h.spaceService.RecordSpaceVisit(authDTO.User.ID, spaceID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "space visit recorded",
	})
}
