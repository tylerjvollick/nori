package repositories

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupTestDB creates a test database connection using the same PostgreSQL instance
func setupTestDB(t *testing.T) *gorm.DB {
	// Check if we're in a testing environment with database available
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "password"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "nori"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}

	dsn := "host=" + host + " user=" + user + " password=" + password + " dbname=" + dbname + " port=" + port + " sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}

	// Clean up test data at the start
	db.Exec("DELETE FROM space_member")
	db.Exec("DELETE FROM spaces")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM accounts")

	return db
}

// createTestAccount creates a test account
func createTestAccount(t *testing.T, db *gorm.DB) *models.Account {
	name := "Test Account"
	// Create a user first to satisfy the CreatedByUserID requirement
	tempUser := &models.User{
		ID:    uuid.New(),
		Email: "system@example.com",
	}
	if err := db.Create(tempUser).Error; err != nil {
		t.Fatalf("Failed to create temp user: %v", err)
	}

	account := &models.Account{
		ID:              uuid.New(),
		Name:            &name,
		Plan:            models.Trial,
		CreatedByUserID: tempUser.ID,
	}
	if err := db.Create(account).Error; err != nil {
		t.Fatalf("Failed to create test account: %v", err)
	}
	return account
}

// createTestUser creates a test user
func createTestUser(t *testing.T, db *gorm.DB, email string, accountID uuid.UUID) *models.User {
	user := &models.User{
		ID:               uuid.New(),
		Email:            email,
		DefaultAccountID: &accountID,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	return user
}

// createTestSpace creates a test space
func createTestSpace(t *testing.T, db *gorm.DB, name string, accountID uuid.UUID) *models.Space {
	space := &models.Space{
		ID:        uuid.New(),
		Name:      name,
		AccountID: accountID,
	}
	if err := db.Create(space).Error; err != nil {
		t.Fatalf("Failed to create test space: %v", err)
	}
	return space
}

func TestSpaceMemberRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSpaceMemberRepository(db)

	// Setup test data
	account := createTestAccount(t, db)
	user := createTestUser(t, db, "test@example.com", account.ID)
	space := createTestSpace(t, db, "Test Space", account.ID)

	// Create space member
	spaceMember := &models.SpaceMember{
		UserID:  user.ID,
		SpaceID: space.ID,
	}

	err := repo.Create(spaceMember)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the member was created
	if spaceMember.ID == uuid.Nil {
		t.Error("Expected ID to be set after creation")
	}

	if spaceMember.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set after creation")
	}

	// Verify we can retrieve it from the database
	var retrieved models.SpaceMember
	err = db.First(&retrieved, "id = ?", spaceMember.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve created space member: %v", err)
	}

	if retrieved.UserID != user.ID {
		t.Errorf("Expected UserID to be %s, got %s", user.ID, retrieved.UserID)
	}

	if retrieved.SpaceID != space.ID {
		t.Errorf("Expected SpaceID to be %s, got %s", space.ID, retrieved.SpaceID)
	}
}

func TestSpaceMemberRepository_Create_UniqueConstraint(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSpaceMemberRepository(db)

	// Setup test data
	account := createTestAccount(t, db)
	user := createTestUser(t, db, "test@example.com", account.ID)
	space := createTestSpace(t, db, "Test Space", account.ID)

	// Create first space member
	spaceMember1 := &models.SpaceMember{
		UserID:  user.ID,
		SpaceID: space.ID,
	}
	err := repo.Create(spaceMember1)
	if err != nil {
		t.Fatalf("Expected no error on first create, got: %v", err)
	}

	// Attempt to create duplicate (same user + space)
	spaceMember2 := &models.SpaceMember{
		UserID:  user.ID,
		SpaceID: space.ID,
	}
	err = repo.Create(spaceMember2)
	if err == nil {
		t.Error("Expected error when creating duplicate space member, got nil")
	}
}

func TestSpaceMemberRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSpaceMemberRepository(db)

	// Setup test data
	account := createTestAccount(t, db)
	user := createTestUser(t, db, "test@example.com", account.ID)
	space := createTestSpace(t, db, "Test Space", account.ID)

	// Create space member
	spaceMember := &models.SpaceMember{
		UserID:  user.ID,
		SpaceID: space.ID,
	}
	err := repo.Create(spaceMember)
	if err != nil {
		t.Fatalf("Failed to create space member: %v", err)
	}

	// Delete the member
	err = repo.Delete(user.ID, space.ID)
	if err != nil {
		t.Fatalf("Expected no error on delete, got: %v", err)
	}

	// Verify it's deleted
	var retrieved models.SpaceMember
	err = db.First(&retrieved, "user_id = ? AND space_id = ?", user.ID, space.ID).Error
	if err != gorm.ErrRecordNotFound {
		t.Error("Expected record to be deleted")
	}
}

func TestSpaceMemberRepository_Delete_NonExistent(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSpaceMemberRepository(db)

	// Try to delete a non-existent member
	err := repo.Delete(uuid.New(), uuid.New())
	if err != nil {
		t.Errorf("Expected no error when deleting non-existent member, got: %v", err)
	}
}

func TestSpaceMemberRepository_GetByUserAndSpace(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSpaceMemberRepository(db)

	// Setup test data
	account := createTestAccount(t, db)
	user := createTestUser(t, db, "test@example.com", account.ID)
	space := createTestSpace(t, db, "Test Space", account.ID)

	// Create space member
	spaceMember := &models.SpaceMember{
		UserID:  user.ID,
		SpaceID: space.ID,
	}
	err := repo.Create(spaceMember)
	if err != nil {
		t.Fatalf("Failed to create space member: %v", err)
	}

	// Retrieve by user and space
	retrieved, err := repo.GetByUserAndSpace(user.ID, space.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to retrieve space member, got nil")
	}

	if retrieved.UserID != user.ID {
		t.Errorf("Expected UserID to be %s, got %s", user.ID, retrieved.UserID)
	}

	if retrieved.SpaceID != space.ID {
		t.Errorf("Expected SpaceID to be %s, got %s", space.ID, retrieved.SpaceID)
	}
}

func TestSpaceMemberRepository_GetByUserAndSpace_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSpaceMemberRepository(db)

	// Try to get a non-existent member
	retrieved, err := repo.GetByUserAndSpace(uuid.New(), uuid.New())
	if err != gorm.ErrRecordNotFound {
		t.Errorf("Expected ErrRecordNotFound, got: %v", err)
	}

	if retrieved != nil {
		t.Error("Expected nil result when member not found")
	}
}

func TestSpaceMemberRepository_GetByUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSpaceMemberRepository(db)

	// Setup test data
	account := createTestAccount(t, db)
	user := createTestUser(t, db, "test@example.com", account.ID)
	space1 := createTestSpace(t, db, "Space 1", account.ID)
	space2 := createTestSpace(t, db, "Space 2", account.ID)

	// Create multiple space memberships for the user
	member1 := &models.SpaceMember{
		UserID:  user.ID,
		SpaceID: space1.ID,
	}
	err := repo.Create(member1)
	if err != nil {
		t.Fatalf("Failed to create space member 1: %v", err)
	}

	// Wait a bit to ensure different created times
	time.Sleep(10 * time.Millisecond)

	member2 := &models.SpaceMember{
		UserID:  user.ID,
		SpaceID: space2.ID,
	}
	err = repo.Create(member2)
	if err != nil {
		t.Fatalf("Failed to create space member 2: %v", err)
	}

	// Retrieve all spaces for the user
	spaces, err := repo.GetByUser(user.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(spaces) != 2 {
		t.Fatalf("Expected 2 spaces, got %d", len(spaces))
	}

	// Verify the spaces are loaded
	foundSpace1 := false
	foundSpace2 := false
	for _, space := range spaces {
		if space.ID == space1.ID {
			foundSpace1 = true
			if space.Name != "Space 1" {
				t.Errorf("Expected space name 'Space 1', got '%s'", space.Name)
			}
		}
		if space.ID == space2.ID {
			foundSpace2 = true
			if space.Name != "Space 2" {
				t.Errorf("Expected space name 'Space 2', got '%s'", space.Name)
			}
		}
	}

	if !foundSpace1 {
		t.Error("Expected to find Space 1")
	}
	if !foundSpace2 {
		t.Error("Expected to find Space 2")
	}
}

