package handlers

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/services"
)

// ProductServiceInterface defines the service methods needed by ProductHandler.
type ProductServiceInterface interface {
	Create(spaceID uuid.UUID, in services.CreateProductInput) (*models.Product, error)
	List(spaceID uuid.UUID) ([]models.Product, error)
	GetByID(spaceID, id uuid.UUID) (*models.Product, error)
	Update(spaceID, id uuid.UUID, in services.UpdateProductInput) (*models.Product, error)
	Delete(spaceID, id uuid.UUID) error
	CreateVariant(spaceID, productID uuid.UUID, in services.CreateVariantInput) (*models.ProductVariant, error)
	UpdateVariant(spaceID, productID, variantID uuid.UUID, in services.UpdateVariantInput) (*models.ProductVariant, error)
	DeleteVariant(spaceID, productID, variantID uuid.UUID) error
}

// ProductHandler handles HTTP requests for products and product variants.
type ProductHandler struct {
	productService ProductServiceInterface
}

// NewProductHandler creates a new ProductHandler.
func NewProductHandler(productService ProductServiceInterface) *ProductHandler {
	return &ProductHandler{productService: productService}
}

// RegisterProductRoutes registers product API routes under a space-scoped
// router. Expects the router to already have :spaceId in its path and the
// RequireSpace middleware applied.
func (h *ProductHandler) RegisterProductRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	group := router.Group("/products", middlewares...)

	group.Get("", h.ListProducts)
	group.Post("", h.CreateProduct)
	group.Get("/:id", h.GetProduct)
	group.Put("/:id", h.UpdateProduct)
	group.Delete("/:id", h.DeleteProduct)
	group.Post("/:id/variants", h.CreateVariant)
	group.Put("/:id/variants/:variantId", h.UpdateVariant)
	group.Delete("/:id/variants/:variantId", h.DeleteVariant)
}

// productErrorResponse maps service errors to HTTP responses. Not-found
// errors (including cross-space access) return 404; recipe validation
// failures return 422; validation errors return 400.
func productErrorResponse(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, services.ErrProductNotFound),
		errors.Is(err, services.ErrProductVariantNotFound):
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, services.ErrVariantRecipeNotFound):
		return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, services.ErrProductNameRequired):
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
}

// CreateProduct creates a new product in the space.
func (h *ProductHandler) CreateProduct(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	var req dtos.CreateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	product, err := h.productService.Create(spaceID, services.CreateProductInput{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		return productErrorResponse(c, err)
	}

	return c.Status(http.StatusCreated).JSON(dtos.ProductResponseFromModel(product))
}

// ListProducts returns all products in the space.
func (h *ProductHandler) ListProducts(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	products, err := h.productService.List(spaceID)
	if err != nil {
		return productErrorResponse(c, err)
	}

	resp := make([]dtos.ProductResponse, len(products))
	for i := range products {
		resp[i] = dtos.ProductResponseFromModel(&products[i])
	}
	return c.JSON(resp)
}

// GetProduct returns a single product by ID, including all its variants.
func (h *ProductHandler) GetProduct(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid product ID",
		})
	}

	product, err := h.productService.GetByID(spaceID, id)
	if err != nil {
		return productErrorResponse(c, err)
	}

	return c.JSON(dtos.ProductResponseFromModel(product))
}

// UpdateProduct updates product fields.
func (h *ProductHandler) UpdateProduct(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid product ID",
		})
	}

	var req dtos.UpdateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	product, err := h.productService.Update(spaceID, id, services.UpdateProductInput{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    req.IsActive,
	})
	if err != nil {
		return productErrorResponse(c, err)
	}

	return c.JSON(dtos.ProductResponseFromModel(product))
}

// DeleteProduct soft-deletes a product and deactivates all its variants.
func (h *ProductHandler) DeleteProduct(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid product ID",
		})
	}

	if err := h.productService.Delete(spaceID, id); err != nil {
		return productErrorResponse(c, err)
	}

	return c.Status(http.StatusNoContent).Send(nil)
}

// CreateVariant creates a new variant under a product.
func (h *ProductHandler) CreateVariant(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	productID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid product ID",
		})
	}

	var req dtos.CreateProductVariantRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	variant, err := h.productService.CreateVariant(spaceID, productID, services.CreateVariantInput{
		Name:            req.Name,
		RecipeID:        req.RecipeID,
		RecipeVariables: models.JSONB(req.RecipeVariables),
		Price:           req.Price,
	})
	if err != nil {
		return productErrorResponse(c, err)
	}

	return c.Status(http.StatusCreated).JSON(dtos.ProductVariantResponseFromModel(variant))
}

// UpdateVariant updates variant fields.
func (h *ProductHandler) UpdateVariant(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	productID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid product ID",
		})
	}

	variantID, err := uuid.Parse(c.Params("variantId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid variant ID",
		})
	}

	var req dtos.UpdateProductVariantRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	variant, err := h.productService.UpdateVariant(spaceID, productID, variantID, services.UpdateVariantInput{
		Name:            req.Name,
		RecipeID:        req.RecipeID,
		RecipeVariables: models.JSONB(req.RecipeVariables),
		Price:           req.Price,
		IsActive:        req.IsActive,
	})
	if err != nil {
		return productErrorResponse(c, err)
	}

	return c.JSON(dtos.ProductVariantResponseFromModel(variant))
}

// DeleteVariant hard-deletes a variant from a product.
func (h *ProductHandler) DeleteVariant(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	productID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid product ID",
		})
	}

	variantID, err := uuid.Parse(c.Params("variantId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid variant ID",
		})
	}

	if err := h.productService.DeleteVariant(spaceID, productID, variantID); err != nil {
		return productErrorResponse(c, err)
	}

	return c.Status(http.StatusNoContent).Send(nil)
}
