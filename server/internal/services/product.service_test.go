package services

import (
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/tylerjvollick/nori/internal/dbfactory"
	"github.com/tylerjvollick/nori/internal/dbtest"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
)

// TestMain starts a shared PostgreSQL test container for the services
// package. Tests that need a real database use dbtest.TestTx(t).
func TestMain(m *testing.M) {
	// Go test runs from the package directory (server/internal/services/).
	// Migrations are at server/migrations/, two levels up.
	os.Exit(dbtest.SetupTestMain(m, os.DirFS("../..")))
}

// newProductService builds a ProductService bound to the test transaction.
func newProductService(tx *gorm.DB) *ProductService {
	return NewProductService(tx, repositories.NewRecipeRepository(tx))
}

// --- Create ---

func TestProductService_Create(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)

	desc := "Solid hardwood dining table"
	product, err := svc.Create(space.ID, CreateProductInput{
		Name:        "Dining Table",
		Description: &desc,
	})
	require.NoError(t, err)
	assert.Equal(t, "Dining Table", product.Name)
	assert.Equal(t, space.ID, product.SpaceID)
	require.NotNil(t, product.Description)
	assert.Equal(t, desc, *product.Description)
	assert.True(t, product.IsActive)

	var count int64
	tx.Model(&models.Product{}).Where("id = ?", product.ID).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestProductService_Create_EmptyName(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)

	_, err := svc.Create(space.ID, CreateProductInput{Name: ""})
	assert.ErrorIs(t, err, ErrProductNameRequired)
}

// --- List ---

func TestProductService_List(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	otherSpace := dbfactory.Space(tx)

	dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID; p.Name = "B Table" })
	dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID; p.Name = "A Chair" })
	dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = otherSpace.ID; p.Name = "Other Shelf" })

	products, err := svc.List(space.ID)
	require.NoError(t, err)
	require.Len(t, products, 2)
	// Ordered by name.
	assert.Equal(t, "A Chair", products[0].Name)
	assert.Equal(t, "B Table", products[1].Name)
}

func TestProductService_List_ExcludesSoftDeleted(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)

	keep := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })
	deleted := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })
	require.NoError(t, svc.Delete(space.ID, deleted.ID))

	products, err := svc.List(space.ID)
	require.NoError(t, err)
	require.Len(t, products, 1)
	assert.Equal(t, keep.ID, products[0].ID)
}

// --- GetByID ---

func TestProductService_GetByID_IncludesVariants(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })

	dbfactory.ProductVariant(tx, func(v *models.ProductVariant) {
		v.ProductID = product.ID
		v.Name = "Walnut / Spray PU"
	})
	dbfactory.ProductVariant(tx, func(v *models.ProductVariant) {
		v.ProductID = product.ID
		v.Name = "Oak / Hand Oil"
	})

	got, err := svc.GetByID(space.ID, product.ID)
	require.NoError(t, err)
	assert.Equal(t, product.ID, got.ID)
	require.Len(t, got.Variants, 2)
}

func TestProductService_GetByID_NotFound(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)

	_, err := svc.GetByID(space.ID, uuid.New())
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestProductService_GetByID_CrossSpaceReturnsNotFound(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	otherSpace := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = otherSpace.ID })

	_, err := svc.GetByID(space.ID, product.ID)
	assert.ErrorIs(t, err, ErrProductNotFound)
}

// --- Update ---

func TestProductService_Update(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })

	newName := "Dining Table v2"
	newDesc := "Updated description"
	inactive := false
	updated, err := svc.Update(space.ID, product.ID, UpdateProductInput{
		Name:        &newName,
		Description: &newDesc,
		IsActive:    &inactive,
	})
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
	require.NotNil(t, updated.Description)
	assert.Equal(t, newDesc, *updated.Description)
	assert.False(t, updated.IsActive)

	// Verify persisted.
	var fromDB models.Product
	require.NoError(t, tx.First(&fromDB, "id = ?", product.ID).Error)
	assert.Equal(t, newName, fromDB.Name)
	assert.False(t, fromDB.IsActive)
}

func TestProductService_Update_PartialKeepsOtherFields(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	desc := "Original"
	product := dbfactory.Product(tx, func(p *models.Product) {
		p.SpaceID = space.ID
		p.Name = "Original Name"
		p.Description = &desc
	})

	inactive := false
	updated, err := svc.Update(space.ID, product.ID, UpdateProductInput{IsActive: &inactive})
	require.NoError(t, err)
	assert.Equal(t, "Original Name", updated.Name)
	require.NotNil(t, updated.Description)
	assert.Equal(t, desc, *updated.Description)
	assert.False(t, updated.IsActive)
}

