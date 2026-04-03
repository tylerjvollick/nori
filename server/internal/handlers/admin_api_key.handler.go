package handlers

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/middleware"
	"github.com/tylerjvollick/nori/internal/models"
)

// APIKeyServiceInterface defines the methods needed by the API key handler
type APIKeyServiceInterface interface {
	GenerateAPIKey(accountID uuid.UUID, name string, createdByID uuid.UUID, expiresAt *time.Time) (rawKey string, apiKey *models.APIKey, err error)
}

// APIKeyQueryRepositoryInterface defines the read/write methods needed by the API key handler
// (beyond what the service provides for key generation)
type APIKeyQueryRepositoryInterface interface {
	GetByAccount(accountID uuid.UUID) ([]models.APIKey, error)
	Deactivate(id uuid.UUID) error
	Delete(id uuid.UUID) error
}

type AdminAPIKeyHandler struct {
	apiKeyService APIKeyServiceInterface
	apiKeyRepo    APIKeyQueryRepositoryInterface
}

func NewAdminAPIKeyHandler(
	apiKeyService APIKeyServiceInterface,
	apiKeyRepo APIKeyQueryRepositoryInterface,
) *AdminAPIKeyHandler {
	return &AdminAPIKeyHandler{
		apiKeyService: apiKeyService,
		apiKeyRepo:    apiKeyRepo,
	}
}

func (h *AdminAPIKeyHandler) RegisterAdminAPIKeyRoutes(app *fiber.App) {
	admin := app.Group("/admin")

	// Apply RequireAdmin middleware to all admin routes
	admin.Use(middleware.RequireAdmin())

	// API key management endpoints
	admin.Post("/api-keys", h.CreateAPIKey)
	admin.Get("/api-keys", h.ListAPIKeys)
	admin.Delete("/api-keys/:id", h.DeleteAPIKey)
}

// CreateAPIKeyRequest defines the request body for creating an API key
type CreateAPIKeyRequest struct {
	Name      string  `json:"name"`
	ExpiresAt *string `json:"expiresAt,omitempty"` // RFC3339 format
}

// CreateAPIKeyResponse includes the raw key (shown only once) plus the key metadata
type CreateAPIKeyResponse struct {
	RawKey string        `json:"rawKey"`
	APIKey models.APIKey `json:"apiKey"`
}

// CreateAPIKey handles POST /admin/api-keys
func (h *AdminAPIKeyHandler) CreateAPIKey(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	var req CreateAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "name is required",
		})
	}

	// Parse optional expiration
	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid expiresAt format, must be RFC3339",
			})
		}
		expiresAt = &t
	}

	// Generate the API key via the service
	rawKey, apiKey, err := h.apiKeyService.GenerateAPIKey(
		authDTO.AccountID,
		req.Name,
		authDTO.User.ID,
		expiresAt,
	)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(CreateAPIKeyResponse{
		RawKey: rawKey,
		APIKey: *apiKey,
	})
}

// ListAPIKeys handles GET /admin/api-keys
func (h *AdminAPIKeyHandler) ListAPIKeys(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	apiKeys, err := h.apiKeyRepo.GetByAccount(authDTO.AccountID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(apiKeys)
}

// DeleteAPIKey handles DELETE /admin/api-keys/:id
func (h *AdminAPIKeyHandler) DeleteAPIKey(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	// Parse API key ID from path parameter
	keyIDStr := c.Params("id")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid API key ID",
		})
	}

	// Deactivate the key (soft delete — preserves audit trail)
	err = h.apiKeyRepo.Deactivate(keyID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusNoContent).Send(nil)
}
