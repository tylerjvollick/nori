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

// --- Mock repositories ---

type mockSpaceMemberRepo struct {
	members          map[string]*models.SpaceMember // key: "userID:spaceID"
	createErr        error
	deleteErr        error
	getBySpaceErr    error
	membersBySpaceID map[uuid.UUID][]models.SpaceMember
}

func newMockSpaceMemberRepo() *mockSpaceMemberRepo {
	return &mockSpaceMemberRepo{
		members:          make(map[string]*models.SpaceMember),
		membersBySpaceID: make(map[uuid.UUID][]models.SpaceMember),
	}
}

func memberKey(userID, spaceID uuid.UUID) string {
	return userID.String() + ":" + spaceID.String()
}

func (m *mockSpaceMemberRepo) Create(spaceMember *models.SpaceMember) error {
	if m.createErr != nil {
		return m.createErr
	}
	if spaceMember.ID == uuid.Nil {
		spaceMember.ID = uuid.New()
	}
	spaceMember.CreatedAt = time.Now()
	m.members[memberKey(spaceMember.UserID, spaceMember.SpaceID)] = spaceMember
	return nil
}

func (m *mockSpaceMemberRepo) Delete(userID, spaceID uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.members, memberKey(userID, spaceID))
	return nil
}

func (m *mockSpaceMemberRepo) GetByUserAndSpace(userID, spaceID uuid.UUID) (*models.SpaceMember, error) {
	member, ok := m.members[memberKey(userID, spaceID)]
	if !ok {
		return nil, errors.New("record not found")
	}
	return member, nil
}

func (m *mockSpaceMemberRepo) GetBySpace(spaceID uuid.UUID) ([]models.SpaceMember, error) {
	if m.getBySpaceErr != nil {
		return nil, m.getBySpaceErr
	}
	if members, ok := m.membersBySpaceID[spaceID]; ok {
		return members, nil
	}
	// Build from the members map
	var result []models.SpaceMember
	for _, member := range m.members {
		if member.SpaceID == spaceID {
			result = append(result, *member)
		}
	}
	return result, nil
}

func (m *mockSpaceMemberRepo) GetByUser(userID uuid.UUID) ([]models.Space, error) {
	return nil, nil
}

type mockSpaceRepo struct {
	spaces map[uuid.UUID]*models.Space
}

func newMockSpaceRepo() *mockSpaceRepo {
	return &mockSpaceRepo{
		spaces: make(map[uuid.UUID]*models.Space),
	}
}

func (m *mockSpaceRepo) FindByID(id uuid.UUID) (*models.Space, error) {
	space, ok := m.spaces[id]
	if !ok {
		return nil, errors.New("record not found")
	}
	return space, nil
}

func (m *mockSpaceRepo) FindByAccountID(accountID uuid.UUID) ([]models.Space, error) {
	var result []models.Space
	for _, space := range m.spaces {
		if space.AccountID == accountID {
			result = append(result, *space)
		}
	}
	return result, nil
}

// --- Test helpers ---

func adminAuthDTO(accountID uuid.UUID) *dtos.AuthDTO {
	adminRole := models.RoleAdmin
	return &dtos.AuthDTO{
		User: models.User{
			ID:   uuid.New(),
			Role: &adminRole,
		},
		AccountID: accountID,
	}
}

func setupSpaceMemberApp(handler *AdminSpaceMemberHandler, authDTO *dtos.AuthDTO) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("authDTO", authDTO)
		return c.Next()
	})
	app.Post("/admin/spaces/:id/members", handler.AddSpaceMember)
	app.Get("/admin/spaces/:id/members", handler.GetSpaceMembers)
	app.Delete("/admin/spaces/:id/members/:userId", handler.RemoveSpaceMember)
	return app
}

// --- Route registration tests ---

