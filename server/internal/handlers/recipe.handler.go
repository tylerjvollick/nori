package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
)

// RecipeRepoInterface defines the repository methods needed by RecipeHandler.
type RecipeRepoInterface interface {
	Create(recipe *models.Recipe) error
	GetByID(id uuid.UUID) (*models.Recipe, error)
	List(filter repositories.RecipeFilter) ([]models.Recipe, int64, error)
	Update(recipe *models.Recipe) error
	Delete(id uuid.UUID) error
	CreateVersion(version *models.RecipeVersion) error
	ListVersions(recipeID uuid.UUID) ([]models.RecipeVersion, error)
	GetVersionByID(id int) (*models.RecipeVersion, error)
	PublishVersion(versionID int) error
}

// RecipePourServiceInterface defines the pour method from RecipeService.
type RecipePourServiceInterface interface {
	PourRecipe(recipeID uuid.UUID, spaceID uuid.UUID, createdByID uuid.UUID, vars map[string]string, orderID *uuid.UUID) (*models.Task, error)
}

// RecipeHandler handles HTTP requests for recipes.
type RecipeHandler struct {
	recipeRepo  RecipeRepoInterface
	pourService RecipePourServiceInterface
}

// NewRecipeHandler creates a new RecipeHandler.
func NewRecipeHandler(recipeRepo RecipeRepoInterface, pourService RecipePourServiceInterface) *RecipeHandler {
	return &RecipeHandler{recipeRepo: recipeRepo, pourService: pourService}
}

// RegisterRecipeRoutes registers recipe API routes on the Fiber app.
func (h *RecipeHandler) RegisterRecipeRoutes(app *fiber.App, middlewares ...fiber.Handler) {
	group := app.Group("/api/v1/recipes", middlewares...)

	group.Get("", h.ListRecipes)
	group.Post("", h.CreateRecipe)
	group.Get("/:id", h.GetRecipe)
	group.Put("/:id", h.UpdateRecipe)
	group.Delete("/:id", h.DeleteRecipe)
	group.Get("/:id/versions", h.ListVersions)
	group.Post("/:id/versions", h.CreateVersion)
	group.Post("/:id/pour", h.PourRecipe)
	group.Post("/:id/versions/:vid/publish", h.PublishVersion)
}

// RegisterRecipeVersionRoutes registers the flat recipe-version API routes.
func (h *RecipeHandler) RegisterRecipeVersionRoutes(app *fiber.App, middlewares ...fiber.Handler) {
	group := app.Group("/api/v1/recipe-versions", middlewares...)

	group.Post("/:id/publish", h.PublishVersionFlat)
}

// ListRecipes returns a paginated list of recipes for the active space.
func (h *RecipeHandler) ListRecipes(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	if authDTO.ActiveSpaceID == nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "X-Space-ID header is required",
		})
	}

	filter := repositories.RecipeFilter{
		SpaceID: authDTO.ActiveSpaceID,
	}

	// Parse optional query parameters.
	if slug := c.Query("slug"); slug != "" {
		filter.Slug = slug
	}
	if categoryID := c.Query("categoryId"); categoryID != "" {
		if id, err := uuid.Parse(categoryID); err == nil {
			filter.CategoryID = &id
		}
	}
	if isActive := c.Query("isActive"); isActive != "" {
		b := isActive == "true"
		filter.IsActive = &b
	}
	if offset := c.Query("offset"); offset != "" {
		if v, err := strconv.Atoi(offset); err == nil {
			filter.Offset = v
		}
	}
	if limit := c.Query("limit"); limit != "" {
		if v, err := strconv.Atoi(limit); err == nil {
			filter.Limit = v
		}
	}

	// Default limit.
	if filter.Limit == 0 {
		filter.Limit = 50
	}

	recipes, total, err := h.recipeRepo.List(filter)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	items := make([]dtos.RecipeResponse, len(recipes))
	for i := range recipes {
		items[i] = dtos.RecipeResponseFromModel(&recipes[i])
	}

	return c.Status(http.StatusOK).JSON(dtos.RecipeListResponse{
		Items:  items,
		Total:  total,
		Offset: filter.Offset,
		Limit:  filter.Limit,
	})
}

