package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/tylerjvollick/nori/internal/dbfactory"
	"github.com/tylerjvollick/nori/internal/dbtest"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
)

// The services package TestMain (product.service_test.go) starts the shared
// PostgreSQL container these tests run against.

func newTaskMaterialServiceForTest(t *testing.T) (*TaskMaterialService, models.Space, *gorm.DB) {
	t.Helper()
	tx := dbtest.TestTx(t)
	space := dbfactory.Space(tx)
	svc := NewTaskMaterialService(
		repositories.NewTaskMaterialRepository(tx),
		repositories.NewTaskRepository(tx),
		repositories.NewMaterialRepository(tx),
	)
	return svc, space, tx
}

func createTestMaterial(t *testing.T, tx *gorm.DB, spaceID uuid.UUID, name, unitCost string) models.Material {
	t.Helper()
	cost := decimal.RequireFromString(unitCost)
	m := models.Material{
		SpaceID:  spaceID,
		Name:     name,
		Unit:     "each",
		UnitCost: &cost,
	}
	require.NoError(t, tx.Create(&m).Error)
	return m
}

func TestTaskMaterialService_Add_Success(t *testing.T) {
	svc, space, tx := newTaskMaterialServiceForTest(t)
	task := dbfactory.Task(tx, func(tk *models.Task) { tk.SpaceID = space.ID })
	mat := createTestMaterial(t, tx, space.ID, "Walnut", "14.50")

	tm, err := svc.Add(space.ID, task.ID, &dtos.CreateTaskMaterialRequest{
		MaterialID:      mat.ID,
		QuantityPerUnit: decimal.NewFromInt(8),
		Notes:           strPtrTM("two per corner"),
	})
	require.NoError(t, err)
	assert.True(t, tm.QuantityPerUnit.Equal(decimal.NewFromInt(8)))
	require.NotNil(t, tm.Material, "Material should be preloaded")
	assert.Equal(t, "Walnut", tm.Material.Name)
}

func TestTaskMaterialService_Add_DuplicateRejected(t *testing.T) {
	svc, space, tx := newTaskMaterialServiceForTest(t)
	task := dbfactory.Task(tx, func(tk *models.Task) { tk.SpaceID = space.ID })
	mat := createTestMaterial(t, tx, space.ID, "Walnut", "14.50")

	_, err := svc.Add(space.ID, task.ID, &dtos.CreateTaskMaterialRequest{
		MaterialID: mat.ID, QuantityPerUnit: decimal.NewFromInt(1),
	})
	require.NoError(t, err)

	_, err = svc.Add(space.ID, task.ID, &dtos.CreateTaskMaterialRequest{
		MaterialID: mat.ID, QuantityPerUnit: decimal.NewFromInt(2),
	})
	assert.ErrorIs(t, err, ErrTaskMaterialDuplicate)
}

func TestTaskMaterialService_Add_NonPositiveQuantityRejected(t *testing.T) {
	svc, space, tx := newTaskMaterialServiceForTest(t)
	task := dbfactory.Task(tx, func(tk *models.Task) { tk.SpaceID = space.ID })
	mat := createTestMaterial(t, tx, space.ID, "Walnut", "14.50")

	for _, q := range []decimal.Decimal{decimal.Zero, decimal.NewFromInt(-1)} {
		_, err := svc.Add(space.ID, task.ID, &dtos.CreateTaskMaterialRequest{
			MaterialID: mat.ID, QuantityPerUnit: q,
		})
		var verr *TaskMaterialValidationError
		require.ErrorAs(t, err, &verr)
		assert.Contains(t, verr.Message, "quantityPerUnit must be > 0")
	}
}

func TestTaskMaterialService_Add_TaskInOtherSpaceNotFound(t *testing.T) {
	svc, space, tx := newTaskMaterialServiceForTest(t)
	otherSpace := dbfactory.Space(tx)
	task := dbfactory.Task(tx, func(tk *models.Task) { tk.SpaceID = otherSpace.ID })
	mat := createTestMaterial(t, tx, space.ID, "Walnut", "14.50")

	_, err := svc.Add(space.ID, task.ID, &dtos.CreateTaskMaterialRequest{
		MaterialID: mat.ID, QuantityPerUnit: decimal.NewFromInt(1),
	})
	assert.ErrorIs(t, err, ErrTaskMaterialNotFound)
}

