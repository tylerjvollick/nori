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
	"github.com/tylerjvollick/nori/internal/models"
)

// --- Mock space service ---

type mockSpaceService struct {
	spaces          map[uuid.UUID]*models.Space
	createErr       error
	updateErr       error
	deleteErr       error
	getByIDErr      error
	listErr         error
	recentErr       error
	visitErr        error
	recentSpaces    []models.Space
	deletedSpaceIDs []uuid.UUID
}

func newMockSpaceService() *mockSpaceService {
	return &mockSpaceService{
		spaces:          make(map[uuid.UUID]*models.Space),
		deletedSpaceIDs: []uuid.UUID{},
	}
}

func (m *mockSpaceService) CreateSpace(accountID uuid.UUID, dto *dtos.CreateSpaceDTO, creatorUserID uuid.UUID) (*models.Space, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	slug := "TST"
	if dto.Slug != nil && *dto.Slug != "" {
		slug = *dto.Slug
	}
	space := &models.Space{
		ID:        uuid.New(),
		Name:      dto.Name,
		Slug:      slug,
		AccountID: accountID,
		IsDefault: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.spaces[space.ID] = space
	return space, nil
}

func (m *mockSpaceService) GetSpaceByID(spaceID uuid.UUID, accountID uuid.UUID) (*models.Space, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	space, ok := m.spaces[spaceID]
	if !ok {
		return nil, errors.New("space not found")
	}
	if space.AccountID != accountID {
		return nil, errors.New("unauthorized access to space")
	}
	return space, nil
}

func (m *mockSpaceService) GetSpaceBySlug(slug string, accountID uuid.UUID) (*models.Space, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	for _, space := range m.spaces {
		if space.Slug == slug && space.AccountID == accountID {
			return space, nil
		}
	}
	return nil, errors.New("space not found")
}

func (m *mockSpaceService) GetSpacesByAccountID(accountID uuid.UUID) ([]models.Space, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []models.Space
	for _, space := range m.spaces {
		if space.AccountID == accountID {
			result = append(result, *space)
		}
	}
	return result, nil
}

func (m *mockSpaceService) GetRecentSpaces(userID uuid.UUID, accountID uuid.UUID) ([]models.Space, error) {
	if m.recentErr != nil {
		return nil, m.recentErr
	}
	if m.recentSpaces != nil {
		return m.recentSpaces, nil
	}
	return []models.Space{}, nil
}

func (m *mockSpaceService) UpdateSpace(spaceID uuid.UUID, accountID uuid.UUID, dto *dtos.UpdateSpaceDTO) (*models.Space, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	space, ok := m.spaces[spaceID]
	if !ok {
		return nil, errors.New("space not found")
	}
	if space.AccountID != accountID {
		return nil, errors.New("unauthorized access to space")
	}
	if dto.Name != nil {
		space.Name = *dto.Name
	}
	space.UpdatedAt = time.Now()
	return space, nil
}

func (m *mockSpaceService) DeleteSpace(spaceID uuid.UUID, accountID uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	space, ok := m.spaces[spaceID]
	if !ok {
		return errors.New("space not found")
	}
	if space.AccountID != accountID {
		return errors.New("unauthorized access to space")
	}
	delete(m.spaces, spaceID)
	m.deletedSpaceIDs = append(m.deletedSpaceIDs, spaceID)
	return nil
}

func (m *mockSpaceService) RecordSpaceVisit(userID uuid.UUID, spaceID uuid.UUID) error {
	if m.visitErr != nil {
		return m.visitErr
	}
	return nil
}

// --- Test helpers ---

func userAuthDTO(accountID uuid.UUID) *dtos.AuthDTO {
	userRole := models.RoleUser
	return &dtos.AuthDTO{
		User: models.User{
			ID:   uuid.New(),
			Role: &userRole,
		},
		AccountID: accountID,
	}
}

func setupSpaceApp(handler *SpaceHandler, authDTO *dtos.AuthDTO) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("authDTO", authDTO)
		return c.Next()
	})
	handler.RegisterSpaceRoutes(app)
	return app
}

// --- CreateSpace tests ---