func TestProductService_Update_CrossSpaceReturnsNotFound(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	otherSpace := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = otherSpace.ID })

	newName := "Hijacked"
	_, err := svc.Update(space.ID, product.ID, UpdateProductInput{Name: &newName})
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestProductService_Update_EmptyName(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })

	empty := ""
	_, err := svc.Update(space.ID, product.ID, UpdateProductInput{Name: &empty})
	assert.ErrorIs(t, err, ErrProductNameRequired)
}

// --- Delete ---

func TestProductService_Delete_SoftDeletesAndDeactivatesVariants(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })
	v1 := dbfactory.ProductVariant(tx, func(v *models.ProductVariant) { v.ProductID = product.ID })
	v2 := dbfactory.ProductVariant(tx, func(v *models.ProductVariant) { v.ProductID = product.ID })

	require.NoError(t, svc.Delete(space.ID, product.ID))

	// Product is soft-deleted: gone from default queries, present unscoped.
	_, err := svc.GetByID(space.ID, product.ID)
	assert.ErrorIs(t, err, ErrProductNotFound)

	var fromDB models.Product
	require.NoError(t, tx.Unscoped().First(&fromDB, "id = ?", product.ID).Error)
	assert.True(t, fromDB.DeletedAt.Valid, "expected deleted_at to be set")

	// Variants still exist but are deactivated.
	for _, id := range []uuid.UUID{v1.ID, v2.ID} {
		var variant models.ProductVariant
		require.NoError(t, tx.First(&variant, "id = ?", id).Error)
		assert.False(t, variant.IsActive, "expected variant %s to be inactive", id)
	}
}

func TestProductService_Delete_CrossSpaceReturnsNotFound(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	otherSpace := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = otherSpace.ID })

	err := svc.Delete(space.ID, product.ID)
	assert.ErrorIs(t, err, ErrProductNotFound)

	// Untouched in its own space.
	var fromDB models.Product
	require.NoError(t, tx.First(&fromDB, "id = ?", product.ID).Error)
}

// --- CreateVariant ---

func TestProductService_CreateVariant(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })
	recipe := dbfactory.Recipe(tx, func(r *models.Recipe) { r.SpaceID = space.ID })

	price := decimal.NewFromInt(4200)
	variant, err := svc.CreateVariant(space.ID, product.ID, CreateVariantInput{
		Name:     "Walnut / Spray PU",
		RecipeID: &recipe.ID,
		RecipeVariables: models.JSONB{
			"wood_species": "Walnut",
			"finish":       "Spray PU",
		},
		Price: &price,
	})
	require.NoError(t, err)
	assert.Equal(t, product.ID, variant.ProductID)
	assert.Equal(t, "Walnut / Spray PU", variant.Name)
	require.NotNil(t, variant.RecipeID)
	assert.Equal(t, recipe.ID, *variant.RecipeID)
	assert.True(t, variant.IsActive)

	// Verify the JSONB and price round-trip through PostgreSQL.
	var fromDB models.ProductVariant
	require.NoError(t, tx.First(&fromDB, "id = ?", variant.ID).Error)
	assert.Equal(t, "Walnut", fromDB.RecipeVariables["wood_species"])
	assert.Equal(t, "Spray PU", fromDB.RecipeVariables["finish"])
	require.NotNil(t, fromDB.Price)
	assert.True(t, fromDB.Price.Equal(price), "expected price 4200, got %s", fromDB.Price)
}

func TestProductService_CreateVariant_WithoutRecipe(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })

	variant, err := svc.CreateVariant(space.ID, product.ID, CreateVariantInput{
		Name: "Standard",
	})
	require.NoError(t, err)
	assert.Nil(t, variant.RecipeID)
	assert.Nil(t, variant.Price)
}

func TestProductService_CreateVariant_EmptyName(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })

	_, err := svc.CreateVariant(space.ID, product.ID, CreateVariantInput{Name: ""})
	assert.ErrorIs(t, err, ErrProductNameRequired)
}

func TestProductService_CreateVariant_RecipeInOtherSpace(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	otherSpace := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })
	foreignRecipe := dbfactory.Recipe(tx, func(r *models.Recipe) { r.SpaceID = otherSpace.ID })

	_, err := svc.CreateVariant(space.ID, product.ID, CreateVariantInput{
		Name:     "Walnut",
		RecipeID: &foreignRecipe.ID,
	})
	assert.ErrorIs(t, err, ErrVariantRecipeNotFound)
}

func TestProductService_CreateVariant_RecipeDoesNotExist(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })

	missing := uuid.New()
	_, err := svc.CreateVariant(space.ID, product.ID, CreateVariantInput{
		Name:     "Walnut",
		RecipeID: &missing,
	})
	assert.ErrorIs(t, err, ErrVariantRecipeNotFound)
}

