package handlers

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/storage"
)

// TaskMediaRepoInterface defines the repository methods needed by TaskMediaHandler.
type TaskMediaRepoInterface interface {
	Create(m *models.TaskMedia) error
	GetByID(id uuid.UUID) (*models.TaskMedia, error)
	ListByTaskID(taskID string) ([]models.TaskMedia, error)
	GetMaxDisplayOrder(taskID string) (int, error)
	Delete(id uuid.UUID) error
}

// TaskMediaHandler handles HTTP requests for task media attachments.
type TaskMediaHandler struct {
	mediaRepo   TaskMediaRepoInterface
	taskService SubTaskTaskServiceInterface
	store       *storage.LocalStorage
}

// NewTaskMediaHandler creates a new TaskMediaHandler.
func NewTaskMediaHandler(mediaRepo TaskMediaRepoInterface, taskService SubTaskTaskServiceInterface, store *storage.LocalStorage) *TaskMediaHandler {
	return &TaskMediaHandler{mediaRepo: mediaRepo, taskService: taskService, store: store}
}

// RegisterTaskMediaRoutes registers media routes nested under /tasks/:taskId/media.
func (h *TaskMediaHandler) RegisterTaskMediaRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	group := router.Group("/tasks/:taskId/media", middlewares...)

	group.Get("", h.List)
	group.Post("", h.Upload)
	group.Delete("/:mediaId", h.Delete)
}

// getTaskInSpace verifies the task exists and belongs to the path's space.
func (h *TaskMediaHandler) getTaskInSpace(c *fiber.Ctx, taskID string) (*models.Task, error) {
	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return nil, errResponseSent
	}

	task, err := h.taskService.GetTaskByID(taskID)
	if err != nil || task.SpaceID != spaceID {
		_ = c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "task not found"})
		return nil, errResponseSent
	}
	return task, nil
}

// List returns all media for a task, ordered by display_order.
// GET /api/v1/spaces/:spaceId/tasks/:taskId/media
func (h *TaskMediaHandler) List(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	taskID := c.Params("taskId")
	if taskID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "task ID is required"})
	}
	if _, err := h.getTaskInSpace(c, taskID); err != nil {
		return nil
	}

	media, err := h.mediaRepo.ListByTaskID(taskID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list media"})
	}

	items := make([]dtos.TaskMediaResponse, len(media))
	for i := range media {
		items[i] = dtos.TaskMediaResponseFromModel(&media[i])
	}
	return c.JSON(dtos.TaskMediaListResponse{Items: items, Total: len(items)})
}

// Upload attaches a photo/video to a task.
// POST /api/v1/spaces/:spaceId/tasks/:taskId/media  (multipart form, field "file")
func (h *TaskMediaHandler) Upload(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	taskID := c.Params("taskId")
	if taskID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "task ID is required"})
	}
	if _, err := h.getTaskInSpace(c, taskID); err != nil {
		return nil
	}

	fh, err := c.FormFile("file")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}

	saved, err := h.store.Save(fh, "task-media")
	if err != nil {
		var unsupported *storage.ErrUnsupportedMediaType
		if errors.As(err, &unsupported) {
			return c.Status(http.StatusUnsupportedMediaType).JSON(fiber.Map{"error": unsupported.Error()})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to store file"})
	}

	displayOrder := 0
	if max, err := h.mediaRepo.GetMaxDisplayOrder(taskID); err == nil {
		displayOrder = max + 1
	}

	capturedBy := authDTO.User.ID
	m := &models.TaskMedia{
		TaskID:       taskID,
		FilePath:     saved.URL,
		FileName:     saved.FileName,
		MimeType:     saved.MimeType,
		FileSize:     saved.Size,
		DisplayOrder: displayOrder,
		CapturedByID: &capturedBy,
	}
	if err := h.mediaRepo.Create(m); err != nil {
		_ = h.store.Delete(saved.URL) // best-effort cleanup of orphaned file
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save media record"})
	}

	return c.Status(http.StatusCreated).JSON(dtos.TaskMediaResponseFromModel(m))
}

// Delete removes a media attachment and its backing file.
// DELETE /api/v1/spaces/:spaceId/tasks/:taskId/media/:mediaId
func (h *TaskMediaHandler) Delete(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	taskID := c.Params("taskId")
	if taskID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "task ID is required"})
	}
	if _, err := h.getTaskInSpace(c, taskID); err != nil {
		return nil
	}

	mediaID, err := uuid.Parse(c.Params("mediaId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid media ID"})
	}

	m, err := h.mediaRepo.GetByID(mediaID)
	if err != nil || m.TaskID != taskID {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "media not found"})
	}

	if err := h.mediaRepo.Delete(mediaID); err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "media not found"})
	}
	_ = h.store.Delete(m.FilePath) // best-effort file removal

	return c.Status(http.StatusNoContent).Send(nil)
}
