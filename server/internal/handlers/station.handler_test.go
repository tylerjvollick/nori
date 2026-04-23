package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
)

// --- Mock station repository ---

type mockStationRepo struct {
	stations       []models.Station
	wipCounts      map[uuid.UUID]int
	maxOrder       int
	createErr      error
	getBySpaceErr  error
	getByIDErr     error
	updateErr      error
	deleteErr      error
	getMaxOrderErr error
	getWIPErr      error
	createdStation *models.Station
	updatedStation *models.Station
	deletedID      *uuid.UUID
}

func newMockStationRepo() *mockStationRepo {
	return &mockStationRepo{
		stations:  []models.Station{},
		wipCounts: make(map[uuid.UUID]int),
		maxOrder:  -1,
	}
}

func (m *mockStationRepo) Create(station *models.Station) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.createdStation = station
	m.stations = append(m.stations, *station)
	return nil
}

func (m *mockStationRepo) GetByID(id uuid.UUID) (*models.Station, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	for i := range m.stations {
		if m.stations[i].ID == id {
			return &m.stations[i], nil
		}
	}
	return nil, fmt.Errorf("station not found")
}

func (m *mockStationRepo) GetBySpaceID(spaceID uuid.UUID) ([]models.Station, error) {
	if m.getBySpaceErr != nil {
		return nil, m.getBySpaceErr
	}
	return m.stations, nil
}

func (m *mockStationRepo) Update(station *models.Station) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updatedStation = station
	return nil
}

func (m *mockStationRepo) Delete(id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deletedID = &id
	return nil
}

func (m *mockStationRepo) GetMaxDisplayOrder(spaceID uuid.UUID) (int, error) {
	if m.getMaxOrderErr != nil {
		return 0, m.getMaxOrderErr
	}
	return m.maxOrder, nil
}

func (m *mockStationRepo) GetWIPCounts(spaceID uuid.UUID) (map[uuid.UUID]int, error) {
	if m.getWIPErr != nil {
		return nil, m.getWIPErr
	}
	return m.wipCounts, nil
}

// --- Test helpers ---

// setupStationApp builds a Fiber app that simulates auth + RequireSpace middleware
// for station handler tests. The spaceID is injected into c.Locals("spaceID").
func setupStationApp(handler *StationHandler, authDTO *dtos.AuthDTO, spaceID uuid.UUID) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("authDTO", authDTO)
		c.Locals("spaceID", spaceID)
		return c.Next()
	})
	spaceGroup := app.Group("/api/v1/spaces/:spaceId")
	handler.RegisterStationRoutes(spaceGroup)
	return app
}

// adminAuthDTOWithSpace and userAuthDTOWithSpace are shared test helpers used by
// station and task_dep handler tests. They set ActiveSpaceID for handlers that
// still read it directly (e.g. task_dep), and pass spaceID to setupStationApp for
// handlers that use spaceIDFromPath.
func adminAuthDTOWithSpace(accountID, spaceID uuid.UUID) *dtos.AuthDTO {
	adminRole := models.RoleAdmin
	return &dtos.AuthDTO{
		User: models.User{
			ID:   uuid.New(),
			Role: &adminRole,
		},
		AccountID:     accountID,
		ActiveSpaceID: &spaceID,
	}
}

func userAuthDTOWithSpace(accountID, spaceID uuid.UUID) *dtos.AuthDTO {
	userRole := models.RoleUser
	return &dtos.AuthDTO{
		User: models.User{
			ID:   uuid.New(),
			Role: &userRole,
		},
		AccountID:     accountID,
		ActiveSpaceID: &spaceID,
	}
}

// stationURL builds a path under the space-scoped route prefix.
func stationURL(spaceID uuid.UUID, rest string) string {
	return "/api/v1/spaces/" + spaceID.String() + "/stations" + rest
}

// --- CreateStation tests ---

