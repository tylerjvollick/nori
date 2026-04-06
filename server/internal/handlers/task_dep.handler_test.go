package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
)

// --- Mock task dep repository ---

type mockTaskDepRepo struct {
	deps             []models.TaskDep
	addDepErr        error
	getBlockersErr   error
	getDependentsErr error
	removeByIDErr    error
	addedDep         *models.TaskDep
	removedDepID     *uuid.UUID
}

func newMockTaskDepRepo() *mockTaskDepRepo {
	return &mockTaskDepRepo{
		deps: []models.TaskDep{},
	}
}

func (m *mockTaskDepRepo) AddDep(dep *models.TaskDep) error {
	if m.addDepErr != nil {
		return m.addDepErr
	}
	m.addedDep = dep
	m.deps = append(m.deps, *dep)
	return nil
}

func (m *mockTaskDepRepo) GetBlockers(taskID string) ([]models.TaskDep, error) {
	if m.getBlockersErr != nil {
		return nil, m.getBlockersErr
	}
	var result []models.TaskDep
	for _, d := range m.deps {
		if d.ToTaskID == taskID && d.Type == models.DepTypeBlocks {
			// Actually, GetBlockers returns where to_task_id = taskID
			// Wait: the repo returns deps where to_task_id = taskID AND type = blocks
			// But in our mock we simulate blockers = tasks blocking taskID
			result = append(result, d)
		}
	}
	return result, nil
}

func (m *mockTaskDepRepo) GetDependents(taskID string) ([]models.TaskDep, error) {
	if m.getDependentsErr != nil {
		return nil, m.getDependentsErr
	}
	var result []models.TaskDep
	for _, d := range m.deps {
		if d.FromTaskID == taskID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (m *mockTaskDepRepo) RemoveDepByID(id uuid.UUID) error {
	if m.removeByIDErr != nil {
		return m.removeByIDErr
	}
	m.removedDepID = &id
	for i, d := range m.deps {
		if d.ID == id {
			m.deps = append(m.deps[:i], m.deps[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("dependency %q not found", id)
}

// --- Mock task service for space validation ---

type mockTaskDepTaskService struct {
	tasks map[string]*models.Task
	err   error
}

func newMockTaskDepTaskService() *mockTaskDepTaskService {
	return &mockTaskDepTaskService{
		tasks: make(map[string]*models.Task),
	}
}

func (m *mockTaskDepTaskService) GetTaskByID(id string) (*models.Task, error) {
	if m.err != nil {
		return nil, m.err
	}
	task, ok := m.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found")
	}
	return task, nil
}

// --- Test helpers ---

func setupTaskDepApp(handler *TaskDepHandler, authDTO *dtos.AuthDTO) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("authDTO", authDTO)
		return c.Next()
	})
	handler.RegisterTaskDepRoutes(app)
	return app
}

// --- GetDeps tests ---

func TestGetDeps_Success(t *testing.T) {
	spaceID := uuid.New()
	taskID := "ORD-1.1"
	blockerTaskID := "ORD-1.0"

	depRepo := newMockTaskDepRepo()
	depRepo.deps = []models.TaskDep{
		{
			ID:         uuid.New(),
			FromTaskID: taskID,
			ToTaskID:   blockerTaskID,
			Type:       models.DepTypeBlocks,
			CreatedAt:  time.Now(),
		},
	}

	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+taskID+"/deps", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result dtos.TaskDepsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	// blockers = where to_task_id = taskID, but our mock dep has ToTaskID=blockerTaskID, so 0 blockers
	// dependents = where from_task_id = taskID, which matches our dep
	assert.Len(t, result.Blockers, 0)
	assert.Len(t, result.Dependents, 1)
	assert.Equal(t, blockerTaskID, result.Dependents[0].ToTaskID)
}

func TestGetDeps_WithBlockers(t *testing.T) {
	spaceID := uuid.New()
	taskID := "ORD-1.1"

	depRepo := newMockTaskDepRepo()
	// This dep means: taskID is blocked by "other-task"
	// GetBlockers(taskID) returns where to_task_id = taskID AND type = blocks
	depRepo.deps = []models.TaskDep{
		{
			ID:         uuid.New(),
			FromTaskID: "other-task",
			ToTaskID:   taskID,
			Type:       models.DepTypeBlocks,
			CreatedAt:  time.Now(),
		},
	}

	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+taskID+"/deps", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result dtos.TaskDepsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Len(t, result.Blockers, 1)
	assert.Equal(t, taskID, result.Blockers[0].ToTaskID)
	assert.Len(t, result.Dependents, 0)
}

func TestGetDeps_EmptyResults(t *testing.T) {
	spaceID := uuid.New()
	taskID := "ORD-1.1"

	depRepo := newMockTaskDepRepo()
	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+taskID+"/deps", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result dtos.TaskDepsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Len(t, result.Blockers, 0)
	assert.Len(t, result.Dependents, 0)
}

func TestGetDeps_TaskNotFound(t *testing.T) {
	depRepo := newMockTaskDepRepo()
	taskSvc := newMockTaskDepTaskService()

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/nonexistent/deps", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestGetDeps_WrongSpace(t *testing.T) {
	spaceID := uuid.New()
	otherSpaceID := uuid.New()
	taskID := "ORD-1.1"

	depRepo := newMockTaskDepRepo()
	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: otherSpaceID}

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+taskID+"/deps", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestGetDeps_GetBlockersError(t *testing.T) {
	spaceID := uuid.New()
	taskID := "ORD-1.1"

	depRepo := newMockTaskDepRepo()
	depRepo.getBlockersErr = errors.New("db error")

	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+taskID+"/deps", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "failed to get blockers", result["error"])
}

func TestGetDeps_GetDependentsError(t *testing.T) {
	spaceID := uuid.New()
	taskID := "ORD-1.1"

	depRepo := newMockTaskDepRepo()
	depRepo.getDependentsErr = errors.New("db error")

	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+taskID+"/deps", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "failed to get dependents", result["error"])
}

