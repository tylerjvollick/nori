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

	// Step operation routes - work with template ID
	group.Post("/:id/steps", auth.AuthMiddleware(), h.CreateStep)
	group.Put("/:id/steps/:stepId", auth.AuthMiddleware(), h.UpdateStep)
	group.Delete("/:id/steps/:stepId", auth.AuthMiddleware(), h.DeleteStep)
	group.Patch("/:id/steps/:stepId/reorder", auth.AuthMiddleware(), h.ReorderSteps)

	// Version routes (must come before generic /:id route)
	group.Get("/versions/:versionId", auth.AuthMiddleware(), h.GetSOPVersion)

	// Public routes
	group.Get("/", auth.AuthMiddleware(), h.GetAllSOPs)
	group.Post("/", auth.AuthMiddleware(), h.CreateSOP)

	// Parameterized routes (must come after specific routes)
	group.Get("/:id", auth.AuthMiddleware(), h.GetSOP)
	group.Put("/:id", auth.AuthMiddleware(), h.UpdateSOP)
	group.Delete("/:id", auth.AuthMiddleware(), h.DeleteSOP)
	group.Get("/:id/versions", auth.AuthMiddleware(), h.GetSOPVersions)
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
		templateResponse := h.templateToResponse(&template)
		response = append(response, *templateResponse)
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

	if template.SpaceID != nil {
		s := template.SpaceID.String()
		response.SpaceID = &s
	}

	if template.CurrentVersion != nil {
		response.CurrentVersion = h.versionToResponse(template.CurrentVersion)
	}

	return response
}

func (h *SOPHandler) versionToResponse(version *models.SOPVersion) *dtos.SOPVersionResponseDTO {
	response := &dtos.SOPVersionResponseDTO{
		ID:            version.ID,
		VersionNumber: version.VersionNumber,
		Description:   version.Description,
		ChangeSummary: version.ChangeSummary,
		CreatedAt:     version.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     version.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		IsActive:      version.IsActive,
	}

	if version.Steps != nil && len(version.Steps) > 0 {
		var steps []dtos.SOPStepResponseDTO
		for _, step := range version.Steps {
			stepDTO := dtos.SOPStepResponseDTO{
				ID:                   step.ID,
				Order:                step.Order,
				Title:                step.Title,
				Instructions:         step.Instructions,
				EstimatedTimeMinutes: step.EstimatedTimeMinutes,
				ImageURL:             step.ImageURL,
				VideoURL:             step.VideoURL,
				RequiresApproval:     step.RequiresApproval,
				LinkedSOPTemplateID:  step.LinkedSOPTemplateID,
			}
			if step.StationID != nil {
				s := step.StationID.String()
				stepDTO.StationID = &s
			}
			steps = append(steps, stepDTO)
		}
		response.Steps = steps
	}

	return response
}

// Individual step operation handlers

// CreateStep creates a single step in the SOP (creates new version via auto-versioning)
func (h *SOPHandler) CreateStep(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	templateID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid template id",
		})
	}

	var dto dtos.CreateStepDTO
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	step, err := h.sopService.CreateStepForTemplate(templateID, &dto, authDTO.User.ID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := h.stepToResponse(step)
	return c.Status(http.StatusCreated).JSON(response)
}

// UpdateStep updates a single step in the SOP's current version
func (h *SOPHandler) UpdateStep(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	templateID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid template id",
		})
	}

	stepID, err := strconv.Atoi(c.Params("stepId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid step id",
		})
	}

	var dto dtos.UpdateStepDTO
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	step, err := h.sopService.UpdateStepForTemplate(templateID, stepID, &dto, authDTO.User.ID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := h.stepToResponse(step)
	return c.Status(http.StatusOK).JSON(response)
}

// DeleteStep deletes a single step from the SOP's current version
func (h *SOPHandler) DeleteStep(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	templateID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid template id",
		})
	}

	stepID, err := strconv.Atoi(c.Params("stepId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid step id",
		})
	}

	if err := h.sopService.DeleteStepForTemplate(templateID, stepID, authDTO.User.ID); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusNoContent).Send(nil)
}

// ReorderSteps reorders a single step in the SOP's current version
func (h *SOPHandler) ReorderSteps(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	templateID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid template id",
		})
	}

	stepID, err := strconv.Atoi(c.Params("stepId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid step id",
		})
	}

	var dto dtos.ReorderStepDTO
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	step, err := h.sopService.ReorderStepForTemplate(templateID, stepID, &dto, authDTO.User.ID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := h.stepToResponse(step)
	return c.Status(http.StatusOK).JSON(response)
}

// Helper function to convert a step model to response DTO
func (h *SOPHandler) stepToResponse(step *models.SOPStep) *dtos.SOPStepResponseDTO {
	resp := &dtos.SOPStepResponseDTO{
		ID:                   step.ID,
		Order:                step.Order,
		Title:                step.Title,
		Instructions:         step.Instructions,
		EstimatedTimeMinutes: step.EstimatedTimeMinutes,
		ImageURL:             step.ImageURL,
		VideoURL:             step.VideoURL,
		RequiresApproval:     step.RequiresApproval,
		LinkedSOPTemplateID:  step.LinkedSOPTemplateID,
	}
	if step.StationID != nil {
		s := step.StationID.String()
		resp.StationID = &s
	}
	return resp
}
