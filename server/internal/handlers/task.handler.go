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

// TaskServiceInterface defines the methods needed by TaskHandler.
type TaskServiceInterface interface {
	CreateTask(spaceID uuid.UUID, createdByID uuid.UUID, dto *dtos.CreateTaskRequest) (*models.Task, error)
	GetTaskByID(id string) (*models.Task, error)
	GetTaskTree(id string) (*models.Task, []models.Task, error)
	ListTasks(filter repositories.TaskFilter) ([]models.Task, int64, error)
	UpdateTask(id string, dto *dtos.UpdateTaskRequest) (*models.Task, error)
	DeleteTask(id string) error
	StartTask(taskID string, userID uuid.UUID) (*models.Task, error)
	CompleteTask(taskID string, userID uuid.UUID, actualTimeSecs *int) (*services.CompleteTaskResult, error)
	PauseTask(taskID string, userID uuid.UUID) (*models.Task, error)
	ResumeTask(taskID string, userID uuid.UUID) (*models.Task, error)
	SkipTask(taskID string, userID uuid.UUID) (*models.Task, error)
	AddChildTask(parentID string, dto *dtos.AddChildTaskRequest, userID uuid.UUID) (*models.Task, error)
	AddNote(taskID string, text string) (*models.Task, error)
}

// ReadyWorkServiceInterface defines the methods needed for ready-work queries.
type ReadyWorkServiceInterface interface {
	GetReadyTasks(spaceID uuid.UUID, filter *services.ReadyTaskFilter) ([]models.Task, error)
}

// SaveAsRecipeServiceInterface defines the save-as-recipe method from RecipeService.
type SaveAsRecipeServiceInterface interface {
	SaveAsRecipe(jobID string, spaceID uuid.UUID, createdByID uuid.UUID, name string, opts services.SaveAsRecipeOptions) (*models.Recipe, error)
}

// TaskHandler handles HTTP requests for tasks.
type TaskHandler struct {
	taskService         TaskServiceInterface
	readyWorkService    ReadyWorkServiceInterface
	saveAsRecipeService SaveAsRecipeServiceInterface
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(taskService TaskServiceInterface, readyWorkService ReadyWorkServiceInterface, saveAsRecipeService SaveAsRecipeServiceInterface) *TaskHandler {
	return &TaskHandler{taskService: taskService, readyWorkService: readyWorkService, saveAsRecipeService: saveAsRecipeService}
}

// RegisterTaskRoutes registers task API routes on the Fiber app.
func (h *TaskHandler) RegisterTaskRoutes(app *fiber.App, middlewares ...fiber.Handler) {
	group := app.Group("/api/v1/tasks", middlewares...)

	group.Get("/ready", h.GetReadyTasks)
	group.Get("", h.ListTasks)
	group.Post("", h.CreateTask)
	group.Get("/:id", h.GetTask)
	group.Get("/:id/tree", h.GetTaskTree)
	group.Put("/:id", h.UpdateTask)
	group.Delete("/:id", h.DeleteTask)
	group.Post("/:id/start", h.StartTask)
	group.Post("/:id/complete", h.CompleteTask)
	group.Post("/:id/pause", h.PauseTask)
	group.Post("/:id/resume", h.ResumeTask)
	group.Post("/:id/skip", h.SkipTask)
	group.Post("/:id/children", h.AddChildTask)
	group.Post("/:id/notes", h.AddNote)
	group.Post("/:id/save-as-recipe", h.SaveAsRecipe)
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
	if taskType := c.Query("type"); taskType != "" {
		t := models.TaskType(taskType)
		filter.Type = &t
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

// GetTaskTree returns a task and all its descendants as a nested tree structure.
// Uses a single database query to fetch all descendants, then assembles the tree
// in memory — O(n) where n = total descendants.
func (h *TaskHandler) GetTaskTree(c *fiber.Ctx) error {
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

	// Verify root task belongs to the requester's space.
	root, err := h.getTaskInSpace(c, authDTO, id)
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
	if _, err := h.getTaskInSpace(c, authDTO, id); err != nil {
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
	if _, err := h.getTaskInSpace(c, authDTO, id); err != nil {
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

	return c.Status(http.StatusOK).JSON(resp)
}

// PauseTask pauses an active task.
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

// ResumeTask resumes a paused task.
func (h *TaskHandler) ResumeTask(c *fiber.Ctx) error {
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

	// Verify task belongs to the requester's space before resuming.
	if _, err := h.getTaskInSpace(c, authDTO, id); err != nil {
		return err
	}

	task, err := h.taskService.ResumeTask(id, authDTO.User.ID)
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
	if _, err := h.getTaskInSpace(c, authDTO, id); err != nil {
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
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	if authDTO.ActiveSpaceID == nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "X-Space-ID header is required",
		})
	}

	var filter services.ReadyTaskFilter
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

	tasks, err := h.readyWorkService.GetReadyTasks(*authDTO.ActiveSpaceID, &filter)
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
	if _, err := h.getTaskInSpace(c, authDTO, id); err != nil {
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
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(dtos.TaskResponseFromModel(task))
}

// AddNote appends a deviation note to the specified task.
func (h *TaskHandler) AddNote(c *fiber.Ctx) error {
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

	// Verify task belongs to the requester's space.
	if _, err := h.getTaskInSpace(c, authDTO, id); err != nil {
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

// SaveAsRecipe saves a job as a new recipe by cloning its task tree.
func (h *TaskHandler) SaveAsRecipe(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	if authDTO.ActiveSpaceID == nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "X-Space-ID header is required",
		})
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
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
		*authDTO.ActiveSpaceID,
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