func TestCreateStation_AdminSuccess(t *testing.T) {
	repo := newMockStationRepo()
	repo.maxOrder = 2

	handler := NewStationHandler(repo)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "Milling",
	})
	req := httptest.NewRequest(http.MethodPost, stationURL(spaceID, ""), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result dtos.StationResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Milling", result.Name)
	assert.Equal(t, 3, result.DisplayOrder) // maxOrder(2) + 1
	assert.Equal(t, 1, result.WIPLimit)     // default
	assert.True(t, result.IsActive)
	assert.Nil(t, result.Description)

	// Verify repo received the correct station
	require.NotNil(t, repo.createdStation)
	assert.Equal(t, spaceID, repo.createdStation.SpaceID)
	assert.Equal(t, "Milling", repo.createdStation.Name)
}

func TestCreateStation_WithOptionalFields(t *testing.T) {
	repo := newMockStationRepo()
	repo.maxOrder = -1 // no existing stations

	handler := NewStationHandler(repo)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	desc := "The main assembly area"
	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Assembly",
		"description": desc,
		"wipLimit":    5,
		"bufferSize":  3,
	})
	req := httptest.NewRequest(http.MethodPost, stationURL(spaceID, ""), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result dtos.StationResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Assembly", result.Name)
	assert.Equal(t, 0, result.DisplayOrder) // -1 + 1
	assert.Equal(t, 5, result.WIPLimit)
	assert.Equal(t, 3, result.BufferSize)
	require.NotNil(t, result.Description)
	assert.Equal(t, desc, *result.Description)
}