func TestAdminSpaceMemberRoutes_AllRegistered(t *testing.T) {
	app := fiber.New()
	admin := app.Group("/admin")
	handler := NewAdminSpaceMemberHandler(newMockSpaceMemberRepo(), newMockSpaceRepo())
	handler.RegisterAdminSpaceMemberRoutes(admin)

	routes := app.GetRoutes()

	expectedRoutes := []struct {
		method string
		path   string
	}{
		{"POST", "/admin/spaces/:id/members"},
		{"GET", "/admin/spaces/:id/members"},
		{"DELETE", "/admin/spaces/:id/members/:userId"},
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

func TestAdminSpaceMemberRoutes_RequireAdminMiddleware(t *testing.T) {
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

	handler := NewAdminSpaceMemberHandler(newMockSpaceMemberRepo(), newMockSpaceRepo())
	admin := app.Group("/admin")
	admin.Use(middleware.RequireAdmin())
	handler.RegisterAdminSpaceMemberRoutes(admin)

	spaceID := uuid.New()
	requestBody := map[string]interface{}{
		"userId": uuid.New().String(),
	}
	body, err := json.Marshal(requestBody)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/admin/spaces/"+spaceID.String()+"/members", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.NoError(t, err)
	assert.Equal(t, "Admin access required", responseBody["error"])
}

// --- AddSpaceMember tests ---

func TestAddSpaceMember_Success(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()
	userID := uuid.New()

	spaceRepo := newMockSpaceRepo()
	spaceRepo.spaces[spaceID] = &models.Space{
		ID:        spaceID,
		Name:      "Test Space",
		AccountID: accountID,
	}

	memberRepo := newMockSpaceMemberRepo()

	handler := NewAdminSpaceMemberHandler(memberRepo, spaceRepo)
	authDTO := adminAuthDTO(accountID)
	app := setupSpaceMemberApp(handler, authDTO)

	requestBody := map[string]interface{}{
		"userId": userID.String(),
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/admin/spaces/"+spaceID.String()+"/members", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.NoError(t, err)
	assert.Equal(t, userID.String(), responseBody["userId"])
	assert.Equal(t, spaceID.String(), responseBody["spaceId"])

	// Verify member was actually created in the mock
	_, err = memberRepo.GetByUserAndSpace(userID, spaceID)
	assert.NoError(t, err)
}

func TestAddSpaceMember_ValidatesSpaceID(t *testing.T) {
	handler := NewAdminSpaceMemberHandler(newMockSpaceMemberRepo(), newMockSpaceRepo())
	authDTO := adminAuthDTO(uuid.New())
	app := setupSpaceMemberApp(handler, authDTO)

	requestBody := map[string]interface{}{
		"userId": uuid.New().String(),
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/admin/spaces/invalid-uuid/members", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "invalid space ID", responseBody["error"])
}

func TestAddSpaceMember_ValidatesUserID(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()

	spaceRepo := newMockSpaceRepo()
	spaceRepo.spaces[spaceID] = &models.Space{
		ID:        spaceID,
		Name:      "Test Space",
		AccountID: accountID,
	}

	handler := NewAdminSpaceMemberHandler(newMockSpaceMemberRepo(), spaceRepo)
	authDTO := adminAuthDTO(accountID)
	app := setupSpaceMemberApp(handler, authDTO)

	requestBody := map[string]interface{}{
		"userId": "invalid-uuid",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/admin/spaces/"+spaceID.String()+"/members", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "invalid user ID", responseBody["error"])
}

func TestAddSpaceMember_RequiresUserID(t *testing.T) {
	handler := NewAdminSpaceMemberHandler(newMockSpaceMemberRepo(), newMockSpaceRepo())
	authDTO := adminAuthDTO(uuid.New())
	app := setupSpaceMemberApp(handler, authDTO)

	spaceID := uuid.New()
	requestBody := map[string]interface{}{}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/admin/spaces/"+spaceID.String()+"/members", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "userId is required", responseBody["error"])
}

func TestAddSpaceMember_SpaceNotFound(t *testing.T) {
	handler := NewAdminSpaceMemberHandler(newMockSpaceMemberRepo(), newMockSpaceRepo())
	authDTO := adminAuthDTO(uuid.New())
	app := setupSpaceMemberApp(handler, authDTO)

	spaceID := uuid.New()
	requestBody := map[string]interface{}{
		"userId": uuid.New().String(),
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/admin/spaces/"+spaceID.String()+"/members", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "space not found", responseBody["error"])
}

func TestAddSpaceMember_SpaceBelongsToDifferentAccount(t *testing.T) {
	accountID := uuid.New()
	otherAccountID := uuid.New()
	spaceID := uuid.New()

	spaceRepo := newMockSpaceRepo()
	spaceRepo.spaces[spaceID] = &models.Space{
		ID:        spaceID,
		Name:      "Other Account Space",
		AccountID: otherAccountID,
	}

	handler := NewAdminSpaceMemberHandler(newMockSpaceMemberRepo(), spaceRepo)
	authDTO := adminAuthDTO(accountID)
	app := setupSpaceMemberApp(handler, authDTO)

	requestBody := map[string]interface{}{
		"userId": uuid.New().String(),
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/admin/spaces/"+spaceID.String()+"/members", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "space does not belong to your account", responseBody["error"])
}

func TestAddSpaceMember_DuplicateMembership(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()
	userID := uuid.New()

	spaceRepo := newMockSpaceRepo()
	spaceRepo.spaces[spaceID] = &models.Space{
		ID:        spaceID,
		Name:      "Test Space",
		AccountID: accountID,
	}

	memberRepo := newMockSpaceMemberRepo()
	memberRepo.members[memberKey(userID, spaceID)] = &models.SpaceMember{
		ID:      uuid.New(),
		UserID:  userID,
		SpaceID: spaceID,
	}

	handler := NewAdminSpaceMemberHandler(memberRepo, spaceRepo)
	authDTO := adminAuthDTO(accountID)
	app := setupSpaceMemberApp(handler, authDTO)

	requestBody := map[string]interface{}{
		"userId": userID.String(),
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/admin/spaces/"+spaceID.String()+"/members", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "user is already a member of this space", responseBody["error"])
}

func TestAddSpaceMember_CreateError(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()
	userID := uuid.New()

	spaceRepo := newMockSpaceRepo()
	spaceRepo.spaces[spaceID] = &models.Space{
		ID:        spaceID,
		Name:      "Test Space",
		AccountID: accountID,
	}

	memberRepo := newMockSpaceMemberRepo()
	memberRepo.createErr = errors.New("database connection error")

	handler := NewAdminSpaceMemberHandler(memberRepo, spaceRepo)
	authDTO := adminAuthDTO(accountID)
	app := setupSpaceMemberApp(handler, authDTO)

	requestBody := map[string]interface{}{
		"userId": userID.String(),
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/admin/spaces/"+spaceID.String()+"/members", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "database connection error", responseBody["error"])
}

// --- RemoveSpaceMember tests ---

func TestRemoveSpaceMember_Success(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()
	userID := uuid.New()

	spaceRepo := newMockSpaceRepo()
	spaceRepo.spaces[spaceID] = &models.Space{
		ID:        spaceID,
		Name:      "Test Space",
		AccountID: accountID,
	}

	memberRepo := newMockSpaceMemberRepo()
	memberRepo.members[memberKey(userID, spaceID)] = &models.SpaceMember{
		ID:      uuid.New(),
		UserID:  userID,
		SpaceID: spaceID,
	}

	handler := NewAdminSpaceMemberHandler(memberRepo, spaceRepo)
	authDTO := adminAuthDTO(accountID)
	app := setupSpaceMemberApp(handler, authDTO)

	req := httptest.NewRequest(http.MethodDelete, "/admin/spaces/"+spaceID.String()+"/members/"+userID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify member was removed from the mock
	_, err = memberRepo.GetByUserAndSpace(userID, spaceID)
	assert.Error(t, err)
}

func TestRemoveSpaceMember_ValidatesSpaceID(t *testing.T) {
	handler := NewAdminSpaceMemberHandler(newMockSpaceMemberRepo(), newMockSpaceRepo())
	authDTO := adminAuthDTO(uuid.New())
	app := setupSpaceMemberApp(handler, authDTO)

	userID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/admin/spaces/invalid-uuid/members/"+userID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "invalid space ID", responseBody["error"])
}

func TestRemoveSpaceMember_ValidatesUserID(t *testing.T) {
	handler := NewAdminSpaceMemberHandler(newMockSpaceMemberRepo(), newMockSpaceRepo())
	authDTO := adminAuthDTO(uuid.New())
	app := setupSpaceMemberApp(handler, authDTO)

	spaceID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/admin/spaces/"+spaceID.String()+"/members/invalid-uuid", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "invalid user ID", responseBody["error"])
}

func TestRemoveSpaceMember_SpaceNotFound(t *testing.T) {
	handler := NewAdminSpaceMemberHandler(newMockSpaceMemberRepo(), newMockSpaceRepo())
	authDTO := adminAuthDTO(uuid.New())
	app := setupSpaceMemberApp(handler, authDTO)

	spaceID := uuid.New()
	userID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/admin/spaces/"+spaceID.String()+"/members/"+userID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "space not found", responseBody["error"])
}

func TestRemoveSpaceMember_SpaceBelongsToDifferentAccount(t *testing.T) {
	accountID := uuid.New()
	otherAccountID := uuid.New()
	spaceID := uuid.New()

	spaceRepo := newMockSpaceRepo()
	spaceRepo.spaces[spaceID] = &models.Space{
		ID:        spaceID,
		Name:      "Other Account Space",
		AccountID: otherAccountID,
	}

	handler := NewAdminSpaceMemberHandler(newMockSpaceMemberRepo(), spaceRepo)
	authDTO := adminAuthDTO(accountID)
	app := setupSpaceMemberApp(handler, authDTO)

	userID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/admin/spaces/"+spaceID.String()+"/members/"+userID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "space does not belong to your account", responseBody["error"])
}

func TestRemoveSpaceMember_DeleteError(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()

	spaceRepo := newMockSpaceRepo()
	spaceRepo.spaces[spaceID] = &models.Space{
		ID:        spaceID,
		Name:      "Test Space",
		AccountID: accountID,
	}

	memberRepo := newMockSpaceMemberRepo()
	memberRepo.deleteErr = errors.New("database error")

	handler := NewAdminSpaceMemberHandler(memberRepo, spaceRepo)
	authDTO := adminAuthDTO(accountID)
	app := setupSpaceMemberApp(handler, authDTO)

	userID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/admin/spaces/"+spaceID.String()+"/members/"+userID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "database error", responseBody["error"])
}

// --- GetSpaceMembers tests ---

func TestGetSpaceMembers_Success(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()
	userID1 := uuid.New()
	userID2 := uuid.New()

	spaceRepo := newMockSpaceRepo()
	spaceRepo.spaces[spaceID] = &models.Space{
		ID:        spaceID,
		Name:      "Test Space",
		AccountID: accountID,
	}

	memberRepo := newMockSpaceMemberRepo()
	memberRepo.members[memberKey(userID1, spaceID)] = &models.SpaceMember{
		ID:      uuid.New(),
		UserID:  userID1,
		SpaceID: spaceID,
		User: models.User{
			ID: userID1,
		},
	}
	memberRepo.members[memberKey(userID2, spaceID)] = &models.SpaceMember{
		ID:      uuid.New(),
		UserID:  userID2,
		SpaceID: spaceID,
		User: models.User{
			ID: userID2,
		},
	}

	handler := NewAdminSpaceMemberHandler(memberRepo, spaceRepo)
	authDTO := adminAuthDTO(accountID)
	app := setupSpaceMemberApp(handler, authDTO)

	req := httptest.NewRequest(http.MethodGet, "/admin/spaces/"+spaceID.String()+"/members", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var responseBody []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.NoError(t, err)
	assert.Len(t, responseBody, 2)
}

func TestGetSpaceMembers_EmptyList(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()

	spaceRepo := newMockSpaceRepo()
	spaceRepo.spaces[spaceID] = &models.Space{
		ID:        spaceID,
		Name:      "Test Space",
		AccountID: accountID,
	}

	memberRepo := newMockSpaceMemberRepo()

	handler := NewAdminSpaceMemberHandler(memberRepo, spaceRepo)
	authDTO := adminAuthDTO(accountID)
	app := setupSpaceMemberApp(handler, authDTO)

	req := httptest.NewRequest(http.MethodGet, "/admin/spaces/"+spaceID.String()+"/members", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var responseBody []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.NoError(t, err)
	assert.Empty(t, responseBody)
}

func TestGetSpaceMembers_StripsPasswordHashes(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()
	userID := uuid.New()
	passwordHash := "hashed_password_value"

	spaceRepo := newMockSpaceRepo()
	spaceRepo.spaces[spaceID] = &models.Space{
		ID:        spaceID,
		Name:      "Test Space",
		AccountID: accountID,
	}

	memberRepo := newMockSpaceMemberRepo()
	// Use membersBySpaceID to return members with password hashes
	memberRepo.membersBySpaceID[spaceID] = []models.SpaceMember{
		{
			ID:      uuid.New(),
			UserID:  userID,
			SpaceID: spaceID,
			User: models.User{
				ID:       userID,
				Password: &passwordHash,
			},
		},
	}

	handler := NewAdminSpaceMemberHandler(memberRepo, spaceRepo)
	authDTO := adminAuthDTO(accountID)
	app := setupSpaceMemberApp(handler, authDTO)

	req := httptest.NewRequest(http.MethodGet, "/admin/spaces/"+spaceID.String()+"/members", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var responseBody []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.NoError(t, err)
	assert.Len(t, responseBody, 1)

	// Verify password is not in the response
	user, ok := responseBody[0]["user"].(map[string]interface{})
	assert.True(t, ok)
	_, hasPassword := user["password"]
	assert.False(t, hasPassword, "password hash should not be included in the response")
}

func TestGetSpaceMembers_ValidatesSpaceID(t *testing.T) {
	handler := NewAdminSpaceMemberHandler(newMockSpaceMemberRepo(), newMockSpaceRepo())
	authDTO := adminAuthDTO(uuid.New())
	app := setupSpaceMemberApp(handler, authDTO)

	req := httptest.NewRequest(http.MethodGet, "/admin/spaces/invalid-uuid/members", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "invalid space ID", responseBody["error"])
}

func TestGetSpaceMembers_SpaceNotFound(t *testing.T) {
	handler := NewAdminSpaceMemberHandler(newMockSpaceMemberRepo(), newMockSpaceRepo())
	authDTO := adminAuthDTO(uuid.New())
	app := setupSpaceMemberApp(handler, authDTO)

	spaceID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/admin/spaces/"+spaceID.String()+"/members", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "space not found", responseBody["error"])
}

func TestGetSpaceMembers_SpaceBelongsToDifferentAccount(t *testing.T) {
	accountID := uuid.New()
	otherAccountID := uuid.New()
	spaceID := uuid.New()

	spaceRepo := newMockSpaceRepo()
	spaceRepo.spaces[spaceID] = &models.Space{
		ID:        spaceID,
		Name:      "Other Account Space",
		AccountID: otherAccountID,
	}

	handler := NewAdminSpaceMemberHandler(newMockSpaceMemberRepo(), spaceRepo)
	authDTO := adminAuthDTO(accountID)
	app := setupSpaceMemberApp(handler, authDTO)

	req := httptest.NewRequest(http.MethodGet, "/admin/spaces/"+spaceID.String()+"/members", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "space does not belong to your account", responseBody["error"])
}

func TestGetSpaceMembers_RepoError(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()

	spaceRepo := newMockSpaceRepo()
	spaceRepo.spaces[spaceID] = &models.Space{
		ID:        spaceID,
		Name:      "Test Space",
		AccountID: accountID,
	}

	memberRepo := newMockSpaceMemberRepo()
	memberRepo.getBySpaceErr = errors.New("database error")

	handler := NewAdminSpaceMemberHandler(memberRepo, spaceRepo)
	authDTO := adminAuthDTO(accountID)
	app := setupSpaceMemberApp(handler, authDTO)

	req := httptest.NewRequest(http.MethodGet, "/admin/spaces/"+spaceID.String()+"/members", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "database error", responseBody["error"])
}
