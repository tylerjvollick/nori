package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/services"
)

// CostServiceInterface defines the methods needed by CostHandler.
type CostServiceInterface interface {
	ComputeTaskLaborCost(taskID string, createdByID uuid.UUID) (*services.LaborCostResult, error)
	ComputeJobLaborCost(jobID string, createdByID uuid.UUID) (*services.JobLaborCostResult, error)
	EstimateRecipeLaborCost(rootTaskID string, spaceID uuid.UUID) (*services.EstimatedLaborCost, error)
	GetTaskCosts(taskID string) ([]models.CostEntry, error)
	GetJobCostSummary(jobID string) (*services.CostSummary, error)
}

// CostHandler handles HTTP requests for cost computation.
type CostHandler struct {
	costService CostServiceInterface
}

// NewCostHandler creates a new CostHandler.
func NewCostHandler(costService CostServiceInterface) *CostHandler {
	return &CostHandler{costService: costService}
}

// RegisterCostRoutes registers cost API routes on the Fiber app.
func (h *CostHandler) RegisterCostRoutes(app *fiber.App, middlewares ...fiber.Handler) {
	group := app.Group("/api/v1/costs", middlewares...)

	group.Get("/tasks/:id", h.GetTaskCosts)
	group.Post("/tasks/:id/compute-labor", h.ComputeTaskLaborCost)
	group.Post("/jobs/:id/compute-labor", h.ComputeJobLaborCost)
	group.Get("/jobs/:id/cost-summary", h.GetJobCostSummary)
	group.Get("/recipes/:rootTaskId/estimate-labor", h.EstimateRecipeLaborCost)
}

// GetTaskCosts returns all cost entries for a task.
func (h *CostHandler) GetTaskCosts(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	taskID := c.Params("id")
	entries, err := h.costService.GetTaskCosts(taskID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(entries)
}

// ComputeTaskLaborCost computes and stores labor cost for a single task.
func (h *CostHandler) ComputeTaskLaborCost(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	taskID := c.Params("id")
	result, err := h.costService.ComputeTaskLaborCost(taskID, authDTO.User.ID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(result)
}

// ComputeJobLaborCost computes and stores labor costs for a job and all descendants.
func (h *CostHandler) ComputeJobLaborCost(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	jobID := c.Params("id")
	result, err := h.costService.ComputeJobLaborCost(jobID, authDTO.User.ID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(result)
}

// GetJobCostSummary returns the aggregated cost summary for a job.
func (h *CostHandler) GetJobCostSummary(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	jobID := c.Params("id")
	result, err := h.costService.GetJobCostSummary(jobID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(result)
}

// EstimateRecipeLaborCost computes estimated labor cost from recipe step times.
func (h *CostHandler) EstimateRecipeLaborCost(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	if authDTO.ActiveSpaceID == nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "X-Space-ID header is required",
		})
	}

	rootTaskID := c.Params("rootTaskId")
	result, err := h.costService.EstimateRecipeLaborCost(rootTaskID, *authDTO.ActiveSpaceID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(result)
}
