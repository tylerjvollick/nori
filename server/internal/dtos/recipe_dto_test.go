package dtos

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tylerjvollick/nori/internal/models"
)

// A recipe can be deleted between fetch and conversion (e.g. concurrent test
// reset). The converter must not panic on nil (nori-vyt).
func TestRecipeResponseFromModel_NilRecipe(t *testing.T) {
	assert.NotPanics(t, func() {
		resp := RecipeResponseFromModel(nil)
		assert.Equal(t, RecipeResponse{}, resp)
	})
}

func TestRecipeResponseFromModel_NameFallsBackToSlug(t *testing.T) {
	r := &models.Recipe{
		ID:   uuid.New(),
		Slug: "garden-bench",
	}

	resp := RecipeResponseFromModel(r)

	assert.Equal(t, "garden-bench", resp.Name)
	assert.Equal(t, "garden-bench", resp.Slug)
	assert.Nil(t, resp.CurrentVersion)
}
