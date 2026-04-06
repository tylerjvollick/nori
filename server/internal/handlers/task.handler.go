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
	ClaimTask(taskID string, userID uuid.UUID) (*models.Task, error)
	CompleteTask(taskID string, userID uuid.UUID) (*models.Task, error)
	PauseTask(taskID string, userID uuid.UUID) (*models.Task, error)
}

// ReadyWorkServiceInterface defines the methods needed for ready-work queries.
type ReadyWorkServiceInterface interface {
	GetReadyTasks(spaceID uuid.UUID) ([]models.Task, error)
}

// TaskHandler handles HTTP requests for tasks.
type TaskHandler struct {
	taskService      TaskServiceInterface
	readyWorkService ReadyWorkServiceInterface
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(taskService TaskServiceInterface, readyWorkService ReadyWorkServiceInterface) *TaskHandler {
	return &TaskHandler{taskService: taskService, readyWorkService: readyWorkService}
}

// RegisterTaskRoutes registers task API routes on the Fiber app.
func (h *TaskHandler) RegisterTaskRoutes(app *fiber.App, middlewares ...fiber.Handler) {
	group := app.Group("/api/v1/tasks", middlewares...)

	group.Get("/ready", h.GetReadyTasks)
	group.Get("", h.ListTasks)
	group.Post("", h.CreateTask)
	group.Get("/:id", h.GetTask)
	group.Put("/:id", h.UpdateTask)
	group.Delete("/:id", h.DeleteTask)
	group.Post("/:id/claim", h.ClaimTask)
	group.Post("/:id/complete", h.CompleteTask)
	group.Post("/:id/pause", h.PauseTask)
}

// ListTasks returns a paginated list of tasks for the active space.
func (h *TaskHandler) ListTasks(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
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
	if assigneeID := c.Query("assigneeId"); assigneeID != "" {
		if id, err := uuid.Parse(assigneeID); err == nil {
			filter.AssigneeID = &id
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
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
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

// getTaskInSpace fetches a task by ID and verifies it belongs to the
// requester's active space. Returns 404 on mismatch (not 403) to avoid
// leaking existence of tasks in other spaces.
func (h *TaskHandler) getTaskInSpace(c *fiber.Ctx, authDTO *dtos.AuthDTO, taskID string) (*models.Task, error) {
	if authDTO.ActiveSpaceID == nil {
		return nil, c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "X-Space-ID header is required",
		})
	}

	task, err := h.taskService.GetTaskByID(taskID)
	if err != nil {
		return nil, c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "task not found",
		})
	}

	if task.SpaceID != *authDTO.ActiveSpaceID {
		return nil, c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "task not found",
		})
	}

	return task, nil
}

// GetTask returns a single task by ID.
func (h *TaskHandler) GetTask(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	task, err := h.getTaskInSpace(c, authDTO, id)
	if err != nil {
		return err
	}

	return c.Status(http.StatusOK).JSON(dtos.TaskResponseFromModel(task))
}

// UpdateTask updates an existing task.
func (h *TaskHandler) UpdateTask(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	// Verify task belongs to the requester's space before updating.
	if _, err := h.getTaskInSpace(c, authDTO, id); err != nil {
		return err
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
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	// Verify task belongs to the requester's space before deleting.
	if _, err := h.getTaskInSpace(c, authDTO, id); err != nil {
		return err
	}

	if err := h.taskService.DeleteTask(id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusNoContent).Send(nil)
}

// ClaimTask assigns the authenticated user to the task and sets it to active.
func (h *TaskHandler) ClaimTask(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	// Verify task belongs to the requester's space before claiming.
	if _, err := h.getTaskInSpace(c, authDTO, id); err != nil {
		return err
	}

	task, err := h.taskService.ClaimTask(id, authDTO.User.ID)
	if err != nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(dtos.TaskResponseFromModel(task))
}

// CompleteTask marks the task as done. Only the assigned user can complete it.
func (h *TaskHandler) CompleteTask(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	// Verify task belongs to the requester's space before completing.
	if _, err := h.getTaskInSpace(c, authDTO, id); err != nil {
		return err
	}

	task, err := h.taskService.CompleteTask(id, authDTO.User.ID)
	if err != nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(dtos.TaskResponseFromModel(task))
}

// PauseTask pauses an active task. Only the assigned user can pause it.
func (h *TaskHandler) PauseTask(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	// Verify task belongs to the requester's space before pausing.
	if _, err := h.getTaskInSpace(c, authDTO, id); err != nil {
		return err
	}

	task, err := h.taskService.PauseTask(id, authDTO.User.ID)
	if err != nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(dtos.TaskResponseFromModel(task))
}

// GetReadyTasks returns unblocked tasks for the active space, sorted by priority.
func (h *TaskHandler) GetReadyTasks(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	if authDTO.ActiveSpaceID == nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "X-Space-ID header is required",
		})
	}

	tasks, err := h.readyWorkService.GetReadyTasks(*authDTO.ActiveSpaceID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	items := make([]dtos.TaskResponse, len(tasks))
	for i := range tasks {
		items[i] = dtos.TaskResponseFromModel(&tasks[i])
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"items": items,
		"total": len(items),
	})
}
