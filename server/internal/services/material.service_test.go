package services

import (
	"errors"
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

func newMaterialServiceForTest(t *testing.T) (*MaterialService, models.Space, *gorm.DB) {
	t.Helper()
	tx := dbtest.TestTx(t)
	space := dbfactory.Space(tx)
	return NewMaterialService(repositories.NewMaterialRepository(tx)), space, tx
}

func matStrPtr(s string) *string { return &s }

func matDecPtr(s string) *decimal.Decimal {
	d := decimal.RequireFromString(s)
	return &d
}

// --- Create ---

func TestMaterialService_Create_Success(t *testing.T) {
	svc, space, tx := newMaterialServiceForTest(t)

	material, err := svc.Create(space.ID, &dtos.CreateMaterialRequest{
		Name:     "8/4 Walnut",
		Category: matStrPtr("lumber"),
		Unit:     "board_feet",
		Supplier: matStrPtr("Goby Walnut"),
		SKU:      matStrPtr("WAL-84"),
		UnitCost: matDecPtr("14.50"),
	})
	require.NoError(t, err)
	require.NotNil(t, material)

	// Verify persisted values via a fresh read.
	var stored models.Material
	require.NoError(t, tx.First(&stored, "id = ?", material.ID).Error)
	assert.Equal(t, space.ID, stored.SpaceID)
	assert.Equal(t, "8/4 Walnut", stored.Name)
	assert.Equal(t, models.MaterialCategoryLumber, stored.Category)
	assert.Equal(t, "board_feet", stored.Unit)
	require.NotNil(t, stored.Supplier)
	assert.Equal(t, "Goby Walnut", *stored.Supplier)
	require.NotNil(t, stored.SKU)
	assert.Equal(t, "WAL-84", *stored.SKU)
	require.NotNil(t, stored.UnitCost)
	assert.True(t, stored.UnitCost.Equal(decimal.RequireFromString("14.50")))
	assert.True(t, stored.IsActive)
}

func TestMaterialService_Create_DefaultsCategoryToOther(t *testing.T) {
	svc, space, _ := newMaterialServiceForTest(t)

	material, err := svc.Create(space.ID, &dtos.CreateMaterialRequest{
		Name: "Titebond III",
		Unit: "oz",
	})
	require.NoError(t, err)
	assert.Equal(t, models.MaterialCategoryOther, material.Category)
}

func TestMaterialService_Create_MissingName(t *testing.T) {
	svc, space, _ := newMaterialServiceForTest(t)

	_, err := svc.Create(space.ID, &dtos.CreateMaterialRequest{Name: "  ", Unit: "each"})
	require.Error(t, err)

	var validationErr *MaterialValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "name is required", validationErr.Message)
}

func TestMaterialService_Create_MissingUnit(t *testing.T) {
	svc, space, _ := newMaterialServiceForTest(t)

	_, err := svc.Create(space.ID, &dtos.CreateMaterialRequest{Name: "Lacquer", Unit: ""})
	require.Error(t, err)

	var validationErr *MaterialValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "unit is required", validationErr.Message)
}

func TestMaterialService_Create_NegativeCost(t *testing.T) {
	svc, space, _ := newMaterialServiceForTest(t)

	_, err := svc.Create(space.ID, &dtos.CreateMaterialRequest{
		Name:     "Lacquer",
		Unit:     "gallons",
		UnitCost: matDecPtr("-1.00"),
	})
	require.Error(t, err)

	var validationErr *MaterialValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "unitCost must be >= 0", validationErr.Message)
}

func TestMaterialService_Create_InvalidCategory(t *testing.T) {
	svc, space, _ := newMaterialServiceForTest(t)

	_, err := svc.Create(space.ID, &dtos.CreateMaterialRequest{
		Name:     "Lacquer",
		Unit:     "gallons",
		Category: matStrPtr("plasma"),
	})
	require.Error(t, err)

	var validationErr *MaterialValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Contains(t, validationErr.Message, "invalid category")
}

// --- List ---

