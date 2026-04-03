package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/middleware"
	"github.com/tylerjvollick/nori/internal/models"
)

// --- Mock service and repository ---

type mockAPIKeyService struct {
	generateErr     error
	generatedKey    *models.APIKey
	generatedRawKey string
}

func (m *mockAPIKeyService) GenerateAPIKey(accountID uuid.UUID, name string, createdByID uuid.UUID, expiresAt *time.Time) (string, *models.APIKey, error) {
	if m.generateErr != nil {
		return "", nil, m.generateErr
	}
	key := m.generatedKey
	if key == nil {
		key = &models.APIKey{
			ID:          uuid.New(),
			AccountID:   accountID,
			Name:        name,
			KeyHash:     "hashed_value",
			IsActive:    true,
			CreatedAt:   time.Now(),
			CreatedByID: createdByID,
			ExpiresAt:   expiresAt,
		}
	}
	rawKey := m.generatedRawKey
	if rawKey == "" {
		rawKey = "nori_testapikey123456"
	}
	return rawKey, key, nil
}

type mockAPIKeyRepo struct {
	keys          []models.APIKey
	getByAccErr   error
	deactivateErr error
	deleteErr     error
	deactivatedID *uuid.UUID
}

func (m *mockAPIKeyRepo) GetByAccount(accountID uuid.UUID) ([]models.APIKey, error) {
	if m.getByAccErr != nil {
		return nil, m.getByAccErr
	}
	return m.keys, nil
}

func (m *mockAPIKeyRepo) Deactivate(id uuid.UUID) error {
	if m.deactivateErr != nil {
		return m.deactivateErr
	}
	m.deactivatedID = &id
	return nil
}

func (m *mockAPIKeyRepo) Delete(id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

// --- Test helpers ---

func apiKeyAdminAuthDTO(accountID uuid.UUID) *dtos.AuthDTO {
	adminRole := models.RoleAdmin
	return &dtos.AuthDTO{
		User: models.User{
			ID:   uuid.New(),
			Role: &adminRole,
		},
		AccountID: accountID,
	}
}

func setupAPIKeyApp(handler *AdminAPIKeyHandler, authDTO *dtos.AuthDTO) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("authDTO", authDTO)
		return c.Next()
	})
	app.Post("/admin/api-keys", handler.CreateAPIKey)
	app.Get("/admin/api-keys", handler.ListAPIKeys)
	app.Delete("/admin/api-keys/:id", handler.DeleteAPIKey)
	return app
}

// --- Route registration tests ---

func TestAdminAPIKeyRoutes_AllRegistered(t *testing.T) {
	app := fiber.New()
	handler := NewAdminAPIKeyHandler(nil, nil)
	handler.RegisterAdminAPIKeyRoutes(app)

	routes := app.GetRoutes()

	expectedRoutes := []struct {
		method string
		path   string
	}{
		{"POST", "/admin/api-keys"},
		{"GET", "/admin/api-keys"},
		{"DELETE", "/admin/api-keys/:id"},
	}

	for _, expected := range expectedRoutes {
		found := false
		for _, route := range routes {
			if route.Method == expected.method && route.Path == expected.path {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected route %s %s to be registered", expected.method, expected.path)
	}
}

func TestAdminAPIKeyRoutes_RequireAdminMiddleware(t *testing.T) {
	app := fiber.New()

	userRole := models.RoleUser
	accountID := uuid.New()
	authDTO := &dtos.AuthDTO{
		User: models.User{
			ID:   uuid.New(),
			Role: &userRole,
		},
		AccountID: accountID,
	}

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("authDTO", authDTO)
		return c.Next()
	})

	handler := NewAdminAPIKeyHandler(&mockAPIKeyService{}, &mockAPIKeyRepo{})
	admin := app.Group("/admin")
	admin.Use(middleware.RequireAdmin())
	admin.Post("/api-keys", handler.CreateAPIKey)

	requestBody := map[string]interface{}{
		"name": "My API Key",
	}
	body, err := json.Marshal(requestBody)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.NoError(t, err)
	assert.Equal(t, "Admin access required", responseBody["error"])
}

// --- CreateAPIKey tests ---

