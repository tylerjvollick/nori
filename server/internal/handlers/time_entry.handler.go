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

// TimeEntryServiceInterface defines the methods needed by TimeEntryHandler.
type TimeEntryServiceInterface interface {
	StartTimer(taskID string, spaceID uuid.UUID, userID uuid.UUID) (*models.TimeEntry, error)
	PauseTimer(taskID string) (*models.TimeEntry, error)
	ResumeTimer(taskID string) (*models.TimeEntry, error)
	StopTimer(taskID string) (*models.TimeEntry, error)
	CreateManualEntry(taskID string, spaceID uuid.UUID, userID uuid.UUID, dto *dtos.CreateTimeEntryRequest) (*models.TimeEntry, error)
	UpdateEntry(id uuid.UUID, dto *dtos.UpdateTimeEntryRequest) (*models.TimeEntry, error)
	DeleteEntry(id uuid.UUID) error
	ListEntries(taskID string) ([]models.TimeEntry, int, error)
	GetUnloggedTasks(spaceID uuid.UUID) ([]models.Task, error)
}

// TimeEntryHandler handles HTTP requests for time entries.
type TimeEntryHandler struct {
	timeEntryService TimeEntryServiceInterface
}

// NewTimeEntryHandler creates a new TimeEntryHandler.
func NewTimeEntryHandler(timeEntryService TimeEntryServiceInterface) *TimeEntryHandler {
	return &TimeEntryHandler{timeEntryService: timeEntryService}
}

// RegisterTimeEntryRoutes registers time entry API routes under a space-scoped router.
func (h *TimeEntryHandler) RegisterTimeEntryRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	group := router.Group("/tasks/:taskId/time-entries", middlewares...)

	group.Post("/start", h.StartTimer)
	group.Post("/pause", h.PauseTimer)
	group.Post("/resume", h.ResumeTimer)
	group.Post("/stop", h.StopTimer)
	group.Post("", h.CreateManualEntry)
	group.Get("", h.ListEntries)
	group.Put("/:entryId", h.UpdateEntry)
	group.Delete("/:entryId", h.DeleteEntry)

	// Unlogged tasks query (not task-scoped, but space-scoped).
	router.Get("/tasks/unlogged", h.GetUnloggedTasks)
}

func (h *TimeEntryHandler) StartTimer(c *fiber.Ctx) error {
	auth, err := requireAuth(c)
	if err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	taskID := c.Params("taskId")

	entry, err := h.timeEntryService.StartTimer(taskID, spaceID, auth.User.ID)
	if err != nil {
		if errors.Is(err, services.ErrTimerAlreadyRunning) {
			return c.Status(http.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(dtos.TimeEntryResponseFromModel(entry))
}

func (h *TimeEntryHandler) PauseTimer(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	taskID := c.Params("taskId")

	entry, err := h.timeEntryService.PauseTimer(taskID)
	if err != nil {
		if errors.Is(err, services.ErrNoTimerRunning) {
			return c.Status(http.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusOK).JSON(dtos.TimeEntryResponseFromModel(entry))
}

func (h *TimeEntryHandler) ResumeTimer(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	taskID := c.Params("taskId")

	entry, err := h.timeEntryService.ResumeTimer(taskID)
	if err != nil {
		if errors.Is(err, services.ErrNoTimerRunning) {
			return c.Status(http.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusOK).JSON(dtos.TimeEntryResponseFromModel(entry))
}

func (h *TimeEntryHandler) StopTimer(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	taskID := c.Params("taskId")

	entry, err := h.timeEntryService.StopTimer(taskID)
	if err != nil {
		if errors.Is(err, services.ErrNoTimerRunning) {
			return c.Status(http.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusOK).JSON(dtos.TimeEntryResponseFromModel(entry))
}

func (h *TimeEntryHandler) CreateManualEntry(c *fiber.Ctx) error {
	auth, err := requireAuth(c)
	if err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	taskID := c.Params("taskId")

	var dto dtos.CreateTimeEntryRequest
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	entry, err := h.timeEntryService.CreateManualEntry(taskID, spaceID, auth.User.ID, &dto)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(dtos.TimeEntryResponseFromModel(entry))
}

func (h *TimeEntryHandler) ListEntries(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	taskID := c.Params("taskId")

	entries, total, err := h.timeEntryService.ListEntries(taskID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	items := make([]dtos.TimeEntryResponse, len(entries))
	for i, e := range entries {
		items[i] = dtos.TimeEntryResponseFromModel(&e)
	}

	return c.Status(http.StatusOK).JSON(dtos.TimeEntryListResponse{
		Items:             items,
		TotalDurationSecs: total,
	})
}

func (h *TimeEntryHandler) UpdateEntry(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	entryID, err := uuid.Parse(c.Params("entryId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid entry ID"})
	}

	var dto dtos.UpdateTimeEntryRequest
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	entry, err := h.timeEntryService.UpdateEntry(entryID, &dto)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusOK).JSON(dtos.TimeEntryResponseFromModel(entry))
}

func (h *TimeEntryHandler) DeleteEntry(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	entryID, err := uuid.Parse(c.Params("entryId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid entry ID"})
	}

	if err := h.timeEntryService.DeleteEntry(entryID); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusNoContent).Send(nil)
}

func (h *TimeEntryHandler) GetUnloggedTasks(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	tasks, err := h.timeEntryService.GetUnloggedTasks(spaceID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	items := make([]dtos.TaskResponse, len(tasks))
	for i, t := range tasks {
		items[i] = dtos.TaskResponseFromModel(&t)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"items": items})
}