func TestCreateSpace_AdminSuccess(t *testing.T) {
	accountID := uuid.New()
	svc := newMockSpaceService()
	memberRepo := newMockSpaceMemberRepo()

	handler := NewSpaceHandler(svc, memberRepo)
	auth := adminAuthDTO(accountID)
	app := setupSpaceApp(handler, auth)

	body, _ := json.Marshal(map[string]string{"name": "New Space"})
	req := httptest.NewRequest(http.MethodPost, "/api/spaces", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "New Space", responseBody["name"])
	assert.Equal(t, accountID.String(), responseBody["accountId"])
}

func TestCreateSpace_NonAdminForbidden(t *testing.T) {
	accountID := uuid.New()
	svc := newMockSpaceService()
	memberRepo := newMockSpaceMemberRepo()

	handler := NewSpaceHandler(svc, memberRepo)
	auth := userAuthDTO(accountID)
	app := setupSpaceApp(handler, auth)

	body, _ := json.Marshal(map[string]string{"name": "New Space"})
	req := httptest.NewRequest(http.MethodPost, "/api/spaces", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "admin access required", responseBody["error"])
}

func TestCreateSpace_NilRoleForbidden(t *testing.T) {
	accountID := uuid.New()
	svc := newMockSpaceService()
	memberRepo := newMockSpaceMemberRepo()

	handler := NewSpaceHandler(svc, memberRepo)
	auth := &dtos.AuthDTO{
		User: models.User{
			ID:   uuid.New(),
			Role: nil, // no role set
		},
		AccountID: accountID,
	}
	app := setupSpaceApp(handler, auth)

	body, _ := json.Marshal(map[string]string{"name": "New Space"})
	req := httptest.NewRequest(http.MethodPost, "/api/spaces", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// --- GetSpaces tests ---

func TestGetSpaces_AdminSeesAllAccountSpaces(t *testing.T) {
	accountID := uuid.New()
	svc := newMockSpaceService()
	memberRepo := newMockSpaceMemberRepo()

	space1ID := uuid.New()
	space2ID := uuid.New()
	svc.spaces[space1ID] = &models.Space{
		ID: space1ID, Name: "Space 1", AccountID: accountID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	svc.spaces[space2ID] = &models.Space{
		ID: space2ID, Name: "Space 2", AccountID: accountID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	handler := NewSpaceHandler(svc, memberRepo)
	auth := adminAuthDTO(accountID)
	app := setupSpaceApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/api/spaces", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var responseBody []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Len(t, responseBody, 2)
}

func TestGetSpaces_UserSeesOnlyMemberSpaces(t *testing.T) {
	accountID := uuid.New()
	svc := newMockSpaceService()
	memberRepo := newMockSpaceMemberRepo()

	// Create two spaces: user is member of only one
	space1ID := uuid.New()
	space2ID := uuid.New()
	svc.spaces[space1ID] = &models.Space{
		ID: space1ID, Name: "Member Space", AccountID: accountID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	svc.spaces[space2ID] = &models.Space{
		ID: space2ID, Name: "Other Space", AccountID: accountID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	handler := NewSpaceHandler(svc, memberRepo)
	auth := userAuthDTO(accountID)

	// Add user as member of space1 only
	memberRepo.members[memberKey(auth.User.ID, space1ID)] = &models.SpaceMember{
		ID:      uuid.New(),
		UserID:  auth.User.ID,
		SpaceID: space1ID,
	}

	// Override GetByUser to return the space the user is a member of
	// The mockSpaceMemberRepo.GetByUser returns nil by default, so we need a
	// custom mock that returns the correct spaces
	memberRepoWithSpaces := &mockSpaceMemberRepoWithSpaces{
		mockSpaceMemberRepo: memberRepo,
		userSpaces: map[uuid.UUID][]models.Space{
			auth.User.ID: {*svc.spaces[space1ID]},
		},
	}

	handler = NewSpaceHandler(svc, memberRepoWithSpaces)
	app := setupSpaceApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/api/spaces", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var responseBody []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Len(t, responseBody, 1)
	assert.Equal(t, "Member Space", responseBody[0]["name"])
}

func TestGetSpaces_UserSeesEmptyListWhenNoMemberships(t *testing.T) {
	accountID := uuid.New()
	svc := newMockSpaceService()
	memberRepo := newMockSpaceMemberRepo()

	space1ID := uuid.New()
	svc.spaces[space1ID] = &models.Space{
		ID: space1ID, Name: "Space 1", AccountID: accountID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	memberRepoWithSpaces := &mockSpaceMemberRepoWithSpaces{
		mockSpaceMemberRepo: memberRepo,
		userSpaces:          make(map[uuid.UUID][]models.Space),
	}

	handler := NewSpaceHandler(svc, memberRepoWithSpaces)
	auth := userAuthDTO(accountID)
	app := setupSpaceApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/api/spaces", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var responseBody []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Empty(t, responseBody)
}

// --- GetSpaceByID tests ---

func TestGetSpaceByID_AdminSuccess(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()
	svc := newMockSpaceService()
	memberRepo := newMockSpaceMemberRepo()

	svc.spaces[spaceID] = &models.Space{
		ID: spaceID, Name: "Test Space", AccountID: accountID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	handler := NewSpaceHandler(svc, memberRepo)
	auth := adminAuthDTO(accountID)
	app := setupSpaceApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/api/spaces/"+spaceID.String(), nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "Test Space", responseBody["name"])
}

func TestGetSpaceByID_MemberSuccess(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()
	svc := newMockSpaceService()
	memberRepo := newMockSpaceMemberRepo()

	svc.spaces[spaceID] = &models.Space{
		ID: spaceID, Name: "Test Space", AccountID: accountID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	handler := NewSpaceHandler(svc, memberRepo)
	auth := userAuthDTO(accountID)

	// User is a member of this space
	memberRepo.members[memberKey(auth.User.ID, spaceID)] = &models.SpaceMember{
		ID: uuid.New(), UserID: auth.User.ID, SpaceID: spaceID,
	}

	app := setupSpaceApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/api/spaces/"+spaceID.String(), nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "Test Space", responseBody["name"])
}

func TestGetSpaceByID_NonMemberForbidden(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()
	svc := newMockSpaceService()
	memberRepo := newMockSpaceMemberRepo()

	svc.spaces[spaceID] = &models.Space{
		ID: spaceID, Name: "Test Space", AccountID: accountID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	handler := NewSpaceHandler(svc, memberRepo)
	auth := userAuthDTO(accountID)
	// User is NOT a member
	app := setupSpaceApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/api/spaces/"+spaceID.String(), nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "access to this space is not permitted", responseBody["error"])
}

// --- UpdateSpace tests ---

func TestUpdateSpace_AdminSuccess(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()
	svc := newMockSpaceService()
	memberRepo := newMockSpaceMemberRepo()

	svc.spaces[spaceID] = &models.Space{
		ID: spaceID, Name: "Old Name", AccountID: accountID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	handler := NewSpaceHandler(svc, memberRepo)
	auth := adminAuthDTO(accountID)
	app := setupSpaceApp(handler, auth)

	newName := "New Name"
	body, _ := json.Marshal(map[string]*string{"name": &newName})
	req := httptest.NewRequest(http.MethodPut, "/api/spaces/"+spaceID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "New Name", responseBody["name"])
}

func TestUpdateSpace_MemberSuccess(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()
	svc := newMockSpaceService()
	memberRepo := newMockSpaceMemberRepo()

	svc.spaces[spaceID] = &models.Space{
		ID: spaceID, Name: "Old Name", AccountID: accountID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	handler := NewSpaceHandler(svc, memberRepo)
	auth := userAuthDTO(accountID)

	memberRepo.members[memberKey(auth.User.ID, spaceID)] = &models.SpaceMember{
		ID: uuid.New(), UserID: auth.User.ID, SpaceID: spaceID,
	}

	app := setupSpaceApp(handler, auth)

	newName := "New Name"
	body, _ := json.Marshal(map[string]*string{"name": &newName})
	req := httptest.NewRequest(http.MethodPut, "/api/spaces/"+spaceID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUpdateSpace_NonMemberForbidden(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()
	svc := newMockSpaceService()
	memberRepo := newMockSpaceMemberRepo()

	svc.spaces[spaceID] = &models.Space{
		ID: spaceID, Name: "Old Name", AccountID: accountID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	handler := NewSpaceHandler(svc, memberRepo)
	auth := userAuthDTO(accountID)
	app := setupSpaceApp(handler, auth)

	newName := "New Name"
	body, _ := json.Marshal(map[string]*string{"name": &newName})
	req := httptest.NewRequest(http.MethodPut, "/api/spaces/"+spaceID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// --- DeleteSpace tests ---

func TestDeleteSpace_AdminSuccess(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()
	svc := newMockSpaceService()
	memberRepo := newMockSpaceMemberRepo()

	svc.spaces[spaceID] = &models.Space{
		ID: spaceID, Name: "To Delete", AccountID: accountID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	handler := NewSpaceHandler(svc, memberRepo)
	auth := adminAuthDTO(accountID)
	app := setupSpaceApp(handler, auth)

	req := httptest.NewRequest(http.MethodDelete, "/api/spaces/"+spaceID.String(), nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Contains(t, svc.deletedSpaceIDs, spaceID)
}

func TestDeleteSpace_NonMemberForbidden(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()
	svc := newMockSpaceService()
	memberRepo := newMockSpaceMemberRepo()

	svc.spaces[spaceID] = &models.Space{
		ID: spaceID, Name: "To Delete", AccountID: accountID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	handler := NewSpaceHandler(svc, memberRepo)
	auth := userAuthDTO(accountID)
	app := setupSpaceApp(handler, auth)

	req := httptest.NewRequest(http.MethodDelete, "/api/spaces/"+spaceID.String(), nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// Verify space was NOT deleted
	_, ok := svc.spaces[spaceID]
	assert.True(t, ok, "space should not be deleted")
}

func TestDeleteSpace_MemberSuccess(t *testing.T) {
	accountID := uuid.New()
	spaceID := uuid.New()
	svc := newMockSpaceService()
	memberRepo := newMockSpaceMemberRepo()

	svc.spaces[spaceID] = &models.Space{
		ID: spaceID, Name: "To Delete", AccountID: accountID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	handler := NewSpaceHandler(svc, memberRepo)
	auth := userAuthDTO(accountID)

	memberRepo.members[memberKey(auth.User.ID, spaceID)] = &models.SpaceMember{
		ID: uuid.New(), UserID: auth.User.ID, SpaceID: spaceID,
	}

	app := setupSpaceApp(handler, auth)

	req := httptest.NewRequest(http.MethodDelete, "/api/spaces/"+spaceID.String(), nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

// --- Route registration tests ---

func TestSpaceRoutes_AllRegistered(t *testing.T) {
	svc := newMockSpaceService()
	memberRepo := newMockSpaceMemberRepo()
	handler := NewSpaceHandler(svc, memberRepo)

	app := fiber.New()
	handler.RegisterSpaceRoutes(app)

	routes := app.GetRoutes()

	expectedRoutes := []struct {
		method string
		path   string
	}{
		{"POST", "/api/spaces"},
		{"GET", "/api/spaces"},
		{"GET", "/api/spaces/recent"},
		{"GET", "/api/spaces/by-slug/:slug"},
		{"GET", "/api/spaces/:id"},
		{"PUT", "/api/spaces/:id"},
		{"DELETE", "/api/spaces/:id"},
		{"POST", "/api/spaces/:id/visit"},
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

// --- Helper: mockSpaceMemberRepoWithSpaces ---
// Extends mockSpaceMemberRepo to return spaces for GetByUser

type mockSpaceMemberRepoWithSpaces struct {
	*mockSpaceMemberRepo
	userSpaces map[uuid.UUID][]models.Space
}

func (m *mockSpaceMemberRepoWithSpaces) GetByUser(userID uuid.UUID) ([]models.Space, error) {
	if spaces, ok := m.userSpaces[userID]; ok {
		return spaces, nil
	}
	return []models.Space{}, nil
}
