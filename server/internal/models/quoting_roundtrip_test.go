package models_test

import (
	"os"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tylerjvollick/nori/internal/dbfactory"
	"github.com/tylerjvollick/nori/internal/dbtest"
	"github.com/tylerjvollick/nori/internal/models"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.SetupTestMain(m, os.DirFS("../..")))
}

func ptr[T any](v T) *T { return &v }

func TestMaterialRoundTrip_CatalogFields(t *testing.T) {
	tx := dbtest.TestTx(t)
	space := dbfactory.Space(tx)

	m := models.Material{
		SpaceID:  space.ID,
		Name:     "Walnut 8/4",
		Unit:     "bf",
		UnitCost: ptr(decimal.NewFromFloat(14.50)),
		Supplier: ptr("Hardwood Supply Co"),
		SKU:      ptr("WAL-84"),
	}
	require.NoError(t, tx.Create(&m).Error)

	var got models.Material
	require.NoError(t, tx.First(&got, "id = ?", m.ID).Error)
	assert.Equal(t, "Walnut 8/4", got.Name)
	require.NotNil(t, got.Supplier)
	assert.Equal(t, "Hardwood Supply Co", *got.Supplier)
	require.NotNil(t, got.SKU)
	assert.Equal(t, "WAL-84", *got.SKU)
	require.NotNil(t, got.UnitCost)
	assert.True(t, got.UnitCost.Equal(decimal.NewFromFloat(14.50)))
}

func TestMaterialSoftDelete(t *testing.T) {
	tx := dbtest.TestTx(t)
	space := dbfactory.Space(tx)

	m := models.Material{SpaceID: space.ID, Name: "Maple", Unit: "bf"}
	require.NoError(t, tx.Create(&m).Error)
	require.NoError(t, tx.Delete(&m).Error)

	var count int64
	tx.Model(&models.Material{}).Where("id = ?", m.ID).Count(&count)
	assert.Equal(t, int64(0), count, "soft-deleted material visible in default scope")

	tx.Unscoped().Model(&models.Material{}).Where("id = ?", m.ID).Count(&count)
	assert.Equal(t, int64(1), count, "soft-deleted material missing from unscoped query")
}

func TestTaskMaterialRoundTrip_UniqueConstraint(t *testing.T) {
	tx := dbtest.TestTx(t)
	space := dbfactory.Space(tx)
	task := dbfactory.Task(tx, func(tk *models.Task) { tk.SpaceID = space.ID })

	mat := models.Material{SpaceID: space.ID, Name: "Brass screws", Unit: "ea"}
	require.NoError(t, tx.Create(&mat).Error)

	tm := models.TaskMaterial{
		TaskID:          task.ID,
		MaterialID:      mat.ID,
		QuantityPerUnit: decimal.NewFromInt(8),
		Notes:           ptr("two per corner"),
	}
	require.NoError(t, tx.Create(&tm).Error)

	var got models.TaskMaterial
	require.NoError(t, tx.Preload("Material").First(&got, "id = ?", tm.ID).Error)
	assert.True(t, got.QuantityPerUnit.Equal(decimal.NewFromInt(8)))
	require.NotNil(t, got.Material)
	assert.Equal(t, "Brass screws", got.Material.Name)

	dup := models.TaskMaterial{
		TaskID:          task.ID,
		MaterialID:      mat.ID,
		QuantityPerUnit: decimal.NewFromInt(1),
	}
	assert.Error(t, tx.Create(&dup).Error, "duplicate (task_id, material_id) must violate unique constraint")
}

func TestProductAndVariantRoundTrip(t *testing.T) {
	tx := dbtest.TestTx(t)
	space := dbfactory.Space(tx)
	recipe := dbfactory.Recipe(tx, func(r *models.Recipe) { r.SpaceID = space.ID })

	p := models.Product{
		SpaceID:     space.ID,
		Name:        "Dining Table",
		Description: ptr("Solid hardwood dining table"),
		IsActive:    true,
	}
	require.NoError(t, tx.Create(&p).Error)

	v := models.ProductVariant{
		ProductID:       p.ID,
		Name:            "Walnut / Oil finish",
		RecipeID:        &recipe.ID,
		RecipeVariables: models.JSONB{"wood": "walnut", "finish": "oil"},
		Price:           ptr(decimal.NewFromInt(2400)),
		IsActive:        true,
	}
	require.NoError(t, tx.Create(&v).Error)

	var got models.Product
	require.NoError(t, tx.Preload("Variants").First(&got, "id = ?", p.ID).Error)
	require.Len(t, got.Variants, 1)
	assert.Equal(t, "Walnut / Oil finish", got.Variants[0].Name)
	assert.Equal(t, "walnut", got.Variants[0].RecipeVariables["wood"])
	require.NotNil(t, got.Variants[0].Price)
	assert.True(t, got.Variants[0].Price.Equal(decimal.NewFromInt(2400)))
}

