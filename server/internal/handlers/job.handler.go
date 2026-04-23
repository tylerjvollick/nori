package handlers

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"github.com/tylerjvollick/nori/internal/services"
)

// SaveAsRecipeServiceInterface defines the save-as-recipe method from RecipeService.
type SaveAsRecipeServiceInterface interface {
	SaveAsRecipe(jobID string, spaceID uuid.UUID, createdByID uuid.UUID, name string, opts services.SaveAsRecipeOptions) (*models.Recipe, error)
}

// JobHandler handles HTTP requests for jobs (tasks with type=job).
type JobHandler struct {
	taskService         TaskServiceInterface
	costService         CostServiceInterface
	saveAsRecipeService SaveAsRecipeServiceInterface
}

// NewJobHandler creates a new JobHandler.
func NewJobHandler(taskService TaskServiceInterface, costService CostServiceInterface, saveAsRecipeService SaveAsRecipeServiceInterface) *JobHandler {
	return &JobHandler{
		taskService:         taskService,
		costService:         costService,
		saveAsRecipeService: saveAsRecipeService,
	}
}

// RegisterJobRoutes registers job API routes on a Fiber router (space-scoped group).
func (h *JobHandler) RegisterJobRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	group := router.Group("/jobs", middlewares...)

	group.Get("", h.ListJobs)
	group.Get("/:id", h.GetJob)
	group.Get("/:id/tasks", h.GetJobTasks)
	group.Get("/:id/cost-summary", h.GetJobCostSummary)
	group.Post("/:id/save-as-recipe", h.SaveJobAsRecipe)
}

// ListJobs returns a paginated list of root job tasks for the active space.
func (h *JobHandler) ListJobs(c *fiber.Ctx) error {
	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	jobType := models.TaskTypeJob
	filter := repositories.TaskFilter{
		SpaceID:  &spaceID,
		Type:     &jobType,
		RootOnly: true,
	}

	if status := c.Query("status"); status != "" {
		s := models.TaskStatus(status)
		filter.Status = &s
	}
	if offset := c.Query("offset"); offset != "" {
		if v, err := strconv.Atoi(offset); err == nil {
			filter.Offset = v
		}
	}
	if limit := c.Query("limit"); limit != "" {
		if v, err := strconv.Atoi(limit); err == nil {
			filter.Limit = v
		}
	}
	if filter.Limit == 0 {
		filter.Limit = 50
	}

	tasks, total, err := h.taskService.ListTasks(filter)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	items := make([]dtos.TaskResponse, len(tasks))
	for i := range tasks {
		items[i] = dtos.TaskResponseFromModel(&tasks[i])
	}

	return c.Status(http.StatusOK).JSON(dtos.TaskListResponse{
		Items:  items,
		Total:  total,
		Offset: filter.Offset,
		Limit:  filter.Limit,
	})
}

// getJobInSpace fetches a job by ID and verifies it belongs to the path-scoped
// space and is of type "job".
func (h *JobHandler) getJobInSpace(c *fiber.Ctx, jobID string) (*models.Task, error) {
	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return nil, err
	}

	task, err := h.taskService.GetTaskByID(jobID)
	if err != nil {
		return nil, c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "job not found",
		})
	}

	if task.SpaceID != spaceID {
		return nil, c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "job not found",
		})
	}

	if task.Type != models.TaskTypeJob {
		return nil, c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "job not found",
		})
	}

	return task, nil
}

// GetJob returns a single job by ID.
func (h *JobHandler) GetJob(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "job ID is required",
		})
	}

	job, err := h.getJobInSpace(c, id)
	if err != nil {
		return err
	}

	return c.Status(http.StatusOK).JSON(dtos.TaskResponseFromModel(job))
}

// GetJobTasks returns the task tree for a job.
func (h *JobHandler) GetJobTasks(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "job ID is required",
		})
	}

	job, err := h.getJobInSpace(c, id)
	if err != nil {
		return err
	}

	_, descendants, err := h.taskService.GetTaskTree(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	tree := dtos.BuildTaskTree(job, descendants)
	return c.Status(http.StatusOK).JSON(tree)
}

// GetJobCostSummary returns the aggregated cost summary for a job.
func (h *JobHandler) GetJobCostSummary(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "job ID is required",
		})
	}

	if _, err := h.getJobInSpace(c, id); err != nil {
		return err
	}

	result, err := h.costService.GetJobCostSummary(id)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(result)
}

// SaveJobAsRecipe saves a job as a new recipe by cloning its task tree.
func (h *JobHandler) SaveJobAsRecipe(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "job ID is required",
		})
	}

	if _, err := h.getJobInSpace(c, id); err != nil {
		return err
	}

	var dto dtos.SaveAsRecipeRequest
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if dto.Name == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "name is required",
		})
	}

	opts := services.SaveAsRecipeOptions{
		Description: dto.Description,
	}
	if dto.BackfillEstimatedFromActual != nil && *dto.BackfillEstimatedFromActual {
		opts.BackfillEstimatedFromActual = true
	}

	recipe, err := h.saveAsRecipeService.SaveAsRecipe(
		id,
		spaceID,
		authDTO.User.ID,
		dto.Name,
		opts,
	)
	if err != nil {
		return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(dtos.RecipeResponseFromModel(recipe))
}
