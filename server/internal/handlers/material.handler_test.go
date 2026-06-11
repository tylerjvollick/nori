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

// --- Mock material service ---
//
// The service is tested against real PostgreSQL in
// services/material.service_test.go; handler tests verify HTTP behavior
// (routing, status codes, error mapping, serialization).

type mockMaterialService struct {
	materials map[uuid.UUID]*models.Material

	createErr error
	listErr   error

	deletedID *uuid.UUID
}

func newMockMaterialService() *mockMaterialService {
	return &mockMaterialService{materials: make(map[uuid.UUID]*models.Material)}
}

func (m *mockMaterialService) Create(spaceID uuid.UUID, req *dtos.CreateMaterialRequest) (*models.Material, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if req.Name == "" {
		return nil, &services.MaterialValidationError{Message: "name is required"}
	}
	if req.Unit == "" {
		return nil, &services.MaterialValidationError{Message: "unit is required"}
	}
	mat := &models.Material{
		ID:       uuid.New(),
		SpaceID:  spaceID,
		Name:     req.Name,
		Category: models.MaterialCategoryOther,
		Unit:     req.Unit,
		Supplier: req.Supplier,
		SKU:      req.SKU,
		UnitCost: req.UnitCost,
		IsActive: true,
	}
	m.materials[mat.ID] = mat
	return mat, nil
}

func (m *mockMaterialService) List(spaceID uuid.UUID, search string) ([]models.Material, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var out []models.Material
	for _, mat := range m.materials {
		if mat.SpaceID == spaceID {
			out = append(out, *mat)
		}
	}
	return out, nil
}

func (m *mockMaterialService) GetByID(spaceID, id uuid.UUID) (*models.Material, error) {
	mat, ok := m.materials[id]
	if !ok || mat.SpaceID != spaceID {
		return nil, services.ErrMaterialNotFound
	}
	return mat, nil
}

func (m *mockMaterialService) Update(spaceID, id uuid.UUID, req *dtos.UpdateMaterialRequest) (*models.Material, error) {
	mat, err := m.GetByID(spaceID, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		mat.Name = *req.Name
	}
	if req.UnitCost != nil {
		if req.UnitCost.IsNegative() {
			return nil, &services.MaterialValidationError{Message: "unitCost must be >= 0"}
		}
		mat.UnitCost = req.UnitCost
	}
	return mat, nil
}

func (m *mockMaterialService) Delete(spaceID, id uuid.UUID) error {
	mat, err := m.GetByID(spaceID, id)
	if err != nil {
		return err
	}
	m.deletedID = &mat.ID
	delete(m.materials, id)
	return nil
}

func setupMaterialApp(handler *MaterialHandler, authDTO *dtos.AuthDTO, spaceID uuid.UUID) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("authDTO", authDTO)
		c.Locals("spaceID", spaceID)
		return c.Next()
	})
	spaceGroup := app.Group("/api/v1/spaces/:spaceId")
	handler.RegisterMaterialRoutes(spaceGroup)
	return app
}

func materialURL(spaceID uuid.UUID, rest string) string {
	return "/api/v1/spaces/" + spaceID.String() + "/materials" + rest
}

func newMaterialTestApp(t *testing.T) (*fiber.App, *mockMaterialService, uuid.UUID) {
	t.Helper()
	spaceID := uuid.New()
	svc := newMockMaterialService()
	app := setupMaterialApp(NewMaterialHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)
	return app, svc, spaceID
}

// --- Create ---

func TestCreateMaterial_Returns201(t *testing.T) {
	app, _, spaceID := newMaterialTestApp(t)

	cost := decimal.RequireFromString("14.50")
	req := jsonRequest(http.MethodPost, materialURL(spaceID, ""), dtos.CreateMaterialRequest{
		Name:     "8/4 Walnut",
		Unit:     "board_feet",
		UnitCost: &cost,
	})
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var body dtos.MaterialResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "8/4 Walnut", body.Name)
	assert.Equal(t, spaceID, body.SpaceID)
	require.NotNil(t, body.UnitCost)
	assert.True(t, body.UnitCost.Equal(cost))
}

