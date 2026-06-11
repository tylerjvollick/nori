package handlers

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/services"
)

// TaskMaterialServiceInterface defines the service methods needed by the handler.
type TaskMaterialServiceInterface interface {
	Add(spaceID uuid.UUID, taskID string, req *dtos.CreateTaskMaterialRequest) (*models.TaskMaterial, error)
	List(spaceID uuid.UUID, taskID string) ([]models.TaskMaterial, error)
	Update(spaceID uuid.UUID, taskID string, id uuid.UUID, req *dtos.UpdateTaskMaterialRequest) (*models.TaskMaterial, error)
	Remove(spaceID uuid.UUID, taskID string, id uuid.UUID) error
}

// TaskMaterialHandler handles HTTP requests for materials attached to tasks.
type TaskMaterialHandler struct {
	service TaskMaterialServiceInterface
}

func NewTaskMaterialHandler(service TaskMaterialServiceInterface) *TaskMaterialHandler {
	return &TaskMaterialHandler{service: service}
}

// RegisterTaskMaterialRoutes registers task-material routes under a space-scoped
// router. Expects :spaceId in the path and the RequireSpace middleware applied.
func (h *TaskMaterialHandler) RegisterTaskMaterialRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	group := router.Group("/tasks/:taskId/materials", middlewares...)

	group.Post("", h.AddMaterial)
	group.Get("", h.ListMaterials)
	group.Put("/:id", h.UpdateMaterial)
	group.Delete("/:id", h.RemoveMaterial)
}

func taskMaterialError(c *fiber.Ctx, err error) error {
	var validationErr *services.TaskMaterialValidationError
	switch {
	case errors.As(err, &validationErr):
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": validationErr.Message})
	case errors.Is(err, services.ErrTaskMaterialDuplicate):
		return c.Status(http.StatusConflict).JSON(fiber.Map{"error": "material already attached to task"})
	case errors.Is(err, services.ErrTaskMaterialNotFound):
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "task material not found"})
	default:
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
}

// AddMaterial handles POST /tasks/:taskId/materials.
func (h *TaskMaterialHandler) AddMaterial(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}
	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}
	taskID := c.Params("taskId")

	var req dtos.CreateTaskMaterialRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	tm, err := h.service.Add(spaceID, taskID, &req)
	if err != nil {
		return taskMaterialError(c, err)
	}
	return c.Status(http.StatusCreated).JSON(dtos.TaskMaterialResponseFromModel(tm))
}

// ListMaterials handles GET /tasks/:taskId/materials.
func (h *TaskMaterialHandler) ListMaterials(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}
	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}
	taskID := c.Params("taskId")

	materials, err := h.service.List(spaceID, taskID)
	if err != nil {
		return taskMaterialError(c, err)
	}

	resp := make([]dtos.TaskMaterialResponse, len(materials))
	for i := range materials {
		resp[i] = dtos.TaskMaterialResponseFromModel(&materials[i])
	}
	return c.JSON(resp)
}

// UpdateMaterial handles PUT /tasks/:taskId/materials/:id.
func (h *TaskMaterialHandler) UpdateMaterial(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}
	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}
	taskID := c.Params("taskId")
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid task material ID"})
	}

	var req dtos.UpdateTaskMaterialRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	tm, err := h.service.Update(spaceID, taskID, id, &req)
	if err != nil {
		return taskMaterialError(c, err)
	}
	return c.JSON(dtos.TaskMaterialResponseFromModel(tm))
}

// RemoveMaterial handles DELETE /tasks/:taskId/materials/:id.
func (h *TaskMaterialHandler) RemoveMaterial(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}
	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}
	taskID := c.Params("taskId")
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid task material ID"})
	}

	if err := h.service.Remove(spaceID, taskID, id); err != nil {
		return taskMaterialError(c, err)
	}
	return c.Status(http.StatusNoContent).Send(nil)
}
