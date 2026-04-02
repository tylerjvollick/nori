package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSpaceMemberModelFields(t *testing.T) {
	// Test that SpaceMember struct has all required fields
	userID := uuid.New()
	spaceID := uuid.New()
	createdAt := time.Now()

	spaceMember := SpaceMember{
		ID:        uuid.New(),
		UserID:    userID,
		SpaceID:   spaceID,
		CreatedAt: createdAt,
	}

	if spaceMember.ID == uuid.Nil {
		t.Error("Expected ID to be set")
	}

	if spaceMember.UserID != userID {
		t.Errorf("Expected UserID to be '%s', got '%s'", userID, spaceMember.UserID)
	}

	if spaceMember.SpaceID != spaceID {
		t.Errorf("Expected SpaceID to be '%s', got '%s'", spaceID, spaceMember.SpaceID)
	}

	if spaceMember.CreatedAt != createdAt {
		t.Errorf("Expected CreatedAt to be '%v', got '%v'", createdAt, spaceMember.CreatedAt)
	}
}

func TestSpaceMemberWithMinimalFields(t *testing.T) {
	// Test creating a SpaceMember with only required fields
	userID := uuid.New()
	spaceID := uuid.New()

	spaceMember := SpaceMember{
		UserID:  userID,
		SpaceID: spaceID,
	}

	// ID should be Nil when not explicitly set (will be set by BeforeCreate hook or DB default)
	if spaceMember.ID != uuid.Nil {
		t.Error("Expected ID to be Nil before creation")
	}

	// UserID and SpaceID should be set
	if spaceMember.UserID != userID {
		t.Error("UserID mismatch")
	}
	if spaceMember.SpaceID != spaceID {
		t.Error("SpaceID mismatch")
	}
}

func TestSpaceMemberWithRelations(t *testing.T) {
	// Test that foreign key relations are properly defined
	userID := uuid.New()
	spaceID := uuid.New()
	accountID := uuid.New()

	user := User{
		ID:    userID,
		Email: "test@example.com",
	}

	space := Space{
		ID:        spaceID,
		Name:      "Test Space",
		AccountID: accountID,
	}

	spaceMember := SpaceMember{
		ID:      uuid.New(),
		UserID:  userID,
		SpaceID: spaceID,
		User:    user,
		Space:   space,
	}

	// Verify the foreign key IDs match
	if spaceMember.UserID != spaceMember.User.ID {
		t.Error("UserID should match User.ID")
	}

	if spaceMember.SpaceID != spaceMember.Space.ID {
		t.Error("SpaceID should match Space.ID")
	}

	// Verify relation data is accessible
	if spaceMember.User.Email != "test@example.com" {
		t.Error("User relation data should be accessible")
	}

	if spaceMember.Space.Name != "Test Space" {
		t.Error("Space relation data should be accessible")
	}
}

func TestSpaceMemberCreatedAtDefaultBehavior(t *testing.T) {
	// Test that CreatedAt has zero value when not set
	// (in practice, the database will set this to now() via DEFAULT)
	spaceMember := SpaceMember{
		UserID:  uuid.New(),
		SpaceID: uuid.New(),
	}

	// In Go, time.Time defaults to zero time
	if !spaceMember.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be zero time before database insert")
	}
}

func TestSpaceMemberBeforeCreateHook(t *testing.T) {
	// Test that the BeforeCreate hook sets the ID if it's nil
	spaceMember := SpaceMember{
		UserID:  uuid.New(),
		SpaceID: uuid.New(),
	}

	// ID should be Nil initially
	if spaceMember.ID != uuid.Nil {
		t.Error("Expected ID to be Nil before BeforeCreate hook")
	}

	// Simulate calling BeforeCreate hook
	err := spaceMember.BeforeCreate(nil)
	if err != nil {
		t.Fatalf("BeforeCreate hook returned error: %v", err)
	}

	// After BeforeCreate, ID should be set
	if spaceMember.ID == uuid.Nil {
		t.Error("Expected ID to be set after BeforeCreate hook")
	}
}

func TestSpaceMemberBeforeCreateHookPreservesExistingID(t *testing.T) {
	// Test that BeforeCreate hook doesn't overwrite an existing ID
	existingID := uuid.New()
	spaceMember := SpaceMember{
		ID:      existingID,
		UserID:  uuid.New(),
		SpaceID: uuid.New(),
	}

	// Call BeforeCreate hook
	err := spaceMember.BeforeCreate(nil)
	if err != nil {
		t.Fatalf("BeforeCreate hook returned error: %v", err)
	}

	// ID should remain the same
	if spaceMember.ID != existingID {
		t.Errorf("Expected ID to remain '%s', but got '%s'", existingID, spaceMember.ID)
	}
}

func TestSpaceMemberWithAllFields(t *testing.T) {
	// Test creating a SpaceMember with all fields populated
	id := uuid.New()
	userID := uuid.New()
	spaceID := uuid.New()
	accountID := uuid.New()
	createdAt := time.Now()

	user := User{
		ID:    userID,
		Email: "member@example.com",
	}

	space := Space{
		ID:        spaceID,
		Name:      "Production Floor",
		AccountID: accountID,
	}

	spaceMember := SpaceMember{
		ID:        id,
		UserID:    userID,
		SpaceID:   spaceID,
		CreatedAt: createdAt,
		User:      user,
		Space:     space,
	}

	// Verify all fields are set correctly
	if spaceMember.ID != id {
		t.Error("ID mismatch")
	}
	if spaceMember.UserID != userID {
		t.Error("UserID mismatch")
	}
	if spaceMember.SpaceID != spaceID {
		t.Error("SpaceID mismatch")
	}
	if spaceMember.CreatedAt != createdAt {
		t.Error("CreatedAt mismatch")
	}
	if spaceMember.User.ID != userID {
		t.Error("User.ID mismatch")
	}
	if spaceMember.User.Email != "member@example.com" {
		t.Error("User.Email mismatch")
	}
	if spaceMember.Space.ID != spaceID {
		t.Error("Space.ID mismatch")
	}
	if spaceMember.Space.Name != "Production Floor" {
		t.Error("Space.Name mismatch")
	}
}

func TestSpaceMemberUniqueConstraintScenario(t *testing.T) {
	// Test that demonstrates the unique constraint expectation
	// (actual constraint is enforced by the database, not Go code)
	userID := uuid.New()
	spaceID := uuid.New()

	member1 := SpaceMember{
		ID:      uuid.New(),
		UserID:  userID,
		SpaceID: spaceID,
	}

	member2 := SpaceMember{
		ID:      uuid.New(),
		UserID:  userID,
		SpaceID: spaceID,
	}

	// Both have same UserID and SpaceID
	if member1.UserID != member2.UserID {
		t.Error("Test setup error: UserIDs should match")
	}
	if member1.SpaceID != member2.SpaceID {
		t.Error("Test setup error: SpaceIDs should match")
	}

	// But different IDs
	if member1.ID == member2.ID {
		t.Error("IDs should be different")
	}

	// Note: In a real database, inserting both would violate the unique constraint
	// This test just validates that the Go structs can be created
	// The actual constraint enforcement is tested in integration tests
}
