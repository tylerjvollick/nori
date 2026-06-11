package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/services"
)

// --- Mock task material service ---

type mockTaskMaterialService struct {
	materials map[uuid.UUID]*models.TaskMaterial

	addErr    error
	listErr   error
	removed   *uuid.UUID
	nextDupe  bool
	notFound  bool
}

func newMockTaskMaterialService() *mockTaskMaterialService {
	return &mockTaskMaterialService{materials: make(map[uuid.UUID]*models.TaskMaterial)}
}

func (m *mockTaskMaterialService) Add(spaceID uuid.UUID, taskID string, req *dtos.CreateTaskMaterialRequest) (*models.TaskMaterial, error) {
	if m.addErr != nil {
		return nil, m.addErr
	}
	if m.nextDupe {
		return nil, services.ErrTaskMaterialDuplicate
	}
	if req.QuantityPerUnit.IsZero() || req.QuantityPerUnit.IsNegative() {
		return nil, &services.TaskMaterialValidationError{Message: "quantityPerUnit must be > 0"}
	}
	tm := &models.TaskMaterial{
		ID: uuid.New(), TaskID: taskID, MaterialID: req.MaterialID,
		QuantityPerUnit: req.QuantityPerUnit, Notes: req.Notes,
	}
	m.materials[tm.ID] = tm
	return tm, nil
}

func (m *mockTaskMaterialService) List(spaceID uuid.UUID, taskID string) ([]models.TaskMaterial, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.notFound {
		return nil, services.ErrTaskMaterialNotFound
	}
	var out []models.TaskMaterial
	for _, tm := range m.materials {
		if tm.TaskID == taskID {
			out = append(out, *tm)
		}
	}
	return out, nil
}

func (m *mockTaskMaterialService) Update(spaceID uuid.UUID, taskID string, id uuid.UUID, req *dtos.UpdateTaskMaterialRequest) (*models.TaskMaterial, error) {
	tm, ok := m.materials[id]
	if !ok {
		return nil, services.ErrTaskMaterialNotFound
	}
	if req.QuantityPerUnit != nil {
		if req.QuantityPerUnit.IsZero() || req.QuantityPerUnit.IsNegative() {
			return nil, &services.TaskMaterialValidationError{Message: "quantityPerUnit must be > 0"}
		}
		tm.QuantityPerUnit = *req.QuantityPerUnit
	}
	return tm, nil
}

func (m *mockTaskMaterialService) Remove(spaceID uuid.UUID, taskID string, id uuid.UUID) error {
	if _, ok := m.materials[id]; !ok {
		return services.ErrTaskMaterialNotFound
	}
	m.removed = &id
	delete(m.materials, id)
	return nil
}

func setupTaskMaterialApp(handler *TaskMaterialHandler, authDTO *dtos.AuthDTO, spaceID uuid.UUID) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("authDTO", authDTO)
		c.Locals("spaceID", spaceID)
		return c.Next()
	})
	spaceGroup := app.Group("/api/v1/spaces/:spaceId")
	handler.RegisterTaskMaterialRoutes(spaceGroup)
	return app
}

func taskMaterialURL(spaceID uuid.UUID, taskID, rest string) string {
	return "/api/v1/spaces/" + spaceID.String() + "/tasks/" + taskID + "/materials" + rest
}

func newTaskMaterialTestApp(t *testing.T) (*fiber.App, *mockTaskMaterialService, uuid.UUID) {
	t.Helper()
	spaceID := uuid.New()
	svc := newMockTaskMaterialService()
	app := setupTaskMaterialApp(NewTaskMaterialHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)
	return app, svc, spaceID
}

func TestAddTaskMaterial_Returns201(t *testing.T) {
	app, _, spaceID := newTaskMaterialTestApp(t)

	req := jsonRequest(http.MethodPost, taskMaterialURL(spaceID, "task-1", ""), dtos.CreateTaskMaterialRequest{
		MaterialID:      uuid.New(),
		QuantityPerUnit: decimal.NewFromInt(8),
	})
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var body dtos.TaskMaterialResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "task-1", body.TaskID)
	assert.True(t, body.QuantityPerUnit.Equal(decimal.NewFromInt(8)))
}

