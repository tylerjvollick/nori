package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
)

// SpaceServiceInterface defines the methods needed by SpaceHandler
type SpaceServiceInterface interface {
	CreateSpace(accountID uuid.UUID, dto *dtos.CreateSpaceDTO) (*models.Space, error)
	GetSpaceByID(spaceID uuid.UUID, accountID uuid.UUID) (*models.Space, error)
	GetSpacesByAccountID(accountID uuid.UUID) ([]models.Space, error)
	GetRecentSpaces(userID uuid.UUID, accountID uuid.UUID) ([]models.Space, error)
	UpdateSpace(spaceID uuid.UUID, accountID uuid.UUID, dto *dtos.UpdateSpaceDTO) (*models.Space, error)
	DeleteSpace(spaceID uuid.UUID, accountID uuid.UUID) error
	RecordSpaceVisit(userID uuid.UUID, spaceID uuid.UUID) error
}

type SpaceHandler struct {
	spaceService    SpaceServiceInterface
	spaceMemberRepo SpaceMemberRepositoryInterface
}

func NewSpaceHandler(spaceService SpaceServiceInterface, spaceMemberRepo SpaceMemberRepositoryInterface) *SpaceHandler {
	return &SpaceHandler{
		spaceService:    spaceService,
		spaceMemberRepo: spaceMemberRepo,
	}
}

func (h *SpaceHandler) RegisterSpaceRoutes(app *fiber.App, middlewares ...fiber.Handler) {
	group := app.Group("/api/spaces", middlewares...)

	group.Post("", h.CreateSpace)
	group.Get("", h.GetSpaces)
	group.Get("/recent", h.GetRecentSpaces)
	group.Get("/:id", h.GetSpaceByID)
	group.Put("/:id", h.UpdateSpace)
	group.Delete("/:id", h.DeleteSpace)
	group.Post("/:id/visit", h.RecordSpaceVisit)
}

// isAdmin returns true if the authenticated user has admin role
func isAdmin(authDTO *dtos.AuthDTO) bool {
	return authDTO.User.Role != nil && *authDTO.User.Role == models.RoleAdmin
}

// hasSpaceAccess returns true if the user is an admin or a member of the given space.
func (h *SpaceHandler) hasSpaceAccess(authDTO *dtos.AuthDTO, spaceID uuid.UUID) bool {
	if isAdmin(authDTO) {
		return true
	}

	_, err := h.spaceMemberRepo.GetByUserAndSpace(authDTO.User.ID, spaceID)
	return err == nil
}

// CreateSpace creates a new space (admin only)
func (h *SpaceHandler) CreateSpace(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	// Only admins can create spaces
	if !isAdmin(authDTO) {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "admin access required",
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

// GetSpaces retrieves spaces for the user.
// Admins see all account spaces; regular users see only spaces they are a member of.
func (h *SpaceHandler) GetSpaces(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	var spaceModels []models.Space

	if isAdmin(authDTO) {
		// Admins see all spaces for the account
		spaceModels, err = h.spaceService.GetSpacesByAccountID(authDTO.AccountID)
	} else {
		// Regular users see only spaces they are a member of
		spaceModels, err = h.spaceMemberRepo.GetByUser(authDTO.User.ID)
	}

	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := []dtos.SpaceResponseDTO{}
	for _, space := range spaceModels {
		// For non-admin users, also filter to their account
		if space.AccountID != authDTO.AccountID {
			continue
		}
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
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
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

// GetSpaceByID retrieves a single space by ID (admin or space member)
func (h *SpaceHandler) GetSpaceByID(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
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

	// Check space access (admin or member)
	if !h.hasSpaceAccess(authDTO, spaceID) {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "access to this space is not permitted",
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

// UpdateSpace updates a space (admin or space member)
func (h *SpaceHandler) UpdateSpace(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	spaceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid space ID",
		})
	}

	// Check space access (admin or member)
	if !h.hasSpaceAccess(authDTO, spaceID) {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "access to this space is not permitted",
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

// DeleteSpace deletes a space (admin or space member)
func (h *SpaceHandler) DeleteSpace(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	spaceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid space ID",
		})
	}

	// Check space access (admin or member)
	if !h.hasSpaceAccess(authDTO, spaceID) {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "access to this space is not permitted",
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

// RecordSpaceVisit records a visit to a space (admin or space member)
func (h *SpaceHandler) RecordSpaceVisit(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
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

	// Check space access (admin or member)
	if !h.hasSpaceAccess(authDTO, spaceID) {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "access to this space is not permitted",
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
