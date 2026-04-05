package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
)

// --- Mock SOP service ---

type mockSOPService struct {
	// templates keyed by ID, each associated with a spaceID
	templates      map[int]*models.SOPTemplate
	templatesByID  map[int]uuid.UUID // templateID -> spaceID
	versions       map[int]*models.SOPVersion
	steps          map[int]*models.SOPStep
	nextTemplateID int
	nextVersionID  int
	nextStepID     int
	createErr      error
	getErr         error
	getAllErr      error
	updateErr      error
	deleteErr      error
	getVersionsErr error
	getVersionErr  error
	createStepErr  error
	updateStepErr  error
	deleteStepErr  error
	reorderStepErr error
}

func newMockSOPService() *mockSOPService {
	return &mockSOPService{
		templates:      make(map[int]*models.SOPTemplate),
		templatesByID:  make(map[int]uuid.UUID),
		versions:       make(map[int]*models.SOPVersion),
		steps:          make(map[int]*models.SOPStep),
		nextTemplateID: 1,
		nextVersionID:  1,
		nextStepID:     1,
	}
}

func (m *mockSOPService) addTemplate(spaceID uuid.UUID, name string) *models.SOPTemplate {
	id := m.nextTemplateID
	m.nextTemplateID++
	t := &models.SOPTemplate{
		ID:        id,
		SpaceID:   &spaceID,
		Name:      name,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.templates[id] = t
	m.templatesByID[id] = spaceID
	return t
}

func (m *mockSOPService) CreateSOP(dto *dtos.CreateSOPDTO, userID uuid.UUID, spaceID uuid.UUID) (*models.SOPTemplate, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	t := m.addTemplate(spaceID, dto.Name)
	t.CreatedBy = userID
	return t, nil
}

func (m *mockSOPService) GetSOP(templateID int, spaceID uuid.UUID) (*models.SOPTemplate, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	t, ok := m.templates[templateID]
	if !ok {
		return nil, errors.New("SOP template not found")
	}
	// Space scoping check
	if ownerSpace, exists := m.templatesByID[templateID]; exists && ownerSpace != spaceID {
		return nil, errors.New("SOP template not found")
	}
	return t, nil
}

func (m *mockSOPService) GetAllSOPs(spaceID uuid.UUID) ([]models.SOPTemplate, error) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	var result []models.SOPTemplate
	for id, t := range m.templates {
		if ownerSpace, exists := m.templatesByID[id]; exists && ownerSpace == spaceID {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *mockSOPService) UpdateSOP(templateID int, dto *dtos.UpdateSOPDTO, userID uuid.UUID, spaceID uuid.UUID) (*models.SOPTemplate, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	t, ok := m.templates[templateID]
	if !ok {
		return nil, errors.New("SOP template not found")
	}
	if ownerSpace, exists := m.templatesByID[templateID]; exists && ownerSpace != spaceID {
		return nil, errors.New("SOP template not found")
	}
	if dto.Name != nil {
		t.Name = *dto.Name
	}
	return t, nil
}

func (m *mockSOPService) DeleteSOP(templateID int, spaceID uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if ownerSpace, exists := m.templatesByID[templateID]; exists && ownerSpace != spaceID {
		return errors.New("SOP template not found")
	}
	delete(m.templates, templateID)
	delete(m.templatesByID, templateID)
	return nil
}

func (m *mockSOPService) GetSOPVersions(templateID int, spaceID uuid.UUID) ([]models.SOPVersion, error) {
	if m.getVersionsErr != nil {
		return nil, m.getVersionsErr
	}
	if ownerSpace, exists := m.templatesByID[templateID]; exists && ownerSpace != spaceID {
		return nil, errors.New("SOP template not found")
	}
	var result []models.SOPVersion
	for _, v := range m.versions {
		if v.SOPTemplateID == templateID {
			result = append(result, *v)
		}
	}
	return result, nil
}

func (m *mockSOPService) GetSOPVersion(versionID int) (*models.SOPVersion, error) {
	if m.getVersionErr != nil {
		return nil, m.getVersionErr
	}
	v, ok := m.versions[versionID]
	if !ok {
		return nil, errors.New("version not found")
	}
	return v, nil
}

func (m *mockSOPService) CreateStepForTemplate(templateID int, dto *dtos.CreateStepDTO, userID uuid.UUID, spaceID uuid.UUID) (*models.SOPStep, error) {
	if m.createStepErr != nil {
		return nil, m.createStepErr
	}
	if ownerSpace, exists := m.templatesByID[templateID]; exists && ownerSpace != spaceID {
		return nil, errors.New("SOP template not found")
	}
	id := m.nextStepID
	m.nextStepID++
	step := &models.SOPStep{
		ID:    id,
		Title: dto.Title,
		Order: "a",
	}
	m.steps[id] = step
	return step, nil
}

func (m *mockSOPService) UpdateStepForTemplate(templateID int, stepID int, dto *dtos.UpdateStepDTO, userID uuid.UUID, spaceID uuid.UUID) (*models.SOPStep, error) {
	if m.updateStepErr != nil {
		return nil, m.updateStepErr
	}
	if ownerSpace, exists := m.templatesByID[templateID]; exists && ownerSpace != spaceID {
		return nil, errors.New("SOP template not found")
	}
	step, ok := m.steps[stepID]
	if !ok {
		return nil, errors.New("step not found")
	}
	if dto.Title != nil {
		step.Title = *dto.Title
	}
	return step, nil
}

func (m *mockSOPService) DeleteStepForTemplate(templateID int, stepID int, userID uuid.UUID, spaceID uuid.UUID) error {
	if m.deleteStepErr != nil {
		return m.deleteStepErr
	}
	if ownerSpace, exists := m.templatesByID[templateID]; exists && ownerSpace != spaceID {
		return errors.New("SOP template not found")
	}
	delete(m.steps, stepID)
	return nil
}

func (m *mockSOPService) ReorderStepForTemplate(templateID int, stepID int, dto *dtos.ReorderStepDTO, userID uuid.UUID, spaceID uuid.UUID) (*models.SOPStep, error) {
	if m.reorderStepErr != nil {
		return nil, m.reorderStepErr
	}
	if ownerSpace, exists := m.templatesByID[templateID]; exists && ownerSpace != spaceID {
		return nil, errors.New("SOP template not found")
	}
	step, ok := m.steps[stepID]
	if !ok {
		return nil, errors.New("step not found")
	}
	return step, nil
}

// --- SOP test helpers ---

func sopAuthDTO(spaceID uuid.UUID) *dtos.AuthDTO {
	userRole := models.RoleUser
	return &dtos.AuthDTO{
		User: models.User{
			ID:   uuid.New(),
			Role: &userRole,
		},
		AccountID:     uuid.New(),
		ActiveSpaceID: &spaceID,
	}
}

func sopAuthDTONoSpace() *dtos.AuthDTO {
	userRole := models.RoleUser
	return &dtos.AuthDTO{
		User: models.User{
			ID:   uuid.New(),
			Role: &userRole,
		},
		AccountID:     uuid.New(),
		ActiveSpaceID: nil,
	}
}

func setupSOPApp(handler *SOPHandler, authDTO *dtos.AuthDTO) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("authDTO", authDTO)
		return c.Next()
	})
	handler.RegisterSOPRoutes(app)
	return app
}

