package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
)

// SubTaskRepoInterface defines the repository methods needed by SubTaskHandler.
type SubTaskRepoInterface interface {
	Create(s *models.SubTask) error
	GetByID(id uuid.UUID) (*models.SubTask, error)
	ListByTaskID(taskID string) ([]models.SubTask, error)
	GetMaxDisplayOrder(taskID string) (int, error)
	Update(id uuid.UUID, updates map[string]interface{}) error
	Delete(id uuid.UUID) error
	Reorder(taskID string, ids []uuid.UUID) error
}

// SubTaskTaskServiceInterface defines the task service methods needed by
// SubTaskHandler for space-scoping validation.
type SubTaskTaskServiceInterface interface {
	GetTaskByID(id string) (*models.Task, error)
}

// SubTaskHandler handles HTTP requests for sub-tasks.
type SubTaskHandler struct {
	subTaskRepo SubTaskRepoInterface
	taskService SubTaskTaskServiceInterface
}

// NewSubTaskHandler creates a new SubTaskHandler.
func NewSubTaskHandler(subTaskRepo SubTaskRepoInterface, taskService SubTaskTaskServiceInterface) *SubTaskHandler {
	return &SubTaskHandler{subTaskRepo: subTaskRepo, taskService: taskService}
}

// RegisterSubTaskRoutes registers sub-task API routes under a space-scoped router.
// Routes are nested under /tasks/:taskId/subtasks.
func (h *SubTaskHandler) RegisterSubTaskRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	group := router.Group("/tasks/:taskId/subtasks", middlewares...)

	group.Get("", h.List)
	group.Post("", h.Create)
	group.Put("/reorder", h.Reorder) // must come before /:subtaskId
	group.Put("/:subtaskId", h.Update)
	group.Delete("/:subtaskId", h.Delete)
}

// getTaskInSpace fetches a task by ID and verifies it belongs to the space from
// the URL path. Returns 404 on mismatch to avoid leaking existence.
func (h *SubTaskHandler) getTaskInSpace(c *fiber.Ctx, taskID string) (*models.Task, error) {
	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return nil, errResponseSent
	}

	task, err := h.taskService.GetTaskByID(taskID)
	if err != nil {
		_ = c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "task not found",
		})
		return nil, errResponseSent
	}

	if task.SpaceID != spaceID {
		_ = c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "task not found",
		})
		return nil, errResponseSent
	}

	return task, nil
}

// getSubTaskForTask fetches a sub-task by UUID and verifies it belongs to the
// given task. Returns 404 on mismatch to avoid leaking existence.
func (h *SubTaskHandler) getSubTaskForTask(c *fiber.Ctx, subtaskID uuid.UUID, taskID string) (*models.SubTask, error) {
	st, err := h.subTaskRepo.GetByID(subtaskID)
	if err != nil {
		_ = c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "sub-task not found",
		})
		return nil, errResponseSent
	}

	if st.TaskID != taskID {
		_ = c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "sub-task not found",
		})
		return nil, errResponseSent
	}

	return st, nil
}

// List returns all sub-tasks for a task, ordered by display_order.
// GET /api/v1/spaces/:spaceId/tasks/:taskId/subtasks
func (h *SubTaskHandler) List(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	taskID := c.Params("taskId")
	if taskID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	if _, err := h.getTaskInSpace(c, taskID); err != nil {
		return nil
	}

	subtasks, err := h.subTaskRepo.ListByTaskID(taskID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list sub-tasks",
		})
	}

	items := make([]dtos.SubTaskResponse, len(subtasks))
	for i := range subtasks {
		items[i] = dtos.SubTaskResponseFromModel(&subtasks[i])
	}

	return c.JSON(dtos.SubTaskListResponse{
		Items: items,
		Total: len(items),
	})
}

