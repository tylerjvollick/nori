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
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/services"
)

// --- Mock product service ---
//
// The service itself is tested against real PostgreSQL in
// services/product.service_test.go; handler tests verify HTTP behavior
// (routing, status codes, error mapping, serialization).

type mockProductService struct {
	products map[uuid.UUID]*models.Product
	variants map[uuid.UUID]*models.ProductVariant

	createErr error
	listErr   error
	updateErr error
	deleteErr error

	variantCreateErr error
	variantUpdateErr error
	variantDeleteErr error

	createdProduct *models.Product
	createdVariant *models.ProductVariant
	deletedProduct *uuid.UUID
	deletedVariant *uuid.UUID
}

func newMockProductService() *mockProductService {
	return &mockProductService{
		products: make(map[uuid.UUID]*models.Product),
		variants: make(map[uuid.UUID]*models.ProductVariant),
	}
}

func (m *mockProductService) Create(spaceID uuid.UUID, in services.CreateProductInput) (*models.Product, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if in.Name == "" {
		return nil, services.ErrProductNameRequired
	}
	p := &models.Product{
		ID:          uuid.New(),
		SpaceID:     spaceID,
		Name:        in.Name,
		Description: in.Description,
		IsActive:    true,
		Variants:    []models.ProductVariant{},
	}
	m.products[p.ID] = p
	m.createdProduct = p
	return p, nil
}

func (m *mockProductService) List(spaceID uuid.UUID) ([]models.Product, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var out []models.Product
	for _, p := range m.products {
		if p.SpaceID == spaceID {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (m *mockProductService) GetByID(spaceID, id uuid.UUID) (*models.Product, error) {
	p, ok := m.products[id]
	if !ok || p.SpaceID != spaceID {
		return nil, services.ErrProductNotFound
	}
	return p, nil
}

func (m *mockProductService) Update(spaceID, id uuid.UUID, in services.UpdateProductInput) (*models.Product, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	p, err := m.GetByID(spaceID, id)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		p.Name = *in.Name
	}
	if in.Description != nil {
		p.Description = in.Description
	}
	if in.IsActive != nil {
		p.IsActive = *in.IsActive
	}
	return p, nil
}

func (m *mockProductService) Delete(spaceID, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, err := m.GetByID(spaceID, id); err != nil {
		return err
	}
	m.deletedProduct = &id
	return nil
}

func (m *mockProductService) CreateVariant(spaceID, productID uuid.UUID, in services.CreateVariantInput) (*models.ProductVariant, error) {
	if m.variantCreateErr != nil {
		return nil, m.variantCreateErr
	}
	if _, err := m.GetByID(spaceID, productID); err != nil {
		return nil, err
	}
	if in.Name == "" {
		return nil, services.ErrProductNameRequired
	}
	v := &models.ProductVariant{
		ID:              uuid.New(),
		ProductID:       productID,
		Name:            in.Name,
		RecipeID:        in.RecipeID,
		RecipeVariables: in.RecipeVariables,
		Price:           in.Price,
		IsActive:        true,
	}
	m.variants[v.ID] = v
	m.createdVariant = v
	return v, nil
}

func (m *mockProductService) UpdateVariant(spaceID, productID, variantID uuid.UUID, in services.UpdateVariantInput) (*models.ProductVariant, error) {
	if m.variantUpdateErr != nil {
		return nil, m.variantUpdateErr
	}
	if _, err := m.GetByID(spaceID, productID); err != nil {
		return nil, err
	}
	v, ok := m.variants[variantID]
	if !ok || v.ProductID != productID {
		return nil, services.ErrProductVariantNotFound
	}
	if in.Name != nil {
		v.Name = *in.Name
	}
	if in.RecipeID != nil {
		v.RecipeID = in.RecipeID
	}
	if in.RecipeVariables != nil {
		v.RecipeVariables = in.RecipeVariables
	}
	if in.Price != nil {
		v.Price = in.Price
	}
	if in.IsActive != nil {
		v.IsActive = *in.IsActive
	}
	return v, nil
}

func (m *mockProductService) DeleteVariant(spaceID, productID, variantID uuid.UUID) error {
	if m.variantDeleteErr != nil {
		return m.variantDeleteErr
	}
	if _, err := m.GetByID(spaceID, productID); err != nil {
		return err
	}
	v, ok := m.variants[variantID]
	if !ok || v.ProductID != productID {
		return services.ErrProductVariantNotFound
	}
	delete(m.variants, variantID)
	m.deletedVariant = &variantID
	return nil
}

// --- Test helpers ---

// setupProductApp builds a Fiber app that simulates auth + RequireSpace
// middleware for product handler tests.
func setupProductApp(handler *ProductHandler, authDTO *dtos.AuthDTO, spaceID uuid.UUID) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("authDTO", authDTO)
		c.Locals("spaceID", spaceID)
		return c.Next()
	})
	spaceGroup := app.Group("/api/v1/spaces/:spaceId")
	handler.RegisterProductRoutes(spaceGroup)
	return app
}