func TestTaskMaterialService_Add_MaterialInOtherSpaceRejected(t *testing.T) {
	svc, space, tx := newTaskMaterialServiceForTest(t)
	otherSpace := dbfactory.Space(tx)
	task := dbfactory.Task(tx, func(tk *models.Task) { tk.SpaceID = space.ID })
	mat := createTestMaterial(t, tx, otherSpace.ID, "Walnut", "14.50")

	_, err := svc.Add(space.ID, task.ID, &dtos.CreateTaskMaterialRequest{
		MaterialID: mat.ID, QuantityPerUnit: decimal.NewFromInt(1),
	})
	var verr *TaskMaterialValidationError
	require.ErrorAs(t, err, &verr)
	assert.Contains(t, verr.Message, "material not found in space")
}

func TestTaskMaterialService_List(t *testing.T) {
	svc, space, tx := newTaskMaterialServiceForTest(t)
	task := dbfactory.Task(tx, func(tk *models.Task) { tk.SpaceID = space.ID })
	walnut := createTestMaterial(t, tx, space.ID, "Walnut", "14.50")
	glue := createTestMaterial(t, tx, space.ID, "Titebond", "0.55")

	_, err := svc.Add(space.ID, task.ID, &dtos.CreateTaskMaterialRequest{MaterialID: walnut.ID, QuantityPerUnit: decimal.NewFromInt(8)})
	require.NoError(t, err)
	_, err = svc.Add(space.ID, task.ID, &dtos.CreateTaskMaterialRequest{MaterialID: glue.ID, QuantityPerUnit: decimal.NewFromInt(2)})
	require.NoError(t, err)

	list, err := svc.List(space.ID, task.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.NotNil(t, list[0].Material)
}

func TestTaskMaterialService_Update(t *testing.T) {
	svc, space, tx := newTaskMaterialServiceForTest(t)
	task := dbfactory.Task(tx, func(tk *models.Task) { tk.SpaceID = space.ID })
	mat := createTestMaterial(t, tx, space.ID, "Walnut", "14.50")

	tm, err := svc.Add(space.ID, task.ID, &dtos.CreateTaskMaterialRequest{MaterialID: mat.ID, QuantityPerUnit: decimal.NewFromInt(8)})
	require.NoError(t, err)

	newQty := decimal.NewFromInt(12)
	updated, err := svc.Update(space.ID, task.ID, tm.ID, &dtos.UpdateTaskMaterialRequest{QuantityPerUnit: &newQty})
	require.NoError(t, err)
	assert.True(t, updated.QuantityPerUnit.Equal(decimal.NewFromInt(12)))
}

func TestTaskMaterialService_Update_NonPositiveRejected(t *testing.T) {
	svc, space, tx := newTaskMaterialServiceForTest(t)
	task := dbfactory.Task(tx, func(tk *models.Task) { tk.SpaceID = space.ID })
	mat := createTestMaterial(t, tx, space.ID, "Walnut", "14.50")

	tm, err := svc.Add(space.ID, task.ID, &dtos.CreateTaskMaterialRequest{MaterialID: mat.ID, QuantityPerUnit: decimal.NewFromInt(8)})
	require.NoError(t, err)

	zero := decimal.Zero
	_, err = svc.Update(space.ID, task.ID, tm.ID, &dtos.UpdateTaskMaterialRequest{QuantityPerUnit: &zero})
	var verr *TaskMaterialValidationError
	require.ErrorAs(t, err, &verr)
}

func TestTaskMaterialService_Remove(t *testing.T) {
	svc, space, tx := newTaskMaterialServiceForTest(t)
	task := dbfactory.Task(tx, func(tk *models.Task) { tk.SpaceID = space.ID })
	mat := createTestMaterial(t, tx, space.ID, "Walnut", "14.50")

	tm, err := svc.Add(space.ID, task.ID, &dtos.CreateTaskMaterialRequest{MaterialID: mat.ID, QuantityPerUnit: decimal.NewFromInt(8)})
	require.NoError(t, err)

	require.NoError(t, svc.Remove(space.ID, task.ID, tm.ID))

	list, err := svc.List(space.ID, task.ID)
	require.NoError(t, err)
	assert.Len(t, list, 0)
}

func strPtrTM(s string) *string { return &s }
