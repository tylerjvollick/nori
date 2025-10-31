package handlers

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/tylerjvollick/nori/internal/auth"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/services"
)

type SOPHandler struct {
	sopService *services.SOPService
}

func NewSOPHandler(sopService *services.SOPService) *SOPHandler {
	return &SOPHandler{sopService: sopService}
}

func (h *SOPHandler) RegisterSOPRoutes(app *fiber.App) {
	group := app.Group("/sops")

	// Public routes (or add auth middleware if needed)
	group.Get("/", auth.AuthMiddleware(), h.GetAllSOPs)
	group.Get("/:id", auth.AuthMiddleware(), h.GetSOP)
	group.Get("/:id/versions", auth.AuthMiddleware(), h.GetSOPVersions)
	group.Get("/versions/:versionId", auth.AuthMiddleware(), h.GetSOPVersion)

	// Protected routes
	group.Post("/", auth.AuthMiddleware(), h.CreateSOP)
	group.Put("/:id", auth.AuthMiddleware(), h.UpdateSOP)
	group.Delete("/:id", auth.AuthMiddleware(), h.DeleteSOP)
}

// CreateSOP creates a new SOP template with its first version
func (h *SOPHandler) CreateSOP(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	var dto dtos.CreateSOPDTO
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	template, err := h.sopService.CreateSOP(&dto, authDTO.User.ID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(h.templateToResponse(template))
}

// UpdateSOP creates a new version of an existing SOP
func (h *SOPHandler) UpdateSOP(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid id",
		})
	}

	var dto dtos.UpdateSOPDTO
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	template, err := h.sopService.UpdateSOP(id, &dto, authDTO.User.ID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(h.templateToResponse(template))
}

// GetSOP gets an SOP template by ID
func (h *SOPHandler) GetSOP(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid id",
		})
	}

	template, err := h.sopService.GetSOP(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "SOP not found",
		})
	}

	return c.Status(http.StatusOK).JSON(h.templateToResponse(template))
}

// GetAllSOPs gets all SOP templates
func (h *SOPHandler) GetAllSOPs(c *fiber.Ctx) error {
	templates, err := h.sopService.GetAllSOPs()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := []dtos.SOPTemplateResponseDTO{}
	for _, template := range templates {
		response = append(response, *h.templateToResponse(&template))
	}

	return c.Status(http.StatusOK).JSON(response)
}

// GetSOPVersions gets all versions of an SOP template
func (h *SOPHandler) GetSOPVersions(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid id",
		})
	}

	versions, err := h.sopService.GetSOPVersions(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := []dtos.SOPVersionResponseDTO{}
	for _, version := range versions {
		response = append(response, *h.versionToResponse(&version))
	}

	return c.Status(http.StatusOK).JSON(response)
}

// GetSOPVersion gets a specific version of an SOP template
func (h *SOPHandler) GetSOPVersion(c *fiber.Ctx) error {
	versionID, err := strconv.Atoi(c.Params("versionId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid version id",
		})
	}

	version, err := h.sopService.GetSOPVersion(versionID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "version not found",
		})
	}

	return c.Status(http.StatusOK).JSON(h.versionToResponse(version))
}

// DeleteSOP deletes an SOP template
func (h *SOPHandler) DeleteSOP(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid id",
		})
	}

	if err := h.sopService.DeleteSOP(id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusNoContent).Send(nil)
}

// Helper functions to convert models to DTOs

func (h *SOPHandler) templateToResponse(template *models.SOPTemplate) *dtos.SOPTemplateResponseDTO {
	response := &dtos.SOPTemplateResponseDTO{
		ID:        template.ID,
		Name:      template.Name,
		CreatedAt: template.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: template.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if template.CurrentVersion != nil {
		response.CurrentVersion = h.versionToResponse(template.CurrentVersion)
	}

	return response
}

func (h *SOPHandler) versionToResponse(version *models.SOPTemplateVersion) *dtos.SOPVersionResponseDTO {
	response := &dtos.SOPVersionResponseDTO{
		ID:            version.ID,
		VersionNumber: version.VersionNumber,
		Description:   version.Description,
		ChangeSummary: version.ChangeSummary,
		CreatedAt:     version.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		IsActive:      version.IsActive,
	}

	if version.Steps != nil && len(version.Steps) > 0 {
		var steps []dtos.SOPStepResponseDTO
		for _, step := range version.Steps {
			steps = append(steps, dtos.SOPStepResponseDTO{
				ID:                   step.ID,
				StepNumber:           step.StepNumber,
				Title:                step.Title,
				Instructions:         step.Instructions,
				EstimatedTimeMinutes: step.EstimatedTimeMinutes,
				ImageURL:             step.ImageURL,
				VideoURL:             step.VideoURL,
				RequiresApproval:     step.RequiresApproval,
			})
		}
		response.Steps = steps
	}

	return response
}