func TestSpaceMemberRepository_GetByUser_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSpaceMemberRepository(db)

	// Create a user with no space memberships
	account := createTestAccount(t, db)
	user := createTestUser(t, db, "test@example.com", account.ID)

	// Retrieve spaces for the user
	spaces, err := repo.GetByUser(user.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(spaces) != 0 {
		t.Errorf("Expected 0 spaces, got %d", len(spaces))
	}
}

func TestSpaceMemberRepository_GetBySpace(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSpaceMemberRepository(db)

	// Setup test data
	account := createTestAccount(t, db)
	user1 := createTestUser(t, db, "user1@example.com", account.ID)
	user2 := createTestUser(t, db, "user2@example.com", account.ID)
	space := createTestSpace(t, db, "Test Space", account.ID)

	// Create multiple members for the space
	member1 := &models.SpaceMember{
		UserID:  user1.ID,
		SpaceID: space.ID,
	}
	err := repo.Create(member1)
	if err != nil {
		t.Fatalf("Failed to create space member 1: %v", err)
	}

	member2 := &models.SpaceMember{
		UserID:  user2.ID,
		SpaceID: space.ID,
	}
	err = repo.Create(member2)
	if err != nil {
		t.Fatalf("Failed to create space member 2: %v", err)
	}

	// Retrieve all members for the space
	members, err := repo.GetBySpace(space.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(members) != 2 {
		t.Fatalf("Expected 2 members, got %d", len(members))
	}

	// Verify the members are loaded
	foundUser1 := false
	foundUser2 := false
	for _, member := range members {
		if member.UserID == user1.ID {
			foundUser1 = true
			if member.User.Email != "user1@example.com" {
				t.Errorf("Expected user email 'user1@example.com', got '%s'", member.User.Email)
			}
		}
		if member.UserID == user2.ID {
			foundUser2 = true
			if member.User.Email != "user2@example.com" {
				t.Errorf("Expected user email 'user2@example.com', got '%s'", member.User.Email)
			}
		}
	}

	if !foundUser1 {
		t.Error("Expected to find user1")
	}
	if !foundUser2 {
		t.Error("Expected to find user2")
	}
}

func TestSpaceMemberRepository_GetBySpace_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSpaceMemberRepository(db)

	// Create a space with no members
	account := createTestAccount(t, db)
	space := createTestSpace(t, db, "Empty Space", account.ID)

	// Retrieve members for the space
	members, err := repo.GetBySpace(space.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(members) != 0 {
		t.Errorf("Expected 0 members, got %d", len(members))
	}
}

func TestSpaceMemberRepository_GetBySpace_WithPreload(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSpaceMemberRepository(db)

	// Setup test data
	account := createTestAccount(t, db)
	firstName := "John"
	lastName := "Doe"
	user := &models.User{
		ID:               uuid.New(),
		Email:            "john.doe@example.com",
		FirstName:        &firstName,
		LastName:         &lastName,
		DefaultAccountID: &account.ID,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	space := createTestSpace(t, db, "Test Space", account.ID)

	// Create space member
	member := &models.SpaceMember{
		UserID:  user.ID,
		SpaceID: space.ID,
	}
	err := repo.Create(member)
	if err != nil {
		t.Fatalf("Failed to create space member: %v", err)
	}

	// Retrieve members with preloaded User data
	members, err := repo.GetBySpace(space.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(members) != 1 {
		t.Fatalf("Expected 1 member, got %d", len(members))
	}

	// Verify User data is preloaded
	if members[0].User.Email != "john.doe@example.com" {
		t.Errorf("Expected user email 'john.doe@example.com', got '%s'", members[0].User.Email)
	}

	if members[0].User.FirstName == nil || *members[0].User.FirstName != "John" {
		t.Error("Expected FirstName to be 'John'")
	}

	if members[0].User.LastName == nil || *members[0].User.LastName != "Doe" {
		t.Error("Expected LastName to be 'Doe'")
	}
}