// CreateRecipe creates a new recipe in the active space.
func (h *RecipeHandler) CreateRecipe(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	if authDTO.ActiveSpaceID == nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "X-Space-ID header is required",
		})
	}

	var dto dtos.CreateRecipeRequest
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if dto.Name == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "name is required",
		})
	}

	now := time.Now()
	recipe := &models.Recipe{
		ID:          uuid.New(),
		SpaceID:     *authDTO.ActiveSpaceID,
		Name:        dto.Name,
		Slug:        slugify(dto.Name),
		Description: dto.Description,
		CategoryID:  dto.CategoryID,
		CreatedByID: authDTO.User.ID,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.recipeRepo.Create(recipe); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(dtos.RecipeResponseFromModel(recipe))
}

// getRecipeInSpace fetches a recipe by ID and verifies it belongs to the
// requester's active space. Returns 404 on mismatch (not 403) to avoid
// leaking existence of recipes in other spaces.
func (h *RecipeHandler) getRecipeInSpace(c *fiber.Ctx, authDTO *dtos.AuthDTO, recipeID uuid.UUID) (*models.Recipe, error) {
	if authDTO.ActiveSpaceID == nil {
		return nil, c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "X-Space-ID header is required",
		})
	}

	recipe, err := h.recipeRepo.GetByID(recipeID)
	if err != nil {
		return nil, c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "recipe not found",
		})
	}

	if recipe.SpaceID != *authDTO.ActiveSpaceID {
		return nil, c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "recipe not found",
		})
	}

	return recipe, nil
}

// GetRecipe returns a single recipe by ID.
func (h *RecipeHandler) GetRecipe(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid recipe ID",
		})
	}

	recipe, err := h.getRecipeInSpace(c, authDTO, id)
	if err != nil {
		return err
	}

	return c.Status(http.StatusOK).JSON(dtos.RecipeResponseFromModel(recipe))
}

// UpdateRecipe updates an existing recipe.
func (h *RecipeHandler) UpdateRecipe(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid recipe ID",
		})
	}

	// Verify recipe belongs to the requester's space before updating.
	recipe, err := h.getRecipeInSpace(c, authDTO, id)
	if err != nil {
		return err
	}

	var dto dtos.UpdateRecipeRequest
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if dto.Name != nil {
		recipe.Name = *dto.Name
		recipe.Slug = slugify(*dto.Name)
	}
	if dto.Description != nil {
		recipe.Description = dto.Description
	}
	if dto.CategoryID != nil {
		recipe.CategoryID = dto.CategoryID
	}
	if dto.IsActive != nil {
		recipe.IsActive = *dto.IsActive
	}
	recipe.UpdatedAt = time.Now()

	if err := h.recipeRepo.Update(recipe); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(dtos.RecipeResponseFromModel(recipe))
}

// DeleteRecipe soft-deletes a recipe by ID.
func (h *RecipeHandler) DeleteRecipe(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid recipe ID",
		})
	}

	// Verify recipe belongs to the requester's space before deleting.
	if _, err := h.getRecipeInSpace(c, authDTO, id); err != nil {
		return err
	}

	if err := h.recipeRepo.Delete(id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusNoContent).Send(nil)
}

// ListVersions returns all versions for a recipe.
func (h *RecipeHandler) ListVersions(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	recipeID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid recipe ID",
		})
	}

	// Verify the recipe exists and belongs to the requester's space.
	if _, err := h.getRecipeInSpace(c, authDTO, recipeID); err != nil {
		return err
	}

	versions, err := h.recipeRepo.ListVersions(recipeID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	items := make([]dtos.RecipeVersionResponse, len(versions))
	for i := range versions {
		items[i] = dtos.RecipeVersionResponseFromModel(&versions[i])
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"items": items,
		"total": len(items),
	})
}

