package handlers

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
)

// errResponseSent is a sentinel error indicating the JSON response was already
// written to the Fiber context. Callers should return nil to Fiber so it does
// not attempt to write another response.
var errResponseSent = errors.New("response already sent")

// TaskDepRepoInterface defines the repository methods needed by TaskDepHandler.
type TaskDepRepoInterface interface {
	AddDep(dep *models.TaskDep) error
	GetBlockers(taskID string) ([]models.TaskDep, error)
	GetDependents(taskID string) ([]models.TaskDep, error)
	RemoveDepByID(id uuid.UUID) error
}

// TaskDepTaskServiceInterface defines the task service methods needed by
// TaskDepHandler for space-scoping validation.
type TaskDepTaskServiceInterface interface {
	GetTaskByID(id string) (*models.Task, error)
}

// TaskDepHandler handles HTTP requests for task dependencies.
type TaskDepHandler struct {
	taskDepRepo TaskDepRepoInterface
	taskService TaskDepTaskServiceInterface
}

// NewTaskDepHandler creates a new TaskDepHandler.
func NewTaskDepHandler(taskDepRepo TaskDepRepoInterface, taskService TaskDepTaskServiceInterface) *TaskDepHandler {
	return &TaskDepHandler{taskDepRepo: taskDepRepo, taskService: taskService}
}

// RegisterTaskDepRoutes registers task dependency API routes under a space-scoped router.
// Routes are nested under /tasks/:id/deps.
func (h *TaskDepHandler) RegisterTaskDepRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	group := router.Group("/tasks/:id/deps", middlewares...)

	group.Get("", h.GetDeps)
	group.Post("", h.AddDep)
	group.Delete("/:depId", h.RemoveDep)
}

// getTaskInSpace fetches a task by ID and verifies it belongs to the
// space from the URL path. Returns 404 on mismatch to avoid leaking existence.
// On failure, the JSON error response is written to c and a non-nil error is
// returned so the caller can short-circuit.
func (h *TaskDepHandler) getTaskInSpace(c *fiber.Ctx, taskID string) (*models.Task, error) {
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

// GetDeps returns all dependencies for a task (both directions).
// GET /api/v1/tasks/:id/deps
func (h *TaskDepHandler) GetDeps(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	if _, err := h.getTaskInSpace(c, id); err != nil {
		return nil
	}

	blockers, err := h.taskDepRepo.GetBlockers(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get blockers",
		})
	}

	dependents, err := h.taskDepRepo.GetDependents(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get dependents",
		})
	}

	blockerDTOs := make([]dtos.TaskDepResponse, len(blockers))
	for i := range blockers {
		blockerDTOs[i] = dtos.TaskDepResponseFromModel(&blockers[i])
	}

	dependentDTOs := make([]dtos.TaskDepResponse, len(dependents))
	for i := range dependents {
		dependentDTOs[i] = dtos.TaskDepResponseFromModel(&dependents[i])
	}

	return c.JSON(dtos.TaskDepsResponse{
		Blockers:   blockerDTOs,
		Dependents: dependentDTOs,
	})
}

// AddDep creates a new dependency edge.
// POST /api/v1/tasks/:id/deps
// The task at :id blocks targetTaskId (targetTaskId depends on :id).
func (h *TaskDepHandler) AddDep(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	// Admin only for mutations.
	if !isAdmin(authDTO) {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "admin access required",
		})
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	if _, err := h.getTaskInSpace(c, id); err != nil {
		return nil
	}

	var req dtos.AddDepRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.TargetTaskID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "targetTaskId is required",
		})
	}

	if req.Type == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "type is required",
		})
	}

	// Verify the target task also belongs to the same space.
	if _, err := h.getTaskInSpace(c, req.TargetTaskID); err != nil {
		return nil
	}

	// :id blocks targetTaskId → FromTaskID=targetTaskId, ToTaskID=:id
	dep := &models.TaskDep{
		ID:         uuid.New(),
		FromTaskID: req.TargetTaskID,
		ToTaskID:   id,
		Type:       req.Type,
	}

	if err := h.taskDepRepo.AddDep(dep); err != nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(dtos.TaskDepResponseFromModel(dep))
}

// RemoveDep removes a dependency edge by its UUID.
// DELETE /api/v1/tasks/:id/deps/:depId
func (h *TaskDepHandler) RemoveDep(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	// Admin only for mutations.
	if !isAdmin(authDTO) {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "admin access required",
		})
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "task ID is required",
		})
	}

	if _, err := h.getTaskInSpace(c, id); err != nil {
		return nil
	}

	depID, err := uuid.Parse(c.Params("depId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid dependency ID",
		})
	}

	if err := h.taskDepRepo.RemoveDepByID(depID); err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "dependency not found",
		})
	}

	return c.Status(http.StatusNoContent).Send(nil)
}
