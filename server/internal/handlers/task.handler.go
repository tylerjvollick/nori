package handlers

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
)

// TaskServiceInterface defines the methods needed by TaskHandler.
type TaskServiceInterface interface {
	CreateTask(spaceID uuid.UUID, createdByID uuid.UUID, dto *dtos.CreateTaskRequest) (*models.Task, error)
	GetTaskByID(id string) (*models.Task, error)
	ListTasks(filter repositories.TaskFilter) ([]models.Task, int64, error)
	UpdateTask(id string, dto *dtos.UpdateTaskRequest) (*models.Task, error)
	DeleteTask(id string) error
}

// TaskHandler handles HTTP requests for tasks.
type TaskHandler struct {
	taskService TaskServiceInterface
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(taskService TaskServiceInterface) *TaskHandler {
	return &TaskHandler{taskService: taskService}
}

// RegisterTaskRoutes registers task API routes on the Fiber app.
func (h *TaskHandler) RegisterTaskRoutes(app *fiber.App, middlewares ...fiber.Handler) {
	group := app.Group("/api/v1/tasks", middlewares...)

	group.Get("", h.ListTasks)
	group.Post("", h.CreateTask)
	group.Get("/:id", h.GetTask)
	group.Put("/:id", h.UpdateTask)
	group.Delete("/:id", h.DeleteTask)
}

// ListTasks returns a paginated list of tasks for the active space.
func (h *TaskHandler) ListTasks(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	if authDTO.ActiveSpaceID == nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "X-Space-ID header is required",
		})
	}

	filter := repositories.TaskFilter{
		SpaceID: authDTO.ActiveSpaceID,
	}

	// Parse optional query parameters
	if status := c.Query("status"); status != "" {
		s := models.TaskStatus(status)
		filter.Status = &s
	}
	if stationID := c.Query("stationId"); stationID != "" {
		if id, err := uuid.Parse(stationID); err == nil {
			filter.StationID = &id
		}
	}
	if parentID := c.Query("parentId"); parentID != "" {
		filter.ParentID = &parentID
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

	// Default limit
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

// CreateTask creates a new task in the active space.
func (h *TaskHandler) CreateTask(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	if authDTO.ActiveSpaceID == nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "X-Space-ID header is required",
		})
	}

	var dto dtos.CreateTaskRequest
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if dto.Title == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "title is required",
		})
	}

	task, err := h.taskService.CreateTask(*authDTO.ActiveSpaceID, authDTO.User.ID, &dto)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(dtos.TaskResponseFromModel(task))
}

// GetTask returns a single task by ID.
func (h *TaskHandler) GetTask(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	task, err := h.taskService.GetTaskByID(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(dtos.TaskResponseFromModel(task))
}

// UpdateTask updates an existing task.
func (h *TaskHandler) UpdateTask(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	var dto dtos.UpdateTaskRequest
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	task, err := h.taskService.UpdateTask(id, &dto)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(dtos.TaskResponseFromModel(task))
}

// DeleteTask deletes a task by ID.
func (h *TaskHandler) DeleteTask(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	if err := h.taskService.DeleteTask(id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusNoContent).Send(nil)
}
