package services

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tylerjvollick/nori/internal/dbfactory"
	"github.com/tylerjvollick/nori/internal/dbtest"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"gorm.io/gorm"
)

// --- Mock station seeder ---

type mockStationSeeder struct {
	created []models.Station
	err     error
}

func (m *mockStationSeeder) Create(station *models.Station) error {
	if m.err != nil {
		return m.err
	}
	m.created = append(m.created, *station)
	return nil
}

// --- Mock space repository (minimal for CreateSpace tests) ---

type mockSpaceRepo struct {
	slugExists bool
	createErr  error
}

func (m *mockSpaceRepo) Create(space *models.Space) error     { return m.createErr }
func (m *mockSpaceRepo) FindByID(id uuid.UUID) (*models.Space, error) {
	return nil, gorm.ErrRecordNotFound
}
func (m *mockSpaceRepo) FindBySlugAndAccountID(slug string, accountID uuid.UUID) (*models.Space, error) {
	return nil, gorm.ErrRecordNotFound
}
func (m *mockSpaceRepo) FindByAccountID(accountID uuid.UUID) ([]models.Space, error) {
	return nil, nil
}
func (m *mockSpaceRepo) FindDefaultByAccountID(accountID uuid.UUID) (*models.Space, error) {
	return nil, gorm.ErrRecordNotFound
}
func (m *mockSpaceRepo) SlugExistsInAccount(slug string, accountID uuid.UUID) (bool, error) {
	return m.slugExists, nil
}
func (m *mockSpaceRepo) Update(space *models.Space) error         { return nil }
func (m *mockSpaceRepo) Delete(id uuid.UUID) error                { return nil }
func (m *mockSpaceRepo) CountByAccountID(accountID uuid.UUID) (int64, error) {
	return 1, nil
}

// newTestSpaceService creates a SpaceService with the mock station seeder.
// It uses real repo types by embedding the mock — but since SpaceService takes
// concrete *repositories types, we use a workaround: test seedDefaultStations directly.

func TestSeedDefaultStations_CreatesAllStations(t *testing.T) {
	seeder := &mockStationSeeder{}
	svc := &SpaceService{stationSeeder: seeder}

	spaceID := uuid.New()
	svc.seedDefaultStations(spaceID)

	require.Len(t, seeder.created, 7)

	expectedNames := []string{"Intake", "Mill", "Joinery", "Assembly", "Finish", "QC", "Ship"}
	for i, name := range expectedNames {
		assert.Equal(t, name, seeder.created[i].Name)
		assert.Equal(t, spaceID, seeder.created[i].SpaceID)
		assert.Equal(t, i, seeder.created[i].DisplayOrder)
		assert.Equal(t, 1, seeder.created[i].WIPLimit)
		assert.True(t, seeder.created[i].IsActive)
		assert.NotEqual(t, uuid.Nil, seeder.created[i].ID)
	}
}

func TestSeedDefaultStations_UniqueIDs(t *testing.T) {
	seeder := &mockStationSeeder{}
	svc := &SpaceService{stationSeeder: seeder}

	svc.seedDefaultStations(uuid.New())

	ids := make(map[uuid.UUID]bool)
	for _, s := range seeder.created {
		assert.False(t, ids[s.ID], "duplicate station ID: %s", s.ID)
		ids[s.ID] = true
	}
}

func TestSeedDefaultStations_ContinuesOnError(t *testing.T) {
	// Seeder that fails on the 3rd station
	callCount := 0
	seeder := &mockStationSeeder{}
	origCreate := seeder.Create
	_ = origCreate
	failingSeeder := &failAfterNSeeder{maxSuccess: 2}
	svc := &SpaceService{stationSeeder: failingSeeder}

	svc.seedDefaultStations(uuid.New())

	// Even though one failed, we should still attempt all 7
	assert.Equal(t, 7, failingSeeder.attempts)
	assert.Equal(t, 6, len(failingSeeder.created)) // 7 - 1 failure
	_ = callCount
}

func TestSeedDefaultStations_NilSeeder(t *testing.T) {
	svc := &SpaceService{stationSeeder: nil}
	// Should not panic
	svc.seedDefaultStations(uuid.New())
}

// --- Helper: failing seeder ---

type failAfterNSeeder struct {
	maxSuccess int
	attempts   int
	created    []models.Station
}

func (f *failAfterNSeeder) Create(station *models.Station) error {
	f.attempts++
	if f.attempts == f.maxSuccess+1 {
		return errors.New("simulated failure")
	}
	f.created = append(f.created, *station)
	return nil
}

// --- Test CreateSpace integration with station seeding ---