func TestCreateAPIKey_Success(t *testing.T) {
	accountID := uuid.New()
	svc := &mockAPIKeyService{}
	repo := &mockAPIKeyRepo{}

	handler := NewAdminAPIKeyHandler(svc, repo)
	authDTO := apiKeyAdminAuthDTO(accountID)
	app := setupAPIKeyApp(handler, authDTO)

	requestBody := map[string]interface{}{
		"name": "Production Key",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var responseBody CreateAPIKeyResponse
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.NoError(t, err)
	assert.Equal(t, "nori_testapikey123456", responseBody.RawKey)
	assert.Equal(t, "Production Key", responseBody.APIKey.Name)
	assert.Equal(t, accountID, responseBody.APIKey.AccountID)
	assert.True(t, responseBody.APIKey.IsActive)
}

func TestCreateAPIKey_SuccessWithExpiry(t *testing.T) {
	accountID := uuid.New()
	svc := &mockAPIKeyService{}
	repo := &mockAPIKeyRepo{}

	handler := NewAdminAPIKeyHandler(svc, repo)
	authDTO := apiKeyAdminAuthDTO(accountID)
	app := setupAPIKeyApp(handler, authDTO)

	expiry := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	requestBody := map[string]interface{}{
		"name":      "Expiring Key",
		"expiresAt": expiry,
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var responseBody CreateAPIKeyResponse
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.NoError(t, err)
	assert.NotNil(t, responseBody.APIKey.ExpiresAt)
}

func TestCreateAPIKey_MissingName(t *testing.T) {
	handler := NewAdminAPIKeyHandler(&mockAPIKeyService{}, &mockAPIKeyRepo{})
	authDTO := apiKeyAdminAuthDTO(uuid.New())
	app := setupAPIKeyApp(handler, authDTO)

	requestBody := map[string]interface{}{}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "name is required", responseBody["error"])
}

func TestCreateAPIKey_InvalidExpiryFormat(t *testing.T) {
	handler := NewAdminAPIKeyHandler(&mockAPIKeyService{}, &mockAPIKeyRepo{})
	authDTO := apiKeyAdminAuthDTO(uuid.New())
	app := setupAPIKeyApp(handler, authDTO)

	requestBody := map[string]interface{}{
		"name":      "Bad Expiry Key",
		"expiresAt": "not-a-date",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "invalid expiresAt format, must be RFC3339", responseBody["error"])
}

func TestCreateAPIKey_InvalidRequestBody(t *testing.T) {
	handler := NewAdminAPIKeyHandler(&mockAPIKeyService{}, &mockAPIKeyRepo{})
	authDTO := apiKeyAdminAuthDTO(uuid.New())
	app := setupAPIKeyApp(handler, authDTO)

	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "invalid request body", responseBody["error"])
}

func TestCreateAPIKey_ServiceError(t *testing.T) {
	svc := &mockAPIKeyService{
		generateErr: errors.New("key generation failed"),
	}
	handler := NewAdminAPIKeyHandler(svc, &mockAPIKeyRepo{})
	authDTO := apiKeyAdminAuthDTO(uuid.New())
	app := setupAPIKeyApp(handler, authDTO)

	requestBody := map[string]interface{}{
		"name": "Failing Key",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "key generation failed", responseBody["error"])
}

// --- ListAPIKeys tests ---