// Create adds a new sub-task to a task.
// POST /api/v1/spaces/:spaceId/tasks/:taskId/subtasks
func (h *SubTaskHandler) Create(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	if !isAdmin(authDTO) {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "admin access required",
		})
	}

	taskID := c.Params("taskId")
	if taskID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	if _, err := h.getTaskInSpace(c, taskID); err != nil {
		return nil
	}

	var req dtos.CreateSubTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Title == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "title is required",
		})
	}

	displayOrder := 0
	if req.DisplayOrder != nil {
		displayOrder = *req.DisplayOrder
	} else {
		max, err := h.subTaskRepo.GetMaxDisplayOrder(taskID)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to determine display order",
			})
		}
		displayOrder = max + 1
	}

	st := &models.SubTask{
		TaskID:       taskID,
		Title:        req.Title,
		Description:  req.Description,
		DisplayOrder: displayOrder,
	}

	if err := h.subTaskRepo.Create(st); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create sub-task",
		})
	}

	return c.Status(http.StatusCreated).JSON(dtos.SubTaskResponseFromModel(st))
}

// Update modifies a sub-task's title or description.
// PUT /api/v1/spaces/:spaceId/tasks/:taskId/subtasks/:subtaskId
func (h *SubTaskHandler) Update(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	if !isAdmin(authDTO) {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "admin access required",
		})
	}

	taskID := c.Params("taskId")
	if taskID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	if _, err := h.getTaskInSpace(c, taskID); err != nil {
		return nil
	}

	subtaskID, err := uuid.Parse(c.Params("subtaskId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid sub-task ID",
		})
	}

	st, err := h.getSubTaskForTask(c, subtaskID, taskID)
	if err != nil {
		return nil
	}

	var req dtos.UpdateSubTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	updates := map[string]interface{}{}
	if req.Title != nil {
		if *req.Title == "" {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "title cannot be empty",
			})
		}
		updates["title"] = *req.Title
		st.Title = *req.Title
	}
	if req.Description != nil {
		updates["description"] = req.Description
		st.Description = req.Description
	}

	if len(updates) == 0 {
		return c.JSON(dtos.SubTaskResponseFromModel(st))
	}

	if err := h.subTaskRepo.Update(subtaskID, updates); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update sub-task",
		})
	}

	// Re-fetch to get accurate UpdatedAt timestamp.
	updated, err := h.subTaskRepo.GetByID(subtaskID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch updated sub-task",
		})
	}

	return c.JSON(dtos.SubTaskResponseFromModel(updated))
}

// Delete removes a sub-task.
// DELETE /api/v1/spaces/:spaceId/tasks/:taskId/subtasks/:subtaskId
func (h *SubTaskHandler) Delete(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	if !isAdmin(authDTO) {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "admin access required",
		})
	}

	taskID := c.Params("taskId")
	if taskID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	if _, err := h.getTaskInSpace(c, taskID); err != nil {
		return nil
	}

	subtaskID, err := uuid.Parse(c.Params("subtaskId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid sub-task ID",
		})
	}

	if _, err := h.getSubTaskForTask(c, subtaskID, taskID); err != nil {
		return nil
	}

	if err := h.subTaskRepo.Delete(subtaskID); err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "sub-task not found",
		})
	}

	return c.Status(http.StatusNoContent).Send(nil)
}

// Reorder updates display_order for all sub-tasks of a task according to the
// provided ordered ID list.
// PUT /api/v1/spaces/:spaceId/tasks/:taskId/subtasks/reorder
func (h *SubTaskHandler) Reorder(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	if !isAdmin(authDTO) {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "admin access required",
		})
	}

	taskID := c.Params("taskId")
	if taskID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	if _, err := h.getTaskInSpace(c, taskID); err != nil {
		return nil
	}

	var req dtos.ReorderSubTasksRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if len(req.IDs) == 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "ids is required",
		})
	}

	// Verify the count matches what's in the database to prevent partial reorders.
	existing, err := h.subTaskRepo.ListByTaskID(taskID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list sub-tasks",
		})
	}

	if len(req.IDs) != len(existing) {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "ids must include all sub-task IDs for the task",
		})
	}

	if err := h.subTaskRepo.Reorder(taskID, req.IDs); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Return the updated ordered list.
	subtasks, err := h.subTaskRepo.ListByTaskID(taskID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch reordered sub-tasks",
		})
	}

	items := make([]dtos.SubTaskResponse, len(subtasks))
	for i := range subtasks {
		items[i] = dtos.SubTaskResponseFromModel(&subtasks[i])
	}

	return c.JSON(dtos.SubTaskListResponse{
		Items: items,
		Total: len(items),
	})
}