// productURL builds a path under the space-scoped route prefix.
func productURL(spaceID uuid.UUID, rest string) string {
	return "/api/v1/spaces/" + spaceID.String() + "/products" + rest
}

func jsonRequest(method, url string, body interface{}) *http.Request {
	var req *http.Request
	if body != nil {
		raw, _ := json.Marshal(body)
		req = httptest.NewRequest(method, url, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	return req
}

// seedMockProduct inserts a product directly into the mock.
func seedMockProduct(m *mockProductService, spaceID uuid.UUID) *models.Product {
	p := &models.Product{
		ID:       uuid.New(),
		SpaceID:  spaceID,
		Name:     "Dining Table",
		IsActive: true,
	}
	m.products[p.ID] = p
	return p
}

// seedMockVariant inserts a variant directly into the mock.
func seedMockVariant(m *mockProductService, productID uuid.UUID) *models.ProductVariant {
	price := decimal.NewFromInt(4200)
	v := &models.ProductVariant{
		ID:        uuid.New(),
		ProductID: productID,
		Name:      "Walnut / Spray PU",
		Price:     &price,
		IsActive:  true,
	}
	m.variants[v.ID] = v
	return v
}

// --- Product CRUD tests ---

func TestCreateProduct_Success(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)

	req := jsonRequest(http.MethodPost, productURL(spaceID, ""), map[string]interface{}{
		"name":        "Dining Table",
		"description": "Solid hardwood",
	})
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result dtos.ProductResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Dining Table", result.Name)
	require.NotNil(t, result.Description)
	assert.Equal(t, "Solid hardwood", *result.Description)
	assert.True(t, result.IsActive)

	require.NotNil(t, svc.createdProduct)
	assert.Equal(t, spaceID, svc.createdProduct.SpaceID)
}

func TestCreateProduct_EmptyName(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)

	req := jsonRequest(http.MethodPost, productURL(spaceID, ""), map[string]interface{}{"name": ""})
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateProduct_InvalidBody(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)

	req := httptest.NewRequest(http.MethodPost, productURL(spaceID, ""), bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateProduct_ServiceError(t *testing.T) {
	svc := newMockProductService()
	svc.createErr = errors.New("db error")
	spaceID := uuid.New()
	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)

	req := jsonRequest(http.MethodPost, productURL(spaceID, ""), map[string]interface{}{"name": "Table"})
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestListProducts_Success(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	seedMockProduct(svc, spaceID)
	seedMockProduct(svc, uuid.New()) // other space — excluded

	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)
	resp, err := app.Test(jsonRequest(http.MethodGet, productURL(spaceID, ""), nil), -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []dtos.ProductResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.Len(t, result, 1)
	assert.Equal(t, "Dining Table", result[0].Name)
}

func TestGetProduct_IncludesVariants(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	product := seedMockProduct(svc, spaceID)
	variant := seedMockVariant(svc, product.ID)
	product.Variants = []models.ProductVariant{*variant}

	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)
	resp, err := app.Test(jsonRequest(http.MethodGet, productURL(spaceID, "/"+product.ID.String()), nil), -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result dtos.ProductResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, product.ID.String(), result.ID)
	require.Len(t, result.Variants, 1)
	assert.Equal(t, "Walnut / Spray PU", result.Variants[0].Name)
	require.NotNil(t, result.Variants[0].Price)
	assert.True(t, result.Variants[0].Price.Equal(decimal.NewFromInt(4200)))
}

func TestGetProduct_NotFound(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)

	resp, err := app.Test(jsonRequest(http.MethodGet, productURL(spaceID, "/"+uuid.New().String()), nil), -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestGetProduct_CrossSpaceReturns404(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	otherProduct := seedMockProduct(svc, uuid.New()) // lives in another space

	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)
	resp, err := app.Test(jsonRequest(http.MethodGet, productURL(spaceID, "/"+otherProduct.ID.String()), nil), -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestGetProduct_InvalidID(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)

	resp, err := app.Test(jsonRequest(http.MethodGet, productURL(spaceID, "/not-a-uuid"), nil), -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateProduct_Success(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	product := seedMockProduct(svc, spaceID)

	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)
	req := jsonRequest(http.MethodPut, productURL(spaceID, "/"+product.ID.String()), map[string]interface{}{
		"name":     "Dining Table v2",
		"isActive": false,
	})
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result dtos.ProductResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Dining Table v2", result.Name)
	assert.False(t, result.IsActive)
}

func TestUpdateProduct_NotFound(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)

	req := jsonRequest(http.MethodPut, productURL(spaceID, "/"+uuid.New().String()), map[string]interface{}{"name": "X"})
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDeleteProduct_Success(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	product := seedMockProduct(svc, spaceID)

	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)
	resp, err := app.Test(jsonRequest(http.MethodDelete, productURL(spaceID, "/"+product.ID.String()), nil), -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	require.NotNil(t, svc.deletedProduct)
	assert.Equal(t, product.ID, *svc.deletedProduct)
}

func TestDeleteProduct_NotFound(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)

	resp, err := app.Test(jsonRequest(http.MethodDelete, productURL(spaceID, "/"+uuid.New().String()), nil), -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// --- Variant endpoint tests ---

func TestCreateVariant_Success(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	product := seedMockProduct(svc, spaceID)
	recipeID := uuid.New()

	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)
	req := jsonRequest(http.MethodPost, productURL(spaceID, "/"+product.ID.String()+"/variants"), map[string]interface{}{
		"name":     "Walnut / Spray PU",
		"recipeId": recipeID.String(),
		"price":    "4200",
		"recipeVariables": map[string]interface{}{
			"wood_species": "Walnut",
		},
	})
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result dtos.ProductVariantResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Walnut / Spray PU", result.Name)
	require.NotNil(t, result.RecipeID)
	assert.Equal(t, recipeID.String(), *result.RecipeID)
	require.NotNil(t, result.Price)
	assert.True(t, result.Price.Equal(decimal.NewFromInt(4200)))
	assert.Equal(t, "Walnut", result.RecipeVariables["wood_species"])

	require.NotNil(t, svc.createdVariant)
	assert.Equal(t, product.ID, svc.createdVariant.ProductID)
}

func TestCreateVariant_EmptyName(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	product := seedMockProduct(svc, spaceID)

	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)
	req := jsonRequest(http.MethodPost, productURL(spaceID, "/"+product.ID.String()+"/variants"), map[string]interface{}{"name": ""})
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateVariant_ProductNotFound(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)

	req := jsonRequest(http.MethodPost, productURL(spaceID, "/"+uuid.New().String()+"/variants"), map[string]interface{}{"name": "Walnut"})
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestCreateVariant_RecipeNotInSpace(t *testing.T) {
	svc := newMockProductService()
	svc.variantCreateErr = services.ErrVariantRecipeNotFound
	spaceID := uuid.New()
	product := seedMockProduct(svc, spaceID)

	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)
	req := jsonRequest(http.MethodPost, productURL(spaceID, "/"+product.ID.String()+"/variants"), map[string]interface{}{
		"name":     "Walnut",
		"recipeId": uuid.New().String(),
	})
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "recipe not found in this space", result["error"])
}

func TestUpdateVariant_Success(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	product := seedMockProduct(svc, spaceID)
	variant := seedMockVariant(svc, product.ID)

	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)
	req := jsonRequest(http.MethodPut, productURL(spaceID, "/"+product.ID.String()+"/variants/"+variant.ID.String()), map[string]interface{}{
		"price": "4500",
	})
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result dtos.ProductVariantResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Walnut / Spray PU", result.Name) // unchanged
	require.NotNil(t, result.Price)
	assert.True(t, result.Price.Equal(decimal.NewFromInt(4500)))
}