func TestCreateMaterial_ValidationErrorReturns400(t *testing.T) {
	app, _, spaceID := newMaterialTestApp(t)

	req := jsonRequest(http.MethodPost, materialURL(spaceID, ""), dtos.CreateMaterialRequest{Unit: "each"})
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateMaterial_InvalidBodyReturns400(t *testing.T) {
	app, _, spaceID := newMaterialTestApp(t)

	req := jsonRequest(http.MethodPost, materialURL(spaceID, ""), nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- List ---

func TestListMaterials_ReturnsSpaceMaterials(t *testing.T) {
	app, svc, spaceID := newMaterialTestApp(t)

	_, err := svc.Create(spaceID, &dtos.CreateMaterialRequest{Name: "Walnut", Unit: "bf"})
	require.NoError(t, err)
	_, err = svc.Create(uuid.New(), &dtos.CreateMaterialRequest{Name: "Other", Unit: "bf"})
	require.NoError(t, err)

	resp, err := app.Test(jsonRequest(http.MethodGet, materialURL(spaceID, ""), nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body []dtos.MaterialResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Len(t, body, 1)
	assert.Equal(t, "Walnut", body[0].Name)
}

func TestListMaterials_EmptyReturnsEmptyArray(t *testing.T) {
	app, _, spaceID := newMaterialTestApp(t)

	resp, err := app.Test(jsonRequest(http.MethodGet, materialURL(spaceID, ""), nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body []dtos.MaterialResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Len(t, body, 0)
}

// --- Get ---

func TestGetMaterial_Success(t *testing.T) {
	app, svc, spaceID := newMaterialTestApp(t)

	mat, err := svc.Create(spaceID, &dtos.CreateMaterialRequest{Name: "Walnut", Unit: "bf"})
	require.NoError(t, err)

	resp, err := app.Test(jsonRequest(http.MethodGet, materialURL(spaceID, "/"+mat.ID.String()), nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body dtos.MaterialResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, mat.ID, body.ID)
}

func TestGetMaterial_NotFoundReturns404(t *testing.T) {
	app, _, spaceID := newMaterialTestApp(t)

	resp, err := app.Test(jsonRequest(http.MethodGet, materialURL(spaceID, "/"+uuid.NewString()), nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestGetMaterial_InvalidIDReturns400(t *testing.T) {
	app, _, spaceID := newMaterialTestApp(t)

	resp, err := app.Test(jsonRequest(http.MethodGet, materialURL(spaceID, "/not-a-uuid"), nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- Update ---

func TestUpdateMaterial_Success(t *testing.T) {
	app, svc, spaceID := newMaterialTestApp(t)

	mat, err := svc.Create(spaceID, &dtos.CreateMaterialRequest{Name: "Walnut", Unit: "bf"})
	require.NoError(t, err)

	cost := decimal.RequireFromString("15.00")
	req := jsonRequest(http.MethodPut, materialURL(spaceID, "/"+mat.ID.String()), dtos.UpdateMaterialRequest{UnitCost: &cost})
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body dtos.MaterialResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.NotNil(t, body.UnitCost)
	assert.True(t, body.UnitCost.Equal(cost))
}

func TestUpdateMaterial_NotFoundReturns404(t *testing.T) {
	app, _, spaceID := newMaterialTestApp(t)

	name := "New Name"
	req := jsonRequest(http.MethodPut, materialURL(spaceID, "/"+uuid.NewString()), dtos.UpdateMaterialRequest{Name: &name})
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestUpdateMaterial_NegativeCostReturns400(t *testing.T) {
	app, svc, spaceID := newMaterialTestApp(t)

	mat, err := svc.Create(spaceID, &dtos.CreateMaterialRequest{Name: "Walnut", Unit: "bf"})
	require.NoError(t, err)

	cost := decimal.RequireFromString("-1")
	req := jsonRequest(http.MethodPut, materialURL(spaceID, "/"+mat.ID.String()), dtos.UpdateMaterialRequest{UnitCost: &cost})
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- Delete ---

func TestDeleteMaterial_Returns204(t *testing.T) {
	app, svc, spaceID := newMaterialTestApp(t)

	mat, err := svc.Create(spaceID, &dtos.CreateMaterialRequest{Name: "Walnut", Unit: "bf"})
	require.NoError(t, err)

	resp, err := app.Test(jsonRequest(http.MethodDelete, materialURL(spaceID, "/"+mat.ID.String()), nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.NotNil(t, svc.deletedID)
	assert.Equal(t, mat.ID, *svc.deletedID)
}

func TestDeleteMaterial_NotFoundReturns404(t *testing.T) {
	app, _, spaceID := newMaterialTestApp(t)

	resp, err := app.Test(jsonRequest(http.MethodDelete, materialURL(spaceID, "/"+uuid.NewString()), nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