func TestListAPIKeys_Success(t *testing.T) {
	accountID := uuid.New()
	now := time.Now()
	repo := &mockAPIKeyRepo{
		keys: []models.APIKey{
			{
				ID:          uuid.New(),
				AccountID:   accountID,
				Name:        "Key One",
				IsActive:    true,
				CreatedAt:   now,
				CreatedByID: uuid.New(),
			},
			{
				ID:          uuid.New(),
				AccountID:   accountID,
				Name:        "Key Two",
				IsActive:    false,
				CreatedAt:   now,
				LastUsedAt:  &now,
				CreatedByID: uuid.New(),
			},
		},
	}

	handler := NewAdminAPIKeyHandler(&mockAPIKeyService{}, repo)
	authDTO := apiKeyAdminAuthDTO(accountID)
	app := setupAPIKeyApp(handler, authDTO)

	req := httptest.NewRequest(http.MethodGet, "/admin/api-keys", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var responseBody []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.NoError(t, err)
	assert.Len(t, responseBody, 2)
	assert.Equal(t, "Key One", responseBody[0]["name"])
	assert.Equal(t, "Key Two", responseBody[1]["name"])
}

func TestListAPIKeys_EmptyList(t *testing.T) {
	repo := &mockAPIKeyRepo{
		keys: []models.APIKey{},
	}

	handler := NewAdminAPIKeyHandler(&mockAPIKeyService{}, repo)
	authDTO := apiKeyAdminAuthDTO(uuid.New())
	app := setupAPIKeyApp(handler, authDTO)

	req := httptest.NewRequest(http.MethodGet, "/admin/api-keys", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var responseBody []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.NoError(t, err)
	assert.Empty(t, responseBody)
}

func TestListAPIKeys_DoesNotExposeKeyHash(t *testing.T) {
	accountID := uuid.New()
	repo := &mockAPIKeyRepo{
		keys: []models.APIKey{
			{
				ID:          uuid.New(),
				AccountID:   accountID,
				Name:        "Secret Key",
				KeyHash:     "super_secret_hash_value",
				IsActive:    true,
				CreatedAt:   time.Now(),
				CreatedByID: uuid.New(),
			},
		},
	}

	handler := NewAdminAPIKeyHandler(&mockAPIKeyService{}, repo)
	authDTO := apiKeyAdminAuthDTO(accountID)
	app := setupAPIKeyApp(handler, authDTO)

	req := httptest.NewRequest(http.MethodGet, "/admin/api-keys", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var responseBody []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.NoError(t, err)
	assert.Len(t, responseBody, 1)

	// KeyHash has json:"-" tag, so it should never appear in the response
	_, hasKeyHash := responseBody[0]["keyHash"]
	assert.False(t, hasKeyHash, "keyHash should not be exposed in API response")
	_, hasKey := responseBody[0]["key_hash"]
	assert.False(t, hasKey, "key_hash should not be exposed in API response")
}

func TestListAPIKeys_RepoError(t *testing.T) {
	repo := &mockAPIKeyRepo{
		getByAccErr: errors.New("database connection lost"),
	}

	handler := NewAdminAPIKeyHandler(&mockAPIKeyService{}, repo)
	authDTO := apiKeyAdminAuthDTO(uuid.New())
	app := setupAPIKeyApp(handler, authDTO)

	req := httptest.NewRequest(http.MethodGet, "/admin/api-keys", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "database connection lost", responseBody["error"])
}

// --- DeleteAPIKey tests ---

func TestDeleteAPIKey_Success(t *testing.T) {
	keyID := uuid.New()
	repo := &mockAPIKeyRepo{}

	handler := NewAdminAPIKeyHandler(&mockAPIKeyService{}, repo)
	authDTO := apiKeyAdminAuthDTO(uuid.New())
	app := setupAPIKeyApp(handler, authDTO)

	req := httptest.NewRequest(http.MethodDelete, "/admin/api-keys/"+keyID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify the correct key ID was deactivated
	assert.NotNil(t, repo.deactivatedID)
	assert.Equal(t, keyID, *repo.deactivatedID)
}

func TestDeleteAPIKey_InvalidKeyID(t *testing.T) {
	handler := NewAdminAPIKeyHandler(&mockAPIKeyService{}, &mockAPIKeyRepo{})
	authDTO := apiKeyAdminAuthDTO(uuid.New())
	app := setupAPIKeyApp(handler, authDTO)

	req := httptest.NewRequest(http.MethodDelete, "/admin/api-keys/invalid-uuid", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "invalid API key ID", responseBody["error"])
}

func TestDeleteAPIKey_DeactivateError(t *testing.T) {
	repo := &mockAPIKeyRepo{
		deactivateErr: errors.New("key not found"),
	}

	handler := NewAdminAPIKeyHandler(&mockAPIKeyService{}, repo)
	authDTO := apiKeyAdminAuthDTO(uuid.New())
	app := setupAPIKeyApp(handler, authDTO)

	keyID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/admin/api-keys/"+keyID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "key not found", responseBody["error"])
}
