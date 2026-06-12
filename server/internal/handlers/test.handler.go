package handlers

import (
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

// TestHandler provides dev-only endpoints for test automation.
type TestHandler struct {
	db *gorm.DB
}

// NewTestHandler creates a new TestHandler.
func NewTestHandler(db *gorm.DB) *TestHandler {
	return &TestHandler{db: db}
}

// RegisterTestRoutes registers test routes. Only call this when NORI_ENV=development.
func (h *TestHandler) RegisterTestRoutes(app *fiber.App, middlewares ...fiber.Handler) {
	group := app.Group("/api/test", middlewares...)
	group.Post("/:spaceId/reset", h.ResetSpace)
}

// baselineStations defines the default stations re-seeded after a reset.
var baselineStations = []struct {
	Name     string
	WIPLimit int
}{
	{Name: "Cutting", WIPLimit: 3},
	{Name: "Assembly", WIPLimit: 2},
	{Name: "Finishing", WIPLimit: 2},
	{Name: "Quality Check", WIPLimit: 5},
}

// ResetSpace deletes all work data in the active space and re-seeds baseline
// stations. Only works when NORI_ENV=development. The space, user, and account
// are preserved.
func (h *TestHandler) ResetSpace(c *fiber.Ctx) error {
	// Runtime guard: reject if not development
	if os.Getenv("NORI_ENV") != "development" {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "test endpoints are only available in development mode",
		})
	}

	_, err := requireAuth(c)
	if err != nil {
		return err
	}

	spaceIDParam := c.Params("spaceId")
	if spaceIDParam == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "space ID is required in path",
		})
	}
	spaceID, err := uuid.Parse(spaceIDParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid space ID format",
		})
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		// Delete in FK-safe order. Tables without space_id use subqueries
		// scoped to tasks/recipes in this space.

		// 1. cost_entry (references task, time_event)
		if err := tx.Exec(
			"DELETE FROM cost_entry WHERE task_id IN (SELECT id FROM task WHERE space_id = ?)", spaceID,
		).Error; err != nil {
			return err
		}

		// 2. activity_entry (references task)
		if err := tx.Exec(
			"DELETE FROM activity_entry WHERE task_id IN (SELECT id FROM task WHERE space_id = ?)", spaceID,
		).Error; err != nil {
			return err
		}

		// 3. bom_item (references recipe_version)
		if err := tx.Exec(
			"DELETE FROM bom_item WHERE recipe_version_id IN (SELECT id FROM recipe_version WHERE recipe_id IN (SELECT id FROM recipe WHERE space_id = ?))", spaceID,
		).Error; err != nil {
			return err
		}

		// 4. time_event
		if err := tx.Exec("DELETE FROM time_event WHERE space_id = ?", spaceID).Error; err != nil {
			return err
		}

		// 5. task_tag (join table, references task)
		if err := tx.Exec(
			"DELETE FROM task_tag WHERE task_id IN (SELECT id FROM task WHERE space_id = ?)", spaceID,
		).Error; err != nil {
			return err
		}

		// 6. task_dep (references task)
		if err := tx.Exec(
			"DELETE FROM task_dep WHERE from_task_id IN (SELECT id FROM task WHERE space_id = ?) OR to_task_id IN (SELECT id FROM task WHERE space_id = ?)",
			spaceID, spaceID,
		).Error; err != nil {
			return err
		}

		// 7. Clear recipe.current_version_id to break circular FK
		if err := tx.Exec(
			"UPDATE recipe SET current_version_id = NULL WHERE space_id = ?", spaceID,
		).Error; err != nil {
			return err
		}

		// 8. recipe_version (references recipe and task via root_task_id)
		if err := tx.Exec(
			"DELETE FROM recipe_version WHERE recipe_id IN (SELECT id FROM recipe WHERE space_id = ?)", spaceID,
		).Error; err != nil {
			return err
		}

		// 9. recipe_tag (join table, references recipe)
		if err := tx.Exec(
			"DELETE FROM recipe_tag WHERE recipe_id IN (SELECT id FROM recipe WHERE space_id = ?)", spaceID,
		).Error; err != nil {
			return err
		}

		// 10. recipe
		if err := tx.Exec("DELETE FROM recipe WHERE space_id = ?", spaceID).Error; err != nil {
			return err
		}

		// 11. task — clear self-referential parent_id first, then delete
		if err := tx.Exec(
			"UPDATE task SET parent_id = NULL WHERE space_id = ?", spaceID,
		).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM task WHERE space_id = ?", spaceID).Error; err != nil {
			return err
		}

		// 12. Delete existing stations (will be re-seeded)
		if err := tx.Exec("DELETE FROM station WHERE space_id = ?", spaceID).Error; err != nil {
			return err
		}

		// 13. Re-seed baseline stations
		for i, s := range baselineStations {
			station := models.Station{
				ID:           uuid.New(),
				SpaceID:      spaceID,
				Name:         s.Name,
				DisplayOrder: i + 1,
				WIPLimit:     s.WIPLimit,
				IsActive:     true,
			}
			if err := tx.Create(&station).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to reset space: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{"reset": true})
}
