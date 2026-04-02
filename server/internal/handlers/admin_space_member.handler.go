package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/middleware"
	"github.com/tylerjvollick/nori/internal/models"
)

// SpaceMemberRepositoryInterface defines the methods needed by the space member handler
type SpaceMemberRepositoryInterface interface {
	Create(spaceMember *models.SpaceMember) error
	Delete(userID, spaceID uuid.UUID) error
	GetByUserAndSpace(userID, spaceID uuid.UUID) (*models.SpaceMember, error)
	GetBySpace(spaceID uuid.UUID) ([]models.SpaceMember, error)
}

// SpaceRepositoryInterface defines the methods needed by the space member handler
type SpaceRepositoryInterface interface {
	FindByID(id uuid.UUID) (*models.Space, error)
}

type AdminSpaceMemberHandler struct {
	spaceMemberRepo SpaceMemberRepositoryInterface
	spaceRepo       SpaceRepositoryInterface
}

func NewAdminSpaceMemberHandler(
	spaceMemberRepo SpaceMemberRepositoryInterface,
	spaceRepo SpaceRepositoryInterface,
) *AdminSpaceMemberHandler {
	return &AdminSpaceMemberHandler{
		spaceMemberRepo: spaceMemberRepo,
		spaceRepo:       spaceRepo,
	}
}

func (h *AdminSpaceMemberHandler) RegisterAdminSpaceMemberRoutes(app *fiber.App) {
	admin := app.Group("/admin")

	// Apply RequireAdmin middleware to all admin routes
	admin.Use(middleware.RequireAdmin())

	// Space membership endpoints
	admin.Post("/spaces/:id/members", h.AddSpaceMember)
	admin.Get("/spaces/:id/members", h.GetSpaceMembers)
	admin.Delete("/spaces/:id/members/:userId", h.RemoveSpaceMember)
}

// AddSpaceMemberRequest defines the request body for adding a user to a space
type AddSpaceMemberRequest struct {
	UserID string `json:"userId"`
}

// AddSpaceMember handles POST /admin/spaces/:id/members
func (h *AdminSpaceMemberHandler) AddSpaceMember(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	// Parse space ID from path parameter
	spaceIDStr := c.Params("id")
	spaceID, err := uuid.Parse(spaceIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid space ID",
		})
	}

	var req AddSpaceMemberRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate required fields
	if req.UserID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "userId is required",
		})
	}

	// Parse user ID
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user ID",
		})
	}

	// Verify space exists and belongs to the account
	space, err := h.spaceRepo.FindByID(spaceID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "space not found",
		})
	}

	if space.AccountID != authDTO.AccountID {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "space does not belong to your account",
		})
	}

	// Check if membership already exists
	existingMember, _ := h.spaceMemberRepo.GetByUserAndSpace(userID, spaceID)
	if existingMember != nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{
			"error": "user is already a member of this space",
		})
	}

	// Create space member
	spaceMember := &models.SpaceMember{
		UserID:  userID,
		SpaceID: spaceID,
	}

	err = h.spaceMemberRepo.Create(spaceMember)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(spaceMember)
}

// RemoveSpaceMember handles DELETE /admin/spaces/:id/members/:userId
func (h *AdminSpaceMemberHandler) RemoveSpaceMember(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	// Parse space ID from path parameter
	spaceIDStr := c.Params("id")
	spaceID, err := uuid.Parse(spaceIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid space ID",
		})
	}

	// Parse user ID from path parameter
	userIDStr := c.Params("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user ID",
		})
	}

	// Verify space exists and belongs to the account
	space, err := h.spaceRepo.FindByID(spaceID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "space not found",
		})
	}

	if space.AccountID != authDTO.AccountID {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "space does not belong to your account",
		})
	}

	// Delete space member
	err = h.spaceMemberRepo.Delete(userID, spaceID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusNoContent).Send(nil)
}

// GetSpaceMembers handles GET /admin/spaces/:id/members
func (h *AdminSpaceMemberHandler) GetSpaceMembers(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	// Parse space ID from path parameter
	spaceIDStr := c.Params("id")
	spaceID, err := uuid.Parse(spaceIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid space ID",
		})
	}

	// Verify space exists and belongs to the account
	space, err := h.spaceRepo.FindByID(spaceID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "space not found",
		})
	}

	if space.AccountID != authDTO.AccountID {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "space does not belong to your account",
		})
	}

	// Get space members
	members, err := h.spaceMemberRepo.GetBySpace(spaceID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Don't return password hashes
	for i := range members {
		if members[i].User.Password != nil {
			members[i].User.Password = nil
		}
	}

	return c.Status(http.StatusOK).JSON(members)
}
