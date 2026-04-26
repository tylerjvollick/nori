package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"github.com/tylerjvollick/nori/internal/services"
)

// TaskServiceInterface defines the methods needed by TaskHandler.
type TaskServiceInterface interface {
	CreateTask(spaceID uuid.UUID, createdByID uuid.UUID, dto *dtos.CreateTaskRequest) (*models.Task, error)
	GetTaskByID(id string) (*models.Task, error)
	GetTaskTree(id string) (*models.Task, []models.Task, error)
	ListTasks(filter repositories.TaskFilter) ([]models.Task, int64, error)
	UpdateTask(id string, dto *dtos.UpdateTaskRequest) (*models.Task, error)
	DeleteTask(id string) error
	SetTaskStatus(taskID string, userID uuid.UUID, newStatus models.TaskStatus) (*models.Task, error)
	StartTask(taskID string, userID uuid.UUID) (*models.Task, error)
	CompleteTask(taskID string, userID uuid.UUID, actualTimeSecs *int) (*services.CompleteTaskResult, error)
	SkipTask(taskID string, userID uuid.UUID) (*models.Task, error)
	AddChildTask(parentID string, dto *dtos.AddChildTaskRequest, userID uuid.UUID) (*models.Task, error)
	AddNote(taskID string, text string) (*models.Task, error)
}

// ReadyWorkServiceInterface defines the methods needed for ready-work queries.
type ReadyWorkServiceInterface interface {
	GetReadyTasks(spaceID uuid.UUID, filter *services.ReadyTaskFilter) ([]models.Task, error)
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

// RegisterTaskRoutes registers task API routes under a space-scoped router.
func (h *TaskHandler) RegisterTaskRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	group := router.Group("/tasks", middlewares...)

	group.Get("/ready", h.GetReadyTasks)
	group.Get("", h.ListTasks)
	group.Post("", h.CreateTask)
	group.Get("/:id", h.GetTask)
	group.Get("/:id/tree", h.GetTaskTree)
	group.Put("/:id", h.UpdateTask)
	group.Delete("/:id", h.DeleteTask)
	group.Put("/:id/status", h.SetStatus)
	group.Post("/:id/start", h.StartTask)
	group.Post("/:id/complete", h.CompleteTask)
	group.Post("/:id/skip", h.SkipTask)
	group.Post("/:id/children", h.AddChildTask)
	group.Post("/:id/notes", h.AddNote)
}

// ListTasks returns a paginated list of tasks for the active space.
func (h *TaskHandler) ListTasks(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	filter := repositories.TaskFilter{
		SpaceID: &spaceID,
	}

	// Parse optional query parameters
	if status := c.Query("status"); status != "" {
		s := models.TaskStatus(status)
		filter.Status = &s
	}
	if taskType := c.Query("type"); taskType != "" {
		t := models.TaskType(taskType)
		filter.Type = &t
	} else {
		// By default, exclude job and recipe root tasks and their descendants
		// from the tasks list. Use GET /api/v1/jobs for jobs and
		// GET /api/v1/recipes/:id/tasks for recipe trees.
		filter.ExcludeTypes = []models.TaskType{models.TaskTypeJob, models.TaskTypeRecipe}
		filter.ExcludeDescendantsOfTypes = []models.TaskType{models.TaskTypeRecipe}
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

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
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

	task, err := h.taskService.CreateTask(spaceID, authDTO.User.ID, &dto)
	if err != nil {
		if errors.Is(err, services.ErrMaxDepthExceeded) {
			return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(dtos.TaskResponseFromModel(task))
}

// getTaskInSpace fetches a task by ID and verifies it belongs to the
// space from the URL path. Returns 404 on mismatch (not 403) to avoid
// leaking existence of tasks in other spaces.
func (h *TaskHandler) getTaskInSpace(c *fiber.Ctx, taskID string) (*models.Task, error) {
	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return nil, err
	}

	task, err := h.taskService.GetTaskByID(taskID)
	if err != nil {
		return nil, c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "task not found",
		})
	}

	if task.SpaceID != spaceID {
		return nil, c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "task not found",
		})
	}

	return task, nil
}

// GetTask returns a single task by ID.
func (h *TaskHandler) GetTask(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	task, err := h.getTaskInSpace(c, id)
	if err != nil {
		return err
	}

	return c.Status(http.StatusOK).JSON(dtos.TaskResponseFromModel(task))
}