func TestMaterialService_List_ScopedToSpaceAndOrdered(t *testing.T) {
	svc, space, tx := newMaterialServiceForTest(t)
	otherSpace := dbfactory.Space(tx)

	for _, name := range []string{"Walnut", "Cherry", "Lacquer"} {
		_, err := svc.Create(space.ID, &dtos.CreateMaterialRequest{Name: name, Unit: "each"})
		require.NoError(t, err)
	}
	_, err := svc.Create(otherSpace.ID, &dtos.CreateMaterialRequest{Name: "Other Space Material", Unit: "each"})
	require.NoError(t, err)

	materials, err := svc.List(space.ID, "")
	require.NoError(t, err)
	require.Len(t, materials, 3)
	assert.Equal(t, "Cherry", materials[0].Name)
	assert.Equal(t, "Lacquer", materials[1].Name)
	assert.Equal(t, "Walnut", materials[2].Name)
}

func TestMaterialService_List_SearchFiltersByNameCaseInsensitive(t *testing.T) {
	svc, space, _ := newMaterialServiceForTest(t)

	for _, name := range []string{"8/4 Walnut", "4/4 Cherry", "Walnut Plugs"} {
		_, err := svc.Create(space.ID, &dtos.CreateMaterialRequest{Name: name, Unit: "each"})
		require.NoError(t, err)
	}

	materials, err := svc.List(space.ID, "walnut")
	require.NoError(t, err)
	require.Len(t, materials, 2)
	assert.Equal(t, "8/4 Walnut", materials[0].Name)
	assert.Equal(t, "Walnut Plugs", materials[1].Name)
}

func TestMaterialService_List_SearchEscapesWildcards(t *testing.T) {
	svc, space, _ := newMaterialServiceForTest(t)

	_, err := svc.Create(space.ID, &dtos.CreateMaterialRequest{Name: "100% Tung Oil", Unit: "oz"})
	require.NoError(t, err)
	_, err = svc.Create(space.ID, &dtos.CreateMaterialRequest{Name: "Shellac", Unit: "oz"})
	require.NoError(t, err)

	materials, err := svc.List(space.ID, "100%")
	require.NoError(t, err)
	require.Len(t, materials, 1)
	assert.Equal(t, "100% Tung Oil", materials[0].Name)
}

// --- GetByID ---

func TestMaterialService_GetByID_Success(t *testing.T) {
	svc, space, _ := newMaterialServiceForTest(t)

	created, err := svc.Create(space.ID, &dtos.CreateMaterialRequest{Name: "Brass Screws", Unit: "each"})
	require.NoError(t, err)

	material, err := svc.GetByID(space.ID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, material.ID)
	assert.Equal(t, "Brass Screws", material.Name)
}

func TestMaterialService_GetByID_NotFound(t *testing.T) {
	svc, space, _ := newMaterialServiceForTest(t)

	_, err := svc.GetByID(space.ID, uuid.New())
	assert.ErrorIs(t, err, ErrMaterialNotFound)
}

func TestMaterialService_GetByID_WrongSpace(t *testing.T) {
	svc, space, tx := newMaterialServiceForTest(t)
	otherSpace := dbfactory.Space(tx)

	created, err := svc.Create(space.ID, &dtos.CreateMaterialRequest{Name: "Brass Screws", Unit: "each"})
	require.NoError(t, err)

	_, err = svc.GetByID(otherSpace.ID, created.ID)
	assert.ErrorIs(t, err, ErrMaterialNotFound)
}

// --- Update ---

func TestMaterialService_Update_Success(t *testing.T) {
	svc, space, tx := newMaterialServiceForTest(t)

	created, err := svc.Create(space.ID, &dtos.CreateMaterialRequest{
		Name:     "8/4 Walnut",
		Unit:     "board_feet",
		UnitCost: matDecPtr("14.50"),
	})
	require.NoError(t, err)

	updated, err := svc.Update(space.ID, created.ID, &dtos.UpdateMaterialRequest{
		UnitCost: matDecPtr("15.00"),
		Supplier: matStrPtr("Goby Walnut"),
	})
	require.NoError(t, err)
	require.NotNil(t, updated.UnitCost)
	assert.True(t, updated.UnitCost.Equal(decimal.RequireFromString("15.00")))
	require.NotNil(t, updated.Supplier)
	assert.Equal(t, "Goby Walnut", *updated.Supplier)
	assert.Equal(t, "8/4 Walnut", updated.Name) // unchanged

	// Verify persistence.
	var stored models.Material
	require.NoError(t, tx.First(&stored, "id = ?", created.ID).Error)
	require.NotNil(t, stored.UnitCost)
	assert.True(t, stored.UnitCost.Equal(decimal.RequireFromString("15.00")))
}