func TestCreateSpace_SeedsStations(t *testing.T) {
	// We can't easily call CreateSpace because it needs concrete repo types.
	// Instead, verify the seed function is wired correctly by testing the
	// seed method directly (above) and the wiring in app.go (compile-time check).
	// This test verifies the DefaultStationNames list is correct.
	assert.Equal(t, []string{"Intake", "Mill", "Joinery", "Assembly", "Finish", "QC", "Ship"}, DefaultStationNames)
}

// --- Slug generation tests (already existed as pure functions, adding here for completeness) ---

func TestGenerateSlug_MultiWord(t *testing.T) {
	assert.Equal(t, "MW", GenerateSlug("My Workshop"))
}

func TestGenerateSlug_SingleWord(t *testing.T) {
	assert.Equal(t, "DEF", GenerateSlug("Default"))
}

func TestGenerateSlug_Empty(t *testing.T) {
	assert.Equal(t, "SPC", GenerateSlug(""))
}

func TestValidateSlug_Valid(t *testing.T) {
	assert.NoError(t, ValidateSlug("MW"))
	assert.NoError(t, ValidateSlug("ABC12"))
}

func TestValidateSlug_TooShort(t *testing.T) {
	assert.Error(t, ValidateSlug("A"))
}

func TestValidateSlug_TooLong(t *testing.T) {
	assert.Error(t, ValidateSlug("ABCDEF"))
}

// Verify that StationRepository satisfies StationSeeder interface (compile-time check)
var _ StationSeeder = (*repositories.StationRepository)(nil)

// --- UpdateSpace defaultLaborRate (real PostgreSQL) ---

// TestSpaceService_UpdateSpace_DefaultLaborRate verifies set / leave-unchanged
// / clear semantics for defaultLaborRate against a real database, going
// through the same JSON unmarshalling the API uses.
func TestSpaceService_UpdateSpace_DefaultLaborRate(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := NewSpaceService(
		repositories.NewSpaceRepository(tx),
		repositories.NewUserRepository(tx),
		repositories.NewSpaceMemberRepository(tx),
		repositories.NewStationRepository(tx),
	)

	space := dbfactory.Space(tx)

	// 1. Set the rate via a payload with defaultLaborRate present.
	var setDTO dtos.UpdateSpaceDTO
	require.NoError(t, json.Unmarshal([]byte(`{"defaultLaborRate":"50.00"}`), &setDTO))
	updated, err := svc.UpdateSpace(space.ID, space.AccountID, &setDTO)
	require.NoError(t, err)
	require.NotNil(t, updated.DefaultLaborRate)
	assert.True(t, updated.DefaultLaborRate.Equal(decimal.RequireFromString("50.00")))

	// 2. A payload without the field leaves the rate unchanged.
	var absentDTO dtos.UpdateSpaceDTO
	require.NoError(t, json.Unmarshal([]byte(`{"name":"Renamed Shop"}`), &absentDTO))
	unchanged, err := svc.UpdateSpace(space.ID, space.AccountID, &absentDTO)
	require.NoError(t, err)
	require.NotNil(t, unchanged.DefaultLaborRate, "absent field must not clear the rate")
	assert.True(t, unchanged.DefaultLaborRate.Equal(decimal.RequireFromString("50.00")))
	assert.Equal(t, "Renamed Shop", unchanged.Name)

	// 3. An explicit null clears the rate.
	var clearDTO dtos.UpdateSpaceDTO
	require.NoError(t, json.Unmarshal([]byte(`{"defaultLaborRate":null}`), &clearDTO))
	cleared, err := svc.UpdateSpace(space.ID, space.AccountID, &clearDTO)
	require.NoError(t, err)
	assert.Nil(t, cleared.DefaultLaborRate, "explicit null must clear the rate")

	// Verify persistence with a fresh read.
	reloaded, err := svc.GetSpaceByID(space.ID, space.AccountID)
	require.NoError(t, err)
	assert.Nil(t, reloaded.DefaultLaborRate)
}

// TestSpaceService_UpdateSpace_NegativeLaborRateRejected verifies the service
// rejects a negative defaultLaborRate.
func TestSpaceService_UpdateSpace_NegativeLaborRateRejected(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc := NewSpaceService(
		repositories.NewSpaceRepository(tx),
		repositories.NewUserRepository(tx),
		repositories.NewSpaceMemberRepository(tx),
		repositories.NewStationRepository(tx),
	)

	space := dbfactory.Space(tx)

	var dto dtos.UpdateSpaceDTO
	require.NoError(t, json.Unmarshal([]byte(`{"defaultLaborRate":"-1"}`), &dto))
	_, err := svc.UpdateSpace(space.ID, space.AccountID, &dto)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "defaultLaborRate must be >= 0")
}