// GetTaskTree returns a task and all its descendants as a nested tree structure.
// Uses a single database query to fetch all descendants, then assembles the tree
// in memory — O(n) where n = total descendants.
func (h *TaskHandler) GetTaskTree(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	// Verify root task belongs to the requester's space.
	root, err := h.getTaskInSpace(c, id)
	if err != nil {
		return err
	}

	// Fetch root + all descendants in a single query.
	_, descendants, err := h.taskService.GetTaskTree(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	tree := dtos.BuildTaskTree(root, descendants)
	return c.Status(http.StatusOK).JSON(tree)
}

// UpdateTask updates an existing task.
func (h *TaskHandler) UpdateTask(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	// Verify task belongs to the requester's space before updating.
	if _, err := h.getTaskInSpace(c, id); err != nil {
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
	if _, err := requireAuth(c); err != nil {
		return err
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	// Verify task belongs to the requester's space before deleting.
	if _, err := h.getTaskInSpace(c, id); err != nil {
		return err
	}

	if err := h.taskService.DeleteTask(id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusNoContent).Send(nil)
}

// StartTask transitions a task from open to active.
func (h *TaskHandler) StartTask(c *fiber.Ctx) error {
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

	// Verify task belongs to the requester's space before starting.
	if _, err := h.getTaskInSpace(c, id); err != nil {
		return err
	}

	task, err := h.taskService.StartTask(id, authDTO.User.ID)
	if err != nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(dtos.TaskResponseFromModel(task))
}

// CompleteTask marks the task as done.
// Accepts an optional JSON body with actualTimeSeconds for time correction.
// Returns the completed task plus a nextTaskId hint for navigation.
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
	if _, err := h.getTaskInSpace(c, id); err != nil {
		return err
	}

	// Parse optional request body for time correction.
	var dto dtos.CompleteTaskRequest
	// BodyParser may fail if no body is sent — that's fine, all fields are optional.
	_ = c.BodyParser(&dto)

	result, err := h.taskService.CompleteTask(id, authDTO.User.ID, dto.ActualTimeSecs)
	if err != nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	resp := dtos.CompleteTaskResponse{
		TaskResponse:  dtos.TaskResponseFromModel(result.Task),
		NextTaskId:    result.NextTaskID,
		NextTaskTitle: result.NextTaskTitle,
	}

	// Include unresolved blockers if any.
	for _, b := range result.UnresolvedBlockers {
		resp.UnresolvedBlockers = append(resp.UnresolvedBlockers, dtos.UnresolvedBlockerResponse{
			ID:     b.ID,
			Title:  b.Title,
			Status: string(b.Status),
		})
	}

	return c.Status(http.StatusOK).JSON(resp)
}

// SetStatus changes the task status via PUT /tasks/:id/status.
func (h *TaskHandler) SetStatus(c *fiber.Ctx) error {
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

	if _, err := h.getTaskInSpace(c, id); err != nil {
		return err
	}

	var dto dtos.SetStatusRequest
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	newStatus := models.TaskStatus(dto.Status)
	task, err := h.taskService.SetTaskStatus(id, authDTO.User.ID, newStatus)
	if err != nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(dtos.TaskResponseFromModel(task))
}

// SkipTask marks a task as skipped.
func (h *TaskHandler) SkipTask(c *fiber.Ctx) error {
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

	// Verify task belongs to the requester's space before skipping.
	if _, err := h.getTaskInSpace(c, id); err != nil {
		return err
	}

	task, err := h.taskService.SkipTask(id, authDTO.User.ID)
	if err != nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(dtos.TaskResponseFromModel(task))
}

// GetReadyTasks returns unblocked tasks for the active space, sorted by priority.
// Supports optional query parameters:
//   - ?stationId=uuid — filter by station
//   - ?assigneeId=uuid — filter by assigned user
func (h *TaskHandler) GetReadyTasks(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	// By default, ready work returns only type=task and excludes descendants
	// of job/recipe roots.
	taskType := models.TaskTypeTask
	var filter services.ReadyTaskFilter
	filter.Type = &taskType
	filter.ExcludeDescendantsOfTypes = []models.TaskType{models.TaskTypeRecipe}
	if stationID := c.Query("stationId"); stationID != "" {
		if id, err := uuid.Parse(stationID); err == nil {
			filter.StationID = &id
		}
	}
	if assigneeID := c.Query("assigneeId"); assigneeID != "" {
		if id, err := uuid.Parse(assigneeID); err == nil {
			filter.AssignedToID = &id
		}
	}

	tasks, err := h.readyWorkService.GetReadyTasks(spaceID, &filter)
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

// AddChildTask creates a new child task under the specified parent task.
func (h *TaskHandler) AddChildTask(c *fiber.Ctx) error {
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

	// Verify parent task belongs to the requester's space.
	if _, err := h.getTaskInSpace(c, id); err != nil {
		return err
	}

	var dto dtos.AddChildTaskRequest
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

	task, err := h.taskService.AddChildTask(id, &dto, authDTO.User.ID)
	if err != nil {
		if errors.Is(err, services.ErrMaxDepthExceeded) {
			return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(dtos.TaskResponseFromModel(task))
}

// AddNote appends a deviation note to the specified task.
func (h *TaskHandler) AddNote(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	// Verify task belongs to the requester's space.
	if _, err := h.getTaskInSpace(c, id); err != nil {
		return err
	}

	var dto dtos.AddNoteRequest
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if dto.Text == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "text is required",
		})
	}

	task, err := h.taskService.AddNote(id, dto.Text)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(dtos.TaskResponseFromModel(task))
}

