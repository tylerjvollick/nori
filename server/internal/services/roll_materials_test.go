package services

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tylerjvollick/nori/internal/dbfactory"
	"github.com/tylerjvollick/nori/internal/dbtest"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
)

// TestRollRecipe_ClonesTaskMaterialsWithSnapshot rolls a published recipe whose
// task has an attached material and verifies the job task receives a cloned
// task_material with the material's current unit cost snapshotted (nori-xz5.3).
func TestRollRecipe_ClonesTaskMaterialsWithSnapshot(t *testing.T) {
	tx := dbtest.TestTx(t)

	space := dbfactory.Space(tx)
	user := dbfactory.User(tx)
	recipe := dbfactory.Recipe(tx, func(r *models.Recipe) {
		r.SpaceID = space.ID
		r.CreatedByID = user.ID
	})

	// Build a minimal published task tree: root (recipe) + one child (task).
	rootID := "nori-rmt1"
	childID := rootID + ".1"
	root := models.Task{
		ID: rootID, SpaceID: space.ID, CreatedByID: user.ID,
		Type: models.TaskTypeRecipe, Status: models.TaskStatusOpen, Title: "Dining Chair",
	}
	require.NoError(t, tx.Create(&root).Error)
	child := models.Task{
		ID: childID, ParentID: &rootID, SpaceID: space.ID, CreatedByID: user.ID,
		Type: models.TaskTypeTask, Status: models.TaskStatusOpen, Title: "Cut legs",
	}
	require.NoError(t, tx.Create(&child).Error)

	version := dbfactory.RecipeVersion(tx, func(v *models.RecipeVersion) {
		v.RecipeID = recipe.ID
		v.AuthorID = user.ID
		v.Status = models.RecipeVersionStatusPublished
		v.RootTaskID = &rootID
	})
	require.NoError(t, tx.Model(&models.Recipe{}).Where("id = ?", recipe.ID).
		Update("current_version_id", version.ID).Error)

	// Material in the catalog with a unit cost, attached to the recipe child task.
	unitCost := decimal.RequireFromString("14.50")
	material := models.Material{SpaceID: space.ID, Name: "8/4 Walnut", Unit: "board_feet", UnitCost: &unitCost}
	require.NoError(t, tx.Create(&material).Error)
	recipeTaskMaterial := models.TaskMaterial{
		TaskID: childID, MaterialID: material.ID, QuantityPerUnit: decimal.NewFromInt(8),
	}
	require.NoError(t, tx.Create(&recipeTaskMaterial).Error)

	svc := NewRecipeService(
		tx,
		repositories.NewRecipeRepository(tx),
		repositories.NewTaskRepository(tx),
		repositories.NewTaskDepRepository(tx),
	)

	job, err := svc.RollRecipe(recipe.ID, space.ID, user.ID, RollRecipeOptions{})
	require.NoError(t, err)

	// The recipe's task material stays un-snapshotted.
	var recipeTM models.TaskMaterial
	require.NoError(t, tx.First(&recipeTM, "id = ?", recipeTaskMaterial.ID).Error)
	assert.Nil(t, recipeTM.SnapshottedUnitCost, "recipe task material must not be snapshotted")

	// The job's cloned child task should have a cloned material with snapshot.
	jobChildID := job.ID + ".1"
	var jobMaterials []models.TaskMaterial
	require.NoError(t, tx.Where("task_id = ?", jobChildID).Find(&jobMaterials).Error)
	require.Len(t, jobMaterials, 1, "job task should have one cloned material")

	clone := jobMaterials[0]
	assert.Equal(t, material.ID, clone.MaterialID)
	assert.True(t, clone.QuantityPerUnit.Equal(decimal.NewFromInt(8)))
	require.NotNil(t, clone.SnapshottedUnitCost, "job material must snapshot the unit cost")
	assert.True(t, clone.SnapshottedUnitCost.Equal(unitCost),
		"snapshot should equal the material's current unit cost")
}

// TestRollRecipe_NoMaterialsClonesNothing ensures rolling a recipe without
// task materials does not error and produces no task_material rows.
func TestRollRecipe_NoMaterialsClonesNothing(t *testing.T) {
	tx := dbtest.TestTx(t)

	space := dbfactory.Space(tx)
	user := dbfactory.User(tx)
	recipe := dbfactory.Recipe(tx, func(r *models.Recipe) {
		r.SpaceID = space.ID
		r.CreatedByID = user.ID
	})

	rootID := "nori-rmt2"
	root := models.Task{
		ID: rootID, SpaceID: space.ID, CreatedByID: user.ID,
		Type: models.TaskTypeRecipe, Status: models.TaskStatusOpen, Title: "Bare Recipe",
	}
	require.NoError(t, tx.Create(&root).Error)

	version := dbfactory.RecipeVersion(tx, func(v *models.RecipeVersion) {
		v.RecipeID = recipe.ID
		v.AuthorID = user.ID
		v.Status = models.RecipeVersionStatusPublished
		v.RootTaskID = &rootID
	})
	require.NoError(t, tx.Model(&models.Recipe{}).Where("id = ?", recipe.ID).
		Update("current_version_id", version.ID).Error)

	svc := NewRecipeService(
		tx,
		repositories.NewRecipeRepository(tx),
		repositories.NewTaskRepository(tx),
		repositories.NewTaskDepRepository(tx),
	)

	job, err := svc.RollRecipe(recipe.ID, space.ID, user.ID, RollRecipeOptions{})
	require.NoError(t, err)

	var count int64
	require.NoError(t, tx.Model(&models.TaskMaterial{}).
		Where("task_id = ? OR task_id LIKE ?", job.ID, job.ID+".%").Count(&count).Error)
	assert.Equal(t, int64(0), count)
}