func TestProductSoftDelete(t *testing.T) {
	tx := dbtest.TestTx(t)
	space := dbfactory.Space(tx)

	p := models.Product{SpaceID: space.ID, Name: "Bench", IsActive: true}
	require.NoError(t, tx.Create(&p).Error)
	require.NoError(t, tx.Delete(&p).Error)

	var count int64
	tx.Model(&models.Product{}).Where("id = ?", p.ID).Count(&count)
	assert.Equal(t, int64(0), count)
	tx.Unscoped().Model(&models.Product{}).Where("id = ?", p.ID).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestQuoteRoundTrip_LinesCascadeDelete(t *testing.T) {
	tx := dbtest.TestTx(t)
	space := dbfactory.Space(tx)
	user := dbfactory.User(tx)
	recipe := dbfactory.Recipe(tx, func(r *models.Recipe) { r.SpaceID = space.ID })

	q := models.Quote{
		SpaceID:     space.ID,
		Status:      models.QuoteStatusDraft,
		Markup:      ptr(decimal.NewFromFloat(0.15)),
		CreatedByID: user.ID,
	}
	require.NoError(t, tx.Create(&q).Error)

	line := models.QuoteLine{
		QuoteID:      q.ID,
		RecipeID:     &recipe.ID,
		Quantity:     15,
		UnitPrice:    ptr(decimal.NewFromInt(950)),
		CostSnapshot: models.JSONB{"labor": 4200.0, "materials": 6300.0},
	}
	require.NoError(t, tx.Create(&line).Error)

	var got models.Quote
	require.NoError(t, tx.Preload("Lines").First(&got, "id = ?", q.ID).Error)
	assert.Equal(t, models.QuoteStatusDraft, got.Status)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, 15, got.Lines[0].Quantity)
	assert.Equal(t, 4200.0, got.Lines[0].CostSnapshot["labor"])

	// Deleting the quote must cascade to its lines (DB-level FK).
	require.NoError(t, tx.Delete(&models.Quote{}, "id = ?", q.ID).Error)
	var lineCount int64
	tx.Model(&models.QuoteLine{}).Where("quote_id = ?", q.ID).Count(&lineCount)
	assert.Equal(t, int64(0), lineCount, "quote_line rows must cascade-delete with parent quote")
}

func TestQuoteStatusConstraint(t *testing.T) {
	tx := dbtest.TestTx(t)
	space := dbfactory.Space(tx)
	user := dbfactory.User(tx)

	q := models.Quote{
		SpaceID:     space.ID,
		Status:      "bogus",
		CreatedByID: user.ID,
	}
	assert.Error(t, tx.Create(&q).Error, "invalid quote status must violate check constraint")
}

func TestStationCostsHourRoundTrip(t *testing.T) {
	tx := dbtest.TestTx(t)

	st := dbfactory.Station(tx, func(s *models.Station) {
		s.CostsHour = ptr(decimal.NewFromFloat(85.00))
	})

	var got models.Station
	require.NoError(t, tx.First(&got, "id = ?", st.ID).Error)
	require.NotNil(t, got.CostsHour)
	assert.True(t, got.CostsHour.Equal(decimal.NewFromFloat(85.00)))

	// Nullable: a station without a rate round-trips as nil.
	st2 := dbfactory.Station(tx)
	var got2 models.Station
	require.NoError(t, tx.First(&got2, "id = ?", st2.ID).Error)
	assert.Nil(t, got2.CostsHour)
}

func TestSpaceDefaultLaborRateRoundTrip(t *testing.T) {
	tx := dbtest.TestTx(t)

	space := dbfactory.Space(tx, func(s *models.Space) {
		s.DefaultLaborRate = ptr(decimal.NewFromFloat(65.00))
	})

	var got models.Space
	require.NoError(t, tx.First(&got, "id = ?", space.ID).Error)
	require.NotNil(t, got.DefaultLaborRate)
	assert.True(t, got.DefaultLaborRate.Equal(decimal.NewFromFloat(65.00)))
}