// CreateVersion creates a new version for a recipe.
func (h *RecipeHandler) CreateVersion(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	recipeID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid recipe ID",
		})
	}

	// Verify the recipe exists and belongs to the requester's space.
	if _, err := h.getRecipeInSpace(c, authDTO, recipeID); err != nil {
		return err
	}

	var dto dtos.CreateRecipeVersionRequest
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if dto.Content == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "content is required",
		})
	}

	// Determine the next version number.
	versions, err := h.recipeRepo.ListVersions(recipeID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	nextVersion := 1
	if len(versions) > 0 {
		// Versions are ordered by version_number DESC, so the first is the latest.
		nextVersion = versions[0].VersionNumber + 1
	}

	now := time.Now()
	version := &models.RecipeVersion{
		RecipeID:      recipeID,
		VersionNumber: nextVersion,
		Status:        models.RecipeVersionStatusDraft,
		Content:       dto.Content,
		ChangeSummary: dto.ChangeSummary,
		AuthorID:      authDTO.User.ID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := h.recipeRepo.CreateVersion(version); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(dtos.RecipeVersionResponseFromModel(version))
}

// PourRecipe pours a recipe into a task graph via the formula engine.
func (h *RecipeHandler) PourRecipe(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	recipeID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid recipe ID",
		})
	}

	// Verify the recipe exists and belongs to the requester's space.
	if _, err := h.getRecipeInSpace(c, authDTO, recipeID); err != nil {
		return err
	}

	var dto dtos.PourRecipeRequest
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	rootTask, err := h.pourService.PourRecipe(
		recipeID,
		*authDTO.ActiveSpaceID,
		authDTO.User.ID,
		dto.Vars,
		dto.OrderID,
	)
	if err != nil {
		return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(dtos.TaskResponseFromModel(rootTask))
}

// PublishVersion publishes a specific version of a recipe, archiving any
// previously published version and setting this as the current version.
func (h *RecipeHandler) PublishVersion(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	recipeID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid recipe ID",
		})
	}

	// Verify the recipe exists and belongs to the requester's space.
	if _, err := h.getRecipeInSpace(c, authDTO, recipeID); err != nil {
		return err
	}

	versionID, err := strconv.Atoi(c.Params("vid"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid version ID",
		})
	}

	// Verify the version belongs to this recipe.
	version, err := h.recipeRepo.GetVersionByID(versionID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "version not found",
		})
	}

	if version.RecipeID != recipeID {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "version not found",
		})
	}

	if err := h.recipeRepo.PublishVersion(versionID); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Re-fetch the version to get the updated status and publishedAt.
	updated, err := h.recipeRepo.GetVersionByID(versionID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(dtos.RecipeVersionResponseFromModel(updated))
}

// PublishVersionFlat publishes a recipe version by its ID alone, without
// requiring the recipe ID in the URL path. Used by the flat
// POST /api/v1/recipe-versions/:id/publish route.
func (h *RecipeHandler) PublishVersionFlat(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	versionID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid version ID",
		})
	}

	// Look up the version to find its parent recipe.
	version, err := h.recipeRepo.GetVersionByID(versionID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "version not found",
		})
	}

	// Verify the recipe belongs to the requester's space.
	if _, err := h.getRecipeInSpace(c, authDTO, version.RecipeID); err != nil {
		return err
	}

	if err := h.recipeRepo.PublishVersion(versionID); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Re-fetch the version to get the updated status and publishedAt.
	updated, err := h.recipeRepo.GetVersionByID(versionID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(dtos.RecipeVersionResponseFromModel(updated))
}

// slugify converts a string into a URL-friendly slug.
func slugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash && b.Len() > 0 {
			b.WriteRune('-')
			prevDash = true
		}
	}
	result := b.String()
	return strings.TrimRight(result, "-")
}