func TestCreateStation_NonAdminForbidden(t *testing.T) {
	repo := newMockStationRepo()
	handler := NewStationHandler(repo)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := userAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	body, _ := json.Marshal(map[string]string{"name": "Milling"})
	req := httptest.NewRequest(http.MethodPost, stationURL(spaceID, ""), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestCreateStation_EmptyName(t *testing.T) {
	repo := newMockStationRepo()
	handler := NewStationHandler(repo)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	body, _ := json.Marshal(map[string]string{"name": ""})
	req := httptest.NewRequest(http.MethodPost, stationURL(spaceID, ""), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "name is required", result["error"])
}

func TestCreateStation_InvalidBody(t *testing.T) {
	repo := newMockStationRepo()
	handler := NewStationHandler(repo)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	req := httptest.NewRequest(http.MethodPost, stationURL(spaceID, ""), bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateStation_RepoCreateError(t *testing.T) {
	repo := newMockStationRepo()
	repo.createErr = errors.New("db error")

	handler := NewStationHandler(repo)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	body, _ := json.Marshal(map[string]string{"name": "Milling"})
	req := httptest.NewRequest(http.MethodPost, stationURL(spaceID, ""), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "failed to create station", result["error"])
}

func TestCreateStation_GetMaxDisplayOrderError(t *testing.T) {
	repo := newMockStationRepo()
	repo.getMaxOrderErr = errors.New("db error")

	handler := NewStationHandler(repo)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	body, _ := json.Marshal(map[string]string{"name": "Milling"})
	req := httptest.NewRequest(http.MethodPost, stationURL(spaceID, ""), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "failed to determine display order", result["error"])
}

// --- GetStation tests ---

func TestGetStation_Success(t *testing.T) {
	stationID := uuid.New()
	spaceID := uuid.New()
	desc := "Main assembly"

	repo := newMockStationRepo()
	repo.stations = []models.Station{
		{
			ID:           stationID,
			SpaceID:      spaceID,
			Name:         "Assembly",
			Description:  &desc,
			DisplayOrder: 0,
			WIPLimit:     3,
			BufferSize:   2,
			IsActive:     true,
		},
	}
	repo.wipCounts = map[uuid.UUID]int{stationID: 1}

	handler := NewStationHandler(repo)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	req := httptest.NewRequest(http.MethodGet, stationURL(spaceID, "/"+stationID.String()), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result dtos.StationResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, stationID, result.ID)
	assert.Equal(t, "Assembly", result.Name)
	assert.Equal(t, 3, result.WIPLimit)
	assert.Equal(t, 2, result.BufferSize)
	assert.Equal(t, 1, result.WIPCount)
	require.NotNil(t, result.Description)
	assert.Equal(t, desc, *result.Description)
}

func TestGetStation_NotFound(t *testing.T) {
	repo := newMockStationRepo()
	handler := NewStationHandler(repo)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	req := httptest.NewRequest(http.MethodGet, stationURL(spaceID, "/"+uuid.New().String()), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestGetStation_WrongSpace(t *testing.T) {
	stationID := uuid.New()
	otherSpaceID := uuid.New()

	repo := newMockStationRepo()
	repo.stations = []models.Station{
		{
			ID:      stationID,
			SpaceID: otherSpaceID, // different space
			Name:    "Assembly",
		},
	}

	handler := NewStationHandler(repo)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	req := httptest.NewRequest(http.MethodGet, stationURL(spaceID, "/"+stationID.String()), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestGetStation_InvalidID(t *testing.T) {
	repo := newMockStationRepo()
	handler := NewStationHandler(repo)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	req := httptest.NewRequest(http.MethodGet, stationURL(spaceID, "/not-a-uuid"), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "invalid station ID", result["error"])
}

// --- UpdateStation tests ---

func TestUpdateStation_Success(t *testing.T) {
	stationID := uuid.New()
	spaceID := uuid.New()

	repo := newMockStationRepo()
	repo.stations = []models.Station{
		{
			ID:           stationID,
			SpaceID:      spaceID,
			Name:         "Assembly",
			DisplayOrder: 0,
			WIPLimit:     1,
			IsActive:     true,
		},
	}
	repo.wipCounts = map[uuid.UUID]int{stationID: 2}

	handler := NewStationHandler(repo)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	newName := "Assembly v2"
	newWIP := 5
	newBuffer := 3
	body, _ := json.Marshal(map[string]interface{}{
		"name":       newName,
		"wipLimit":   newWIP,
		"bufferSize": newBuffer,
	})
	req := httptest.NewRequest(http.MethodPut, stationURL(spaceID, "/"+stationID.String()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result dtos.StationResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Assembly v2", result.Name)
	assert.Equal(t, 5, result.WIPLimit)
	assert.Equal(t, 3, result.BufferSize)
	assert.Equal(t, 2, result.WIPCount)

	// Verify repo received the update
	require.NotNil(t, repo.updatedStation)
	assert.Equal(t, "Assembly v2", repo.updatedStation.Name)
	assert.Equal(t, 5, repo.updatedStation.WIPLimit)
}

func TestUpdateStation_PartialUpdate(t *testing.T) {
	stationID := uuid.New()
	spaceID := uuid.New()
	desc := "Original desc"

	repo := newMockStationRepo()
	repo.stations = []models.Station{
		{
			ID:           stationID,
			SpaceID:      spaceID,
			Name:         "Assembly",
			Description:  &desc,
			DisplayOrder: 0,
			WIPLimit:     3,
			BufferSize:   1,
			IsActive:     true,
		},
	}
	repo.wipCounts = map[uuid.UUID]int{}

	handler := NewStationHandler(repo)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	// Only update isActive
	body, _ := json.Marshal(map[string]interface{}{
		"isActive": false,
	})
	req := httptest.NewRequest(http.MethodPut, stationURL(spaceID, "/"+stationID.String()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result dtos.StationResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Assembly", result.Name) // unchanged
	assert.Equal(t, 3, result.WIPLimit)      // unchanged
	assert.Equal(t, 1, result.BufferSize)    // unchanged
	assert.False(t, result.IsActive)         // updated
	require.NotNil(t, result.Description)
	assert.Equal(t, desc, *result.Description) // unchanged
}

func TestUpdateStation_NonAdminForbidden(t *testing.T) {
	stationID := uuid.New()
	spaceID := uuid.New()

	repo := newMockStationRepo()
	repo.stations = []models.Station{
		{ID: stationID, SpaceID: spaceID, Name: "Assembly"},
	}

	handler := NewStationHandler(repo)
	accountID := uuid.New()
	auth := userAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	body, _ := json.Marshal(map[string]interface{}{"name": "New Name"})
	req := httptest.NewRequest(http.MethodPut, stationURL(spaceID, "/"+stationID.String()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestUpdateStation_NotFound(t *testing.T) {
	repo := newMockStationRepo()
	handler := NewStationHandler(repo)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	body, _ := json.Marshal(map[string]interface{}{"name": "New Name"})
	req := httptest.NewRequest(http.MethodPut, stationURL(spaceID, "/"+uuid.New().String()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestUpdateStation_RepoError(t *testing.T) {
	stationID := uuid.New()
	spaceID := uuid.New()

	repo := newMockStationRepo()
	repo.stations = []models.Station{
		{ID: stationID, SpaceID: spaceID, Name: "Assembly"},
	}
	repo.updateErr = errors.New("db error")

	handler := NewStationHandler(repo)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	body, _ := json.Marshal(map[string]interface{}{"name": "New Name"})
	req := httptest.NewRequest(http.MethodPut, stationURL(spaceID, "/"+stationID.String()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "failed to update station", result["error"])
}

// --- DeleteStation tests ---

func TestDeleteStation_Success(t *testing.T) {
	stationID := uuid.New()
	spaceID := uuid.New()

	repo := newMockStationRepo()
	repo.stations = []models.Station{
		{ID: stationID, SpaceID: spaceID, Name: "Assembly"},
	}

	handler := NewStationHandler(repo)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	req := httptest.NewRequest(http.MethodDelete, stationURL(spaceID, "/"+stationID.String()), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	require.NotNil(t, repo.deletedID)
	assert.Equal(t, stationID, *repo.deletedID)
}

func TestDeleteStation_NonAdminForbidden(t *testing.T) {
	stationID := uuid.New()
	spaceID := uuid.New()

	repo := newMockStationRepo()
	repo.stations = []models.Station{
		{ID: stationID, SpaceID: spaceID, Name: "Assembly"},
	}

	handler := NewStationHandler(repo)
	accountID := uuid.New()
	auth := userAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	req := httptest.NewRequest(http.MethodDelete, stationURL(spaceID, "/"+stationID.String()), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestDeleteStation_NotFound(t *testing.T) {
	repo := newMockStationRepo()
	handler := NewStationHandler(repo)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	req := httptest.NewRequest(http.MethodDelete, stationURL(spaceID, "/"+uuid.New().String()), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDeleteStation_WrongSpace(t *testing.T) {
	stationID := uuid.New()
	otherSpaceID := uuid.New()

	repo := newMockStationRepo()
	repo.stations = []models.Station{
		{ID: stationID, SpaceID: otherSpaceID, Name: "Assembly"},
	}

	handler := NewStationHandler(repo)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	req := httptest.NewRequest(http.MethodDelete, stationURL(spaceID, "/"+stationID.String()), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDeleteStation_RepoError(t *testing.T) {
	stationID := uuid.New()
	spaceID := uuid.New()

	repo := newMockStationRepo()
	repo.stations = []models.Station{
		{ID: stationID, SpaceID: spaceID, Name: "Assembly"},
	}
	repo.deleteErr = errors.New("db error")

	handler := NewStationHandler(repo)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	req := httptest.NewRequest(http.MethodDelete, stationURL(spaceID, "/"+stationID.String()), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "failed to delete station", result["error"])
}

// --- ListStations tests ---

func TestListStations_Success(t *testing.T) {
	stationID := uuid.New()
	spaceID := uuid.New()

	repo := newMockStationRepo()
	repo.stations = []models.Station{
		{
			ID:           stationID,
			SpaceID:      spaceID,
			Name:         "Assembly",
			DisplayOrder: 0,
			WIPLimit:     3,
			IsActive:     true,
		},
	}
	repo.wipCounts = map[uuid.UUID]int{stationID: 1}

	handler := NewStationHandler(repo)
	accountID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth, spaceID)

	req := httptest.NewRequest(http.MethodGet, stationURL(spaceID, ""), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []dtos.StationResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.Len(t, result, 1)
	assert.Equal(t, "Assembly", result[0].Name)
	assert.Equal(t, 1, result[0].WIPCount)
}