func TestGetDeps_NoSpaceID(t *testing.T) {
	depRepo := newMockTaskDepRepo()
	taskSvc := newMockTaskDepTaskService()

	handler := NewTaskDepHandler(depRepo, taskSvc)
	adminRole := models.RoleAdmin
	auth := &dtos.AuthDTO{
		User: models.User{
			ID:   uuid.New(),
			Role: &adminRole,
		},
		AccountID:     uuid.New(),
		ActiveSpaceID: nil,
	}
	app := setupTaskDepApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/ORD-1.1/deps", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "X-Space-ID header is required", result["error"])
}

// --- AddDep tests ---

func TestAddDep_AdminSuccess(t *testing.T) {
	spaceID := uuid.New()
	taskID := "ORD-1.0"
	targetTaskID := "ORD-1.1"

	depRepo := newMockTaskDepRepo()
	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}
	taskSvc.tasks[targetTaskID] = &models.Task{ID: targetTaskID, SpaceID: spaceID}

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	body, _ := json.Marshal(map[string]interface{}{
		"targetTaskId": targetTaskID,
		"type":         "blocks",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+taskID+"/deps", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result dtos.TaskDepResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	// :id (ORD-1.0) blocks targetTaskId (ORD-1.1) → FromTaskID=ORD-1.1, ToTaskID=ORD-1.0
	assert.Equal(t, targetTaskID, result.FromTaskID)
	assert.Equal(t, taskID, result.ToTaskID)
	assert.Equal(t, models.DepTypeBlocks, result.Type)

	// Verify repo received the correct dep
	require.NotNil(t, depRepo.addedDep)
	assert.Equal(t, targetTaskID, depRepo.addedDep.FromTaskID)
	assert.Equal(t, taskID, depRepo.addedDep.ToTaskID)
}

func TestAddDep_NonAdminForbidden(t *testing.T) {
	spaceID := uuid.New()
	taskID := "ORD-1.0"

	depRepo := newMockTaskDepRepo()
	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := userAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	body, _ := json.Marshal(map[string]interface{}{
		"targetTaskId": "ORD-1.1",
		"type":         "blocks",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+taskID+"/deps", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestAddDep_TaskNotFound(t *testing.T) {
	depRepo := newMockTaskDepRepo()
	taskSvc := newMockTaskDepTaskService()

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	body, _ := json.Marshal(map[string]interface{}{
		"targetTaskId": "ORD-1.1",
		"type":         "blocks",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/nonexistent/deps", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestAddDep_TargetTaskNotFound(t *testing.T) {
	spaceID := uuid.New()
	taskID := "ORD-1.0"

	depRepo := newMockTaskDepRepo()
	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}
	// target task does NOT exist

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	body, _ := json.Marshal(map[string]interface{}{
		"targetTaskId": "ORD-1.1",
		"type":         "blocks",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+taskID+"/deps", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestAddDep_TargetTaskWrongSpace(t *testing.T) {
	spaceID := uuid.New()
	otherSpaceID := uuid.New()
	taskID := "ORD-1.0"
	targetTaskID := "ORD-1.1"

	depRepo := newMockTaskDepRepo()
	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}
	taskSvc.tasks[targetTaskID] = &models.Task{ID: targetTaskID, SpaceID: otherSpaceID}

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	body, _ := json.Marshal(map[string]interface{}{
		"targetTaskId": targetTaskID,
		"type":         "blocks",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+taskID+"/deps", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestAddDep_MissingTargetTaskId(t *testing.T) {
	spaceID := uuid.New()
	taskID := "ORD-1.0"

	depRepo := newMockTaskDepRepo()
	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	body, _ := json.Marshal(map[string]interface{}{
		"type": "blocks",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+taskID+"/deps", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "targetTaskId is required", result["error"])
}

func TestAddDep_MissingType(t *testing.T) {
	spaceID := uuid.New()
	taskID := "ORD-1.0"

	depRepo := newMockTaskDepRepo()
	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	body, _ := json.Marshal(map[string]interface{}{
		"targetTaskId": "ORD-1.1",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+taskID+"/deps", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "type is required", result["error"])
}

func TestAddDep_InvalidBody(t *testing.T) {
	spaceID := uuid.New()
	taskID := "ORD-1.0"

	depRepo := newMockTaskDepRepo()
	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+taskID+"/deps", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAddDep_RepoAddError(t *testing.T) {
	spaceID := uuid.New()
	taskID := "ORD-1.0"
	targetTaskID := "ORD-1.1"

	depRepo := newMockTaskDepRepo()
	depRepo.addDepErr = errors.New("would create a cycle")

	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}
	taskSvc.tasks[targetTaskID] = &models.Task{ID: targetTaskID, SpaceID: spaceID}

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	body, _ := json.Marshal(map[string]interface{}{
		"targetTaskId": targetTaskID,
		"type":         "blocks",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+taskID+"/deps", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "would create a cycle", result["error"])
}

// --- RemoveDep tests ---

func TestRemoveDep_AdminSuccess(t *testing.T) {
	spaceID := uuid.New()
	taskID := "ORD-1.0"
	depID := uuid.New()

	depRepo := newMockTaskDepRepo()
	depRepo.deps = []models.TaskDep{
		{
			ID:         depID,
			FromTaskID: "ORD-1.1",
			ToTaskID:   taskID,
			Type:       models.DepTypeBlocks,
			CreatedAt:  time.Now(),
		},
	}

	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+taskID+"/deps/"+depID.String(), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestRemoveDep_NonAdminForbidden(t *testing.T) {
	spaceID := uuid.New()
	taskID := "ORD-1.0"
	depID := uuid.New()

	depRepo := newMockTaskDepRepo()
	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := userAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+taskID+"/deps/"+depID.String(), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestRemoveDep_TaskNotFound(t *testing.T) {
	depRepo := newMockTaskDepRepo()
	taskSvc := newMockTaskDepTaskService()

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	depID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/nonexistent/deps/"+depID.String(), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestRemoveDep_InvalidDepID(t *testing.T) {
	spaceID := uuid.New()
	taskID := "ORD-1.0"

	depRepo := newMockTaskDepRepo()
	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+taskID+"/deps/not-a-uuid", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "invalid dependency ID", result["error"])
}

func TestRemoveDep_DepNotFound(t *testing.T) {
	spaceID := uuid.New()
	taskID := "ORD-1.0"
	depID := uuid.New()

	depRepo := newMockTaskDepRepo()
	// No deps in the repo — RemoveDepByID will return not found

	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+taskID+"/deps/"+depID.String(), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "dependency not found", result["error"])
}

func TestRemoveDep_WrongSpace(t *testing.T) {
	spaceID := uuid.New()
	otherSpaceID := uuid.New()
	taskID := "ORD-1.0"
	depID := uuid.New()

	depRepo := newMockTaskDepRepo()
	taskSvc := newMockTaskDepTaskService()
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: otherSpaceID}

	handler := NewTaskDepHandler(depRepo, taskSvc)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskDepApp(handler, auth)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+taskID+"/deps/"+depID.String(), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