func TestMaterialService_Update_EmptyNameRejected(t *testing.T) {
	svc, space, _ := newMaterialServiceForTest(t)

	created, err := svc.Create(space.ID, &dtos.CreateMaterialRequest{Name: "Lacquer", Unit: "gallons"})
	require.NoError(t, err)

	_, err = svc.Update(space.ID, created.ID, &dtos.UpdateMaterialRequest{Name: matStrPtr("  ")})
	require.Error(t, err)

	var validationErr *MaterialValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "name is required", validationErr.Message)
}

func TestMaterialService_Update_NegativeCostRejected(t *testing.T) {
	svc, space, _ := newMaterialServiceForTest(t)

	created, err := svc.Create(space.ID, &dtos.CreateMaterialRequest{Name: "Lacquer", Unit: "gallons"})
	require.NoError(t, err)

	_, err = svc.Update(space.ID, created.ID, &dtos.UpdateMaterialRequest{UnitCost: matDecPtr("-0.01")})
	require.Error(t, err)

	var validationErr *MaterialValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "unitCost must be >= 0", validationErr.Message)
}

func TestMaterialService_Update_NotFound(t *testing.T) {
	svc, space, _ := newMaterialServiceForTest(t)

	_, err := svc.Update(space.ID, uuid.New(), &dtos.UpdateMaterialRequest{Name: matStrPtr("New")})
	assert.ErrorIs(t, err, ErrMaterialNotFound)
}

func TestMaterialService_Update_WrongSpace(t *testing.T) {
	svc, space, tx := newMaterialServiceForTest(t)
	otherSpace := dbfactory.Space(tx)

	created, err := svc.Create(space.ID, &dtos.CreateMaterialRequest{Name: "Lacquer", Unit: "gallons"})
	require.NoError(t, err)

	_, err = svc.Update(otherSpace.ID, created.ID, &dtos.UpdateMaterialRequest{Name: matStrPtr("New")})
	assert.ErrorIs(t, err, ErrMaterialNotFound)
}

// --- Delete ---

func TestMaterialService_Delete_SoftDeletesAndExcludesFromList(t *testing.T) {
	svc, space, tx := newMaterialServiceForTest(t)

	created, err := svc.Create(space.ID, &dtos.CreateMaterialRequest{Name: "8/4 Walnut", Unit: "board_feet"})
	require.NoError(t, err)
	kept, err := svc.Create(space.ID, &dtos.CreateMaterialRequest{Name: "4/4 Cherry", Unit: "board_feet"})
	require.NoError(t, err)

	require.NoError(t, svc.Delete(space.ID, created.ID))

	// Excluded from list.
	materials, err := svc.List(space.ID, "")
	require.NoError(t, err)
	require.Len(t, materials, 1)
	assert.Equal(t, kept.ID, materials[0].ID)

	// GetByID treats it as gone.
	_, err = svc.GetByID(space.ID, created.ID)
	assert.ErrorIs(t, err, ErrMaterialNotFound)

	// Row still exists with deleted_at set (soft delete, not hard delete).
	var stored models.Material
	require.NoError(t, tx.Unscoped().First(&stored, "id = ?", created.ID).Error)
	assert.True(t, stored.DeletedAt.Valid, "deleted_at should be set")
}

func TestMaterialService_Delete_NotFound(t *testing.T) {
	svc, space, _ := newMaterialServiceForTest(t)

	err := svc.Delete(space.ID, uuid.New())
	assert.ErrorIs(t, err, ErrMaterialNotFound)
}

func TestMaterialService_Delete_WrongSpace(t *testing.T) {
	svc, space, tx := newMaterialServiceForTest(t)
	otherSpace := dbfactory.Space(tx)

	created, err := svc.Create(space.ID, &dtos.CreateMaterialRequest{Name: "Lacquer", Unit: "gallons"})
	require.NoError(t, err)

	err = svc.Delete(otherSpace.ID, created.ID)
	assert.ErrorIs(t, err, ErrMaterialNotFound)
}