func TestProductService_CreateVariant_ProductInOtherSpace(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	otherSpace := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = otherSpace.ID })

	_, err := svc.CreateVariant(space.ID, product.ID, CreateVariantInput{Name: "Walnut"})
	assert.ErrorIs(t, err, ErrProductNotFound)
}

// --- UpdateVariant ---

func TestProductService_UpdateVariant_PriceOnly(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })
	oldPrice := decimal.NewFromInt(4200)
	variant := dbfactory.ProductVariant(tx, func(v *models.ProductVariant) {
		v.ProductID = product.ID
		v.Name = "Walnut / Spray PU"
		v.Price = &oldPrice
	})

	newPrice := decimal.NewFromInt(4500)
	updated, err := svc.UpdateVariant(space.ID, product.ID, variant.ID, UpdateVariantInput{
		Price: &newPrice,
	})
	require.NoError(t, err)
	assert.Equal(t, "Walnut / Spray PU", updated.Name) // unchanged
	require.NotNil(t, updated.Price)
	assert.True(t, updated.Price.Equal(newPrice))

	var fromDB models.ProductVariant
	require.NoError(t, tx.First(&fromDB, "id = ?", variant.ID).Error)
	assert.True(t, fromDB.Price.Equal(newPrice))
}

func TestProductService_UpdateVariant_AllFields(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })
	recipe := dbfactory.Recipe(tx, func(r *models.Recipe) { r.SpaceID = space.ID })
	variant := dbfactory.ProductVariant(tx, func(v *models.ProductVariant) { v.ProductID = product.ID })

	newName := "Cherry / Hand Oil"
	price := decimal.NewFromInt(580)
	inactive := false
	updated, err := svc.UpdateVariant(space.ID, product.ID, variant.ID, UpdateVariantInput{
		Name:            &newName,
		RecipeID:        &recipe.ID,
		RecipeVariables: models.JSONB{"wood_species": "Cherry"},
		Price:           &price,
		IsActive:        &inactive,
	})
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
	require.NotNil(t, updated.RecipeID)
	assert.Equal(t, recipe.ID, *updated.RecipeID)
	assert.Equal(t, "Cherry", updated.RecipeVariables["wood_species"])
	assert.False(t, updated.IsActive)
}

func TestProductService_UpdateVariant_RecipeInOtherSpace(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	otherSpace := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })
	variant := dbfactory.ProductVariant(tx, func(v *models.ProductVariant) { v.ProductID = product.ID })
	foreignRecipe := dbfactory.Recipe(tx, func(r *models.Recipe) { r.SpaceID = otherSpace.ID })

	_, err := svc.UpdateVariant(space.ID, product.ID, variant.ID, UpdateVariantInput{
		RecipeID: &foreignRecipe.ID,
	})
	assert.ErrorIs(t, err, ErrVariantRecipeNotFound)
}

func TestProductService_UpdateVariant_WrongProduct(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })
	otherProduct := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })
	variant := dbfactory.ProductVariant(tx, func(v *models.ProductVariant) { v.ProductID = otherProduct.ID })

	newName := "Hijacked"
	_, err := svc.UpdateVariant(space.ID, product.ID, variant.ID, UpdateVariantInput{Name: &newName})
	assert.ErrorIs(t, err, ErrProductVariantNotFound)
}

func TestProductService_UpdateVariant_ProductInOtherSpace(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	otherSpace := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = otherSpace.ID })
	variant := dbfactory.ProductVariant(tx, func(v *models.ProductVariant) { v.ProductID = product.ID })

	newName := "Hijacked"
	_, err := svc.UpdateVariant(space.ID, product.ID, variant.ID, UpdateVariantInput{Name: &newName})
	assert.ErrorIs(t, err, ErrProductNotFound)
}

// --- DeleteVariant ---

func TestProductService_DeleteVariant(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })
	variant := dbfactory.ProductVariant(tx, func(v *models.ProductVariant) { v.ProductID = product.ID })

	require.NoError(t, svc.DeleteVariant(space.ID, product.ID, variant.ID))

	// Hard delete: row is gone.
	err := tx.First(&models.ProductVariant{}, "id = ?", variant.ID).Error
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound), "expected variant row to be gone, got %v", err)
}

func TestProductService_DeleteVariant_WrongProduct(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := newProductService(tx)
	space := dbfactory.Space(tx)
	product := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })
	otherProduct := dbfactory.Product(tx, func(p *models.Product) { p.SpaceID = space.ID })
	variant := dbfactory.ProductVariant(tx, func(v *models.ProductVariant) { v.ProductID = otherProduct.ID })

	err := svc.DeleteVariant(space.ID, product.ID, variant.ID)
	assert.ErrorIs(t, err, ErrProductVariantNotFound)

	// Variant untouched.
	require.NoError(t, tx.First(&models.ProductVariant{}, "id = ?", variant.ID).Error)
}