func TestAddTaskMaterial_DuplicateReturns409(t *testing.T) {
	app, svc, spaceID := newTaskMaterialTestApp(t)
	svc.nextDupe = true

	req := jsonRequest(http.MethodPost, taskMaterialURL(spaceID, "task-1", ""), dtos.CreateTaskMaterialRequest{
		MaterialID: uuid.New(), QuantityPerUnit: decimal.NewFromInt(1),
	})
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestAddTaskMaterial_NonPositiveReturns400(t *testing.T) {
	app, _, spaceID := newTaskMaterialTestApp(t)

	req := jsonRequest(http.MethodPost, taskMaterialURL(spaceID, "task-1", ""), dtos.CreateTaskMaterialRequest{
		MaterialID: uuid.New(), QuantityPerUnit: decimal.Zero,
	})
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestListTaskMaterials_ReturnsTaskMaterials(t *testing.T) {
	app, svc, spaceID := newTaskMaterialTestApp(t)
	_, err := svc.Add(spaceID, "task-1", &dtos.CreateTaskMaterialRequest{MaterialID: uuid.New(), QuantityPerUnit: decimal.NewFromInt(2)})
	require.NoError(t, err)

	resp, err := app.Test(jsonRequest(http.MethodGet, taskMaterialURL(spaceID, "task-1", ""), nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body []dtos.TaskMaterialResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Len(t, body, 1)
}

func TestListTaskMaterials_TaskNotFoundReturns404(t *testing.T) {
	app, svc, spaceID := newTaskMaterialTestApp(t)
	svc.notFound = true

	resp, err := app.Test(jsonRequest(http.MethodGet, taskMaterialURL(spaceID, "ghost", ""), nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestUpdateTaskMaterial_Success(t *testing.T) {
	app, svc, spaceID := newTaskMaterialTestApp(t)
	tm, err := svc.Add(spaceID, "task-1", &dtos.CreateTaskMaterialRequest{MaterialID: uuid.New(), QuantityPerUnit: decimal.NewFromInt(2)})
	require.NoError(t, err)

	newQty := decimal.NewFromInt(5)
	req := jsonRequest(http.MethodPut, taskMaterialURL(spaceID, "task-1", "/"+tm.ID.String()), dtos.UpdateTaskMaterialRequest{QuantityPerUnit: &newQty})
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body dtos.TaskMaterialResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.True(t, body.QuantityPerUnit.Equal(decimal.NewFromInt(5)))
}

func TestUpdateTaskMaterial_NotFoundReturns404(t *testing.T) {
	app, _, spaceID := newTaskMaterialTestApp(t)

	newQty := decimal.NewFromInt(5)
	req := jsonRequest(http.MethodPut, taskMaterialURL(spaceID, "task-1", "/"+uuid.NewString()), dtos.UpdateTaskMaterialRequest{QuantityPerUnit: &newQty})
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestUpdateTaskMaterial_InvalidIDReturns400(t *testing.T) {
	app, _, spaceID := newTaskMaterialTestApp(t)

	req := jsonRequest(http.MethodPut, taskMaterialURL(spaceID, "task-1", "/not-a-uuid"), dtos.UpdateTaskMaterialRequest{})
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRemoveTaskMaterial_Returns204(t *testing.T) {
	app, svc, spaceID := newTaskMaterialTestApp(t)
	tm, err := svc.Add(spaceID, "task-1", &dtos.CreateTaskMaterialRequest{MaterialID: uuid.New(), QuantityPerUnit: decimal.NewFromInt(2)})
	require.NoError(t, err)

	resp, err := app.Test(jsonRequest(http.MethodDelete, taskMaterialURL(spaceID, "task-1", "/"+tm.ID.String()), nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.NotNil(t, svc.removed)
}

func TestRemoveTaskMaterial_NotFoundReturns404(t *testing.T) {
	app, _, spaceID := newTaskMaterialTestApp(t)

	resp, err := app.Test(jsonRequest(http.MethodDelete, taskMaterialURL(spaceID, "task-1", "/"+uuid.NewString()), nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