// --- No active space returns 400 ---

func TestCreateSOP_NoActiveSpace_Returns400(t *testing.T) {
	svc := newMockSOPService()
	handler := NewSOPHandler(svc)
	auth := sopAuthDTONoSpace()
	app := setupSOPApp(handler, auth)

	body, _ := json.Marshal(map[string]interface{}{
		"name":  "Test SOP",
		"steps": []map[string]interface{}{{"title": "Step 1"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/sops/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "no active space", responseBody["error"])
}

func TestGetAllSOPs_NoActiveSpace_Returns400(t *testing.T) {
	svc := newMockSOPService()
	handler := NewSOPHandler(svc)
	auth := sopAuthDTONoSpace()
	app := setupSOPApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/sops/", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "no active space", responseBody["error"])
}

func TestGetSOP_NoActiveSpace_Returns400(t *testing.T) {
	svc := newMockSOPService()
	handler := NewSOPHandler(svc)
	auth := sopAuthDTONoSpace()
	app := setupSOPApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/sops/1", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "no active space", responseBody["error"])
}

func TestDeleteSOP_NoActiveSpace_Returns400(t *testing.T) {
	svc := newMockSOPService()
	handler := NewSOPHandler(svc)
	auth := sopAuthDTONoSpace()
	app := setupSOPApp(handler, auth)

	req := httptest.NewRequest(http.MethodDelete, "/sops/1", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- CreateSOP associates with active space ---

func TestCreateSOP_AssociatesWithActiveSpace(t *testing.T) {
	spaceA := uuid.New()
	svc := newMockSOPService()
	handler := NewSOPHandler(svc)
	auth := sopAuthDTO(spaceA)
	app := setupSOPApp(handler, auth)

	body, _ := json.Marshal(map[string]interface{}{
		"name":  "Assembly SOP",
		"steps": []map[string]interface{}{{"title": "Step 1"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/sops/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "Assembly SOP", responseBody["name"])
	assert.Equal(t, spaceA.String(), responseBody["spaceId"])
}

// --- Cross-space data leakage prevention ---

func TestGetAllSOPs_OnlyReturnsSOPsFromActiveSpace(t *testing.T) {
	spaceA := uuid.New()
	spaceB := uuid.New()
	svc := newMockSOPService()

	// Create SOPs in both spaces
	svc.addTemplate(spaceA, "SOP in Space A")
	svc.addTemplate(spaceB, "SOP in Space B")
	svc.addTemplate(spaceA, "Another SOP in Space A")

	handler := NewSOPHandler(svc)
	auth := sopAuthDTO(spaceA)
	app := setupSOPApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/sops/", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var responseBody []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Len(t, responseBody, 2, "User in Space A should only see 2 SOPs from Space A")

	for _, sop := range responseBody {
		assert.Equal(t, spaceA.String(), sop["spaceId"], "All returned SOPs should belong to Space A")
	}
}

func TestGetSOP_CannotAccessSOPFromOtherSpace(t *testing.T) {
	spaceA := uuid.New()
	spaceB := uuid.New()
	svc := newMockSOPService()

	// Create SOP in Space B
	sopInB := svc.addTemplate(spaceB, "SOP in Space B")

	handler := NewSOPHandler(svc)
	// User is in Space A
	auth := sopAuthDTO(spaceA)
	app := setupSOPApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/sops/"+itoa(sopInB.ID), nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "User in Space A should NOT see SOP from Space B")
}

func TestGetSOP_CanAccessSOPInOwnSpace(t *testing.T) {
	spaceA := uuid.New()
	svc := newMockSOPService()

	sopInA := svc.addTemplate(spaceA, "My SOP")

	handler := NewSOPHandler(svc)
	auth := sopAuthDTO(spaceA)
	app := setupSOPApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/sops/"+itoa(sopInA.ID), nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.Equal(t, "My SOP", responseBody["name"])
}

func TestUpdateSOP_CannotUpdateSOPFromOtherSpace(t *testing.T) {
	spaceA := uuid.New()
	spaceB := uuid.New()
	svc := newMockSOPService()

	sopInB := svc.addTemplate(spaceB, "SOP in Space B")

	handler := NewSOPHandler(svc)
	auth := sopAuthDTO(spaceA)
	app := setupSOPApp(handler, auth)

	body, _ := json.Marshal(map[string]interface{}{
		"changeSummary": "Attempted cross-space update",
		"steps":         []map[string]interface{}{{"title": "Step 1"}},
	})
	req := httptest.NewRequest(http.MethodPut, "/sops/"+itoa(sopInB.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, "Should fail when updating SOP from another space")
}

func TestDeleteSOP_CannotDeleteSOPFromOtherSpace(t *testing.T) {
	spaceA := uuid.New()
	spaceB := uuid.New()
	svc := newMockSOPService()

	sopInB := svc.addTemplate(spaceB, "SOP in Space B")

	handler := NewSOPHandler(svc)
	auth := sopAuthDTO(spaceA)
	app := setupSOPApp(handler, auth)

	req := httptest.NewRequest(http.MethodDelete, "/sops/"+itoa(sopInB.ID), nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, "Should fail when deleting SOP from another space")

	// Verify SOP was NOT deleted
	_, ok := svc.templates[sopInB.ID]
	assert.True(t, ok, "SOP in Space B should not be deleted by user in Space A")
}

func TestDeleteSOP_CanDeleteSOPInOwnSpace(t *testing.T) {
	spaceA := uuid.New()
	svc := newMockSOPService()

	sopInA := svc.addTemplate(spaceA, "My SOP")

	handler := NewSOPHandler(svc)
	auth := sopAuthDTO(spaceA)
	app := setupSOPApp(handler, auth)

	req := httptest.NewRequest(http.MethodDelete, "/sops/"+itoa(sopInA.ID), nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify SOP was deleted
	_, ok := svc.templates[sopInA.ID]
	assert.False(t, ok, "SOP should be deleted")
}

func TestGetSOPVersions_CannotAccessVersionsFromOtherSpace(t *testing.T) {
	spaceA := uuid.New()
	spaceB := uuid.New()
	svc := newMockSOPService()

	sopInB := svc.addTemplate(spaceB, "SOP in Space B")

	handler := NewSOPHandler(svc)
	auth := sopAuthDTO(spaceA)
	app := setupSOPApp(handler, auth)

	req := httptest.NewRequest(http.MethodGet, "/sops/"+itoa(sopInB.ID)+"/versions", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, "Should fail when listing versions of SOP from another space")
}

// --- Step operations respect space scoping ---

func TestCreateStep_NoActiveSpace_Returns400(t *testing.T) {
	svc := newMockSOPService()
	handler := NewSOPHandler(svc)
	auth := sopAuthDTONoSpace()
	app := setupSOPApp(handler, auth)

	body, _ := json.Marshal(map[string]interface{}{"title": "New Step"})
	req := httptest.NewRequest(http.MethodPost, "/sops/1/steps", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateStep_CannotAddStepToSOPInOtherSpace(t *testing.T) {
	spaceA := uuid.New()
	spaceB := uuid.New()
	svc := newMockSOPService()

	sopInB := svc.addTemplate(spaceB, "SOP in Space B")

	handler := NewSOPHandler(svc)
	auth := sopAuthDTO(spaceA)
	app := setupSOPApp(handler, auth)

	body, _ := json.Marshal(map[string]interface{}{"title": "New Step"})
	req := httptest.NewRequest(http.MethodPost, "/sops/"+itoa(sopInB.ID)+"/steps", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, "Should fail when adding step to SOP in another space")
}

// --- Route registration test ---

func TestSOPRoutes_AllRegistered(t *testing.T) {
	svc := newMockSOPService()
	handler := NewSOPHandler(svc)
	app := fiber.New()
	handler.RegisterSOPRoutes(app)

	routes := app.GetRoutes()

	expectedRoutes := []struct {
		method string
		path   string
	}{
		{"POST", "/sops/"},
		{"GET", "/sops/"},
		{"GET", "/sops/:id"},
		{"PUT", "/sops/:id"},
		{"DELETE", "/sops/:id"},
		{"GET", "/sops/:id/versions"},
		{"GET", "/sops/versions/:versionId"},
		{"POST", "/sops/:id/steps"},
		{"PUT", "/sops/:id/steps/:stepId"},
		{"DELETE", "/sops/:id/steps/:stepId"},
		{"PATCH", "/sops/:id/steps/:stepId/reorder"},
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

// --- Helper ---

func itoa(i int) string {
	return strconv.Itoa(i)
}
