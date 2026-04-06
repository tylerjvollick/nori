package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
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
	getMaxOrderErr error
	getWIPErr      error
	createdStation *models.Station
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

func (m *mockStationRepo) GetBySpaceID(spaceID uuid.UUID) ([]models.Station, error) {
	if m.getBySpaceErr != nil {
		return nil, m.getBySpaceErr
	}
	return m.stations, nil
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

func setupStationApp(handler *StationHandler, authDTO *dtos.AuthDTO) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("authDTO", authDTO)
		return c.Next()
	})
	handler.RegisterStationRoutes(app)
	return app
}

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

// --- CreateStation tests ---

func TestCreateStation_AdminSuccess(t *testing.T) {
	repo := newMockStationRepo()
	repo.maxOrder = 2

	handler := NewStationHandler(repo)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "Milling",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result stationResponse
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
	app := setupStationApp(handler, auth)

	desc := "The main assembly area"
	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Assembly",
		"description": desc,
		"wipLimit":    5,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result stationResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Assembly", result.Name)
	assert.Equal(t, 0, result.DisplayOrder) // -1 + 1
	assert.Equal(t, 5, result.WIPLimit)
	require.NotNil(t, result.Description)
	assert.Equal(t, desc, *result.Description)
}

func TestCreateStation_NonAdminForbidden(t *testing.T) {
	repo := newMockStationRepo()
	handler := NewStationHandler(repo)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := userAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth)

	body, _ := json.Marshal(map[string]string{"name": "Milling"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestCreateStation_NoSpaceID(t *testing.T) {
	repo := newMockStationRepo()
	handler := NewStationHandler(repo)
	accountID := uuid.New()
	// Admin with no active space
	adminRole := models.RoleAdmin
	auth := &dtos.AuthDTO{
		User: models.User{
			ID:   uuid.New(),
			Role: &adminRole,
		},
		AccountID:     accountID,
		ActiveSpaceID: nil,
	}
	app := setupStationApp(handler, auth)

	body, _ := json.Marshal(map[string]string{"name": "Milling"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "X-Space-ID header is required", result["error"])
}

func TestCreateStation_EmptyName(t *testing.T) {
	repo := newMockStationRepo()
	handler := NewStationHandler(repo)
	accountID := uuid.New()
	spaceID := uuid.New()
	auth := adminAuthDTOWithSpace(accountID, spaceID)
	app := setupStationApp(handler, auth)

	body, _ := json.Marshal(map[string]string{"name": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stations", bytes.NewReader(body))
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
	app := setupStationApp(handler, auth)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/stations", bytes.NewReader([]byte("not json")))
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
	app := setupStationApp(handler, auth)

	body, _ := json.Marshal(map[string]string{"name": "Milling"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stations", bytes.NewReader(body))
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
	app := setupStationApp(handler, auth)

	body, _ := json.Marshal(map[string]string{"name": "Milling"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "failed to determine display order", result["error"])
}