func TestUpdateVariant_NotFound(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	product := seedMockProduct(svc, spaceID)

	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)
	req := jsonRequest(http.MethodPut, productURL(spaceID, "/"+product.ID.String()+"/variants/"+uuid.New().String()), map[string]interface{}{
		"price": "4500",
	})
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestUpdateVariant_InvalidVariantID(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	product := seedMockProduct(svc, spaceID)

	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)
	req := jsonRequest(http.MethodPut, productURL(spaceID, "/"+product.ID.String()+"/variants/not-a-uuid"), map[string]interface{}{
		"price": "4500",
	})
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDeleteVariant_Success(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	product := seedMockProduct(svc, spaceID)
	variant := seedMockVariant(svc, product.ID)

	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)
	resp, err := app.Test(jsonRequest(http.MethodDelete, productURL(spaceID, "/"+product.ID.String()+"/variants/"+variant.ID.String()), nil), -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	require.NotNil(t, svc.deletedVariant)
	assert.Equal(t, variant.ID, *svc.deletedVariant)
}

func TestDeleteVariant_NotFound(t *testing.T) {
	svc := newMockProductService()
	spaceID := uuid.New()
	product := seedMockProduct(svc, spaceID)

	app := setupProductApp(NewProductHandler(svc), adminAuthDTOWithSpace(uuid.New(), spaceID), spaceID)
	resp, err := app.Test(jsonRequest(http.MethodDelete, productURL(spaceID, "/"+product.ID.String()+"/variants/"+uuid.New().String()), nil), -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
