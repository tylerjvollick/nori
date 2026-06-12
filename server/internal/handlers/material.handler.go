package handlers

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/services"
)

// MaterialServiceInterface defines the service methods needed by
// MaterialHandler.
type MaterialServiceInterface interface {
	Create(spaceID uuid.UUID, req *dtos.CreateMaterialRequest) (*models.Material, error)
	List(spaceID uuid.UUID, search string) ([]models.Material, error)
	GetByID(spaceID, id uuid.UUID) (*models.Material, error)
	Update(spaceID, id uuid.UUID, req *dtos.UpdateMaterialRequest) (*models.Material, error)
	Delete(spaceID, id uuid.UUID) error
}

// MaterialHandler handles HTTP requests for the material catalog.
type MaterialHandler struct {
	materialService MaterialServiceInterface
}

// NewMaterialHandler creates a new MaterialHandler.
func NewMaterialHandler(materialService MaterialServiceInterface) *MaterialHandler {
	return &MaterialHandler{materialService: materialService}
}

// RegisterMaterialRoutes registers material API routes under a space-scoped
// router. Expects the router to already have :spaceId in its path and the
// RequireSpace middleware applied.
func (h *MaterialHandler) RegisterMaterialRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	group := router.Group("/materials", middlewares...)

	group.Post("", h.CreateMaterial)
	group.Get("", h.ListMaterials)
	group.Get("/:id", h.GetMaterial)
	group.Put("/:id", h.UpdateMaterial)
	group.Delete("/:id", h.DeleteMaterial)
}

// materialErrorResponse maps service errors to HTTP responses.
func materialErrorResponse(c *fiber.Ctx, err error) error {
	var validationErr *services.MaterialValidationError
	if errors.As(err, &validationErr) {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": validationErr.Message})
	}

	if errors.Is(err, services.ErrMaterialNotFound) {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "material not found"})
	}

	return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
}

// CreateMaterial handles POST /api/v1/spaces/:spaceId/materials.
func (h *MaterialHandler) CreateMaterial(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	var req dtos.CreateMaterialRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	material, err := h.materialService.Create(spaceID, &req)
	if err != nil {
		return materialErrorResponse(c, err)
	}

	return c.Status(http.StatusCreated).JSON(dtos.MaterialResponseFromModel(material))
}

// ListMaterials handles GET /api/v1/spaces/:spaceId/materials.
// Supports an optional ?search= query param for name filtering.
func (h *MaterialHandler) ListMaterials(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	materials, err := h.materialService.List(spaceID, c.Query("search"))
	if err != nil {
		return materialErrorResponse(c, err)
	}

	resp := make([]dtos.MaterialResponse, len(materials))
	for i := range materials {
		resp[i] = dtos.MaterialResponseFromModel(&materials[i])
	}

	return c.JSON(resp)
}

// GetMaterial handles GET /api/v1/spaces/:spaceId/materials/:id.
func (h *MaterialHandler) GetMaterial(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid material ID"})
	}

	material, err := h.materialService.GetByID(spaceID, id)
	if err != nil {
		return materialErrorResponse(c, err)
	}

	return c.JSON(dtos.MaterialResponseFromModel(material))
}

// UpdateMaterial handles PUT /api/v1/spaces/:spaceId/materials/:id.
func (h *MaterialHandler) UpdateMaterial(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid material ID"})
	}

	var req dtos.UpdateMaterialRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	material, err := h.materialService.Update(spaceID, id, &req)
	if err != nil {
		return materialErrorResponse(c, err)
	}

	return c.JSON(dtos.MaterialResponseFromModel(material))
}

// DeleteMaterial handles DELETE /api/v1/spaces/:spaceId/materials/:id.
// The material is soft-deleted and excluded from subsequent list results.
func (h *MaterialHandler) DeleteMaterial(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid material ID"})
	}

	if err := h.materialService.Delete(spaceID, id); err != nil {
		return materialErrorResponse(c, err)
	}

	return c.Status(http.StatusNoContent).Send(nil)
}
