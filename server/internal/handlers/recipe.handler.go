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
	"github.com/tylerjvollick/nori/internal/services"
)

// RecipeRepoInterface defines the repository methods needed by RecipeHandler.
type RecipeRepoInterface interface {
	GetByID(id uuid.UUID) (*models.Recipe, error)
	List(filter repositories.RecipeFilter) ([]models.Recipe, int64, error)
	Update(recipe *models.Recipe) error
	Delete(id uuid.UUID) error
	ListVersions(recipeID uuid.UUID) ([]models.RecipeVersion, error)
	GetVersionByID(id int) (*models.RecipeVersion, error)
	GetVersionWithTaskTree(id int) (*models.RecipeVersion, []models.Task, error)
}

// RecipePourServiceInterface defines the pour method from RecipeService.
type RecipePourServiceInterface interface {
	PourRecipe(recipeID uuid.UUID, spaceID uuid.UUID, createdByID uuid.UUID, vars map[string]string, orderID *uuid.UUID) (*models.Task, error)
}

// RecipeRollServiceInterface defines the roll method from RecipeService.
type RecipeRollServiceInterface interface {
	RollRecipe(recipeID uuid.UUID, spaceID uuid.UUID, createdByID uuid.UUID, opts services.RollRecipeOptions) (*models.Task, error)
}

// RecipeAuthoringServiceInterface defines task-tree recipe authoring methods.
type RecipeAuthoringServiceInterface interface {
	CreateRecipeWithTaskTree(spaceID uuid.UUID, createdByID uuid.UUID, name string, description *string, categoryID *uuid.UUID) (*models.Recipe, *models.RecipeVersion, error)
	PublishVersion(recipeID uuid.UUID) (*models.RecipeVersion, error)
	CreateDraftFromPublished(recipeID uuid.UUID, authorID uuid.UUID, changeSummary *string) (*models.RecipeVersion, error)
}

// RecipeHandler handles HTTP requests for recipes.
type RecipeHandler struct {
	recipeRepo      RecipeRepoInterface
	pourService     RecipePourServiceInterface
	rollService     RecipeRollServiceInterface
	authoringService RecipeAuthoringServiceInterface
}

// NewRecipeHandler creates a new RecipeHandler.
func NewRecipeHandler(
	recipeRepo RecipeRepoInterface,
	pourService RecipePourServiceInterface,
	rollService RecipeRollServiceInterface,
	authoringService RecipeAuthoringServiceInterface,
) *RecipeHandler {
	return &RecipeHandler{
		recipeRepo:       recipeRepo,
		pourService:      pourService,
		rollService:      rollService,
		authoringService: authoringService,
	}
}

// RegisterRecipeRoutes registers recipe API routes on a router group.
// The router should already be scoped to /api/v1/spaces/:spaceId.
func (h *RecipeHandler) RegisterRecipeRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	group := router.Group("/recipes", middlewares...)

	group.Get("", h.ListRecipes)
	group.Post("", h.CreateRecipe)
	group.Get("/:id", h.GetRecipe)
	group.Put("/:id", h.UpdateRecipe)
	group.Delete("/:id", h.DeleteRecipe)
	group.Get("/:id/tasks", h.GetRecipeTasks)
	group.Get("/:id/versions", h.ListVersions)
	group.Post("/:id/versions", h.CreateVersion)
	group.Post("/:id/pour", h.PourRecipe)
	group.Post("/:id/roll", h.RollRecipe)
	group.Post("/:id/publish", h.PublishVersion)
}

// RegisterRecipeVersionRoutes registers the flat recipe-version API routes.
// The router should already be scoped to /api/v1/spaces/:spaceId.
func (h *RecipeHandler) RegisterRecipeVersionRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	group := router.Group("/recipe-versions", middlewares...)

	group.Post("/:id/publish", h.PublishVersionFlat)
}

// ListRecipes returns a paginated list of recipes for the active space.
func (h *RecipeHandler) ListRecipes(c *fiber.Ctx) error {
	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	filter := repositories.RecipeFilter{
		SpaceID: &spaceID,
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

// CreateRecipe creates a new recipe in the active space. This creates the recipe,
// a root task (Type='recipe'), and an initial draft version with RootTaskID.
func (h *RecipeHandler) CreateRecipe(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
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

	recipe, _, err := h.authoringService.CreateRecipeWithTaskTree(
		spaceID,
		authDTO.User.ID,
		dto.Name,
		dto.Description,
		dto.CategoryID,
	)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Re-fetch to get CurrentVersion preloaded.
	recipe, err = h.recipeRepo.GetByID(recipe.ID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(dtos.RecipeResponseFromModel(recipe))
}

// getRecipeInSpace fetches a recipe by ID and verifies it belongs to the
// space from the URL path. Returns 404 on mismatch (not 403) to avoid
// leaking existence of recipes in other spaces.
func (h *RecipeHandler) getRecipeInSpace(c *fiber.Ctx, recipeID uuid.UUID) (*models.Recipe, error) {
	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return nil, err
	}

	recipe, err := h.recipeRepo.GetByID(recipeID)
	if err != nil {
		return nil, c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "recipe not found",
		})
	}

	if recipe.SpaceID != spaceID {
		return nil, c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "recipe not found",
		})
	}

	return recipe, nil
}

// GetRecipe returns a single recipe by ID, including the current version's task tree.
func (h *RecipeHandler) GetRecipe(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid recipe ID",
		})
	}

	recipe, err := h.getRecipeInSpace(c, id)
	if err != nil {
		return err
	}

	resp := dtos.RecipeResponseFromModel(recipe)

	// If the recipe has a current version with a task tree, load and include it.
	if recipe.CurrentVersion != nil && recipe.CurrentVersion.RootTaskID != nil {
		version, tasks, err := h.recipeRepo.GetVersionWithTaskTree(recipe.CurrentVersion.ID)
		if err == nil && len(tasks) > 0 {
			cv := dtos.RecipeVersionResponseWithTree(version, tasks)
			resp.CurrentVersion = &cv
		}
	}

	return c.Status(http.StatusOK).JSON(resp)
}

// GetRecipeTasks returns the task tree for the recipe's current version.
func (h *RecipeHandler) GetRecipeTasks(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid recipe ID",
		})
	}

	recipe, err := h.getRecipeInSpace(c, id)
	if err != nil {
		return err
	}

	if recipe.CurrentVersion == nil || recipe.CurrentVersion.RootTaskID == nil {
		return c.Status(http.StatusOK).JSON(fiber.Map{
			"items": []any{},
			"total": 0,
		})
	}

	_, tasks, err := h.recipeRepo.GetVersionWithTaskTree(recipe.CurrentVersion.ID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	items := make([]dtos.TaskResponse, len(tasks))
	for i := range tasks {
		items[i] = dtos.TaskResponseFromModel(&tasks[i])
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"items": items,
		"total": len(items),
	})
}

// UpdateRecipe updates an existing recipe.
func (h *RecipeHandler) UpdateRecipe(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid recipe ID",
		})
	}

	// Verify recipe belongs to the requester's space before updating.
	recipe, err := h.getRecipeInSpace(c, id)
	if err != nil {
		return err
	}

	var dto dtos.UpdateRecipeRequest
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
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
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid recipe ID",
		})
	}

	// Verify recipe belongs to the requester's space before deleting.
	if _, err := h.getRecipeInSpace(c, id); err != nil {
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
	recipeID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid recipe ID",
		})
	}

	// Verify the recipe exists and belongs to the requester's space.
	if _, err := h.getRecipeInSpace(c, recipeID); err != nil {
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

// CreateVersion creates a new draft version for a recipe by cloning the current
// published version's task tree. The resulting draft can be edited via the task API.
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
	if _, err := h.getRecipeInSpace(c, recipeID); err != nil {
		return err
	}

	var dto dtos.CreateRecipeVersionRequest
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	version, err := h.authoringService.CreateDraftFromPublished(
		recipeID,
		authDTO.User.ID,
		dto.ChangeSummary,
	)
	if err != nil {
		return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{
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

	spaceID, err := spaceIDFromPath(c)
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
	if _, err := h.getRecipeInSpace(c, recipeID); err != nil {
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
		spaceID,
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

// RollRecipe rolls a published recipe into a new job task tree.
func (h *RecipeHandler) RollRecipe(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
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
	if _, err := h.getRecipeInSpace(c, recipeID); err != nil {
		return err
	}

	var dto dtos.RollRecipeRequest
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	opts := services.RollRecipeOptions{
		Title:      dto.Title,
		CustomerID: dto.CustomerID,
		DueDate:    dto.DueDate,
		OrderQty:   dto.OrderQty,
	}

	rootTask, err := h.rollService.RollRecipe(
		recipeID,
		spaceID,
		authDTO.User.ID,
		opts,
	)
	if err != nil {
		return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(dtos.TaskResponseFromModel(rootTask))
}

// PublishVersion publishes a recipe's draft version using clone-on-publish.
// The draft task tree is cloned into a frozen snapshot, then the version is
// marked as published.
func (h *RecipeHandler) PublishVersion(c *fiber.Ctx) error {
	recipeID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid recipe ID",
		})
	}

	// Verify the recipe exists and belongs to the requester's space.
	if _, err := h.getRecipeInSpace(c, recipeID); err != nil {
		return err
	}

	published, err := h.authoringService.PublishVersion(recipeID)
	if err != nil {
		return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(dtos.RecipeVersionResponseFromModel(published))
}

// PublishVersionFlat publishes a recipe version by its ID alone, without
// requiring the recipe ID in the URL path. Used by the flat
// POST /api/v1/spaces/:spaceId/recipe-versions/:id/publish route.
func (h *RecipeHandler) PublishVersionFlat(c *fiber.Ctx) error {
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

	// Verify the recipe belongs to the space from the URL path.
	if _, err := h.getRecipeInSpace(c, version.RecipeID); err != nil {
		return err
	}

	published, err := h.authoringService.PublishVersion(version.RecipeID)
	if err != nil {
		return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(dtos.RecipeVersionResponseFromModel(published))
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
