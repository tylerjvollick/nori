package repositories

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupUserTestDB creates a test database connection using the same PostgreSQL instance
func setupUserTestDB(t *testing.T) *gorm.DB {
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
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM accounts")

	return db
}

// createUserTestAccount creates a test account
func createUserTestAccount(t *testing.T, db *gorm.DB) *models.Account {
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

func TestUserRepository_GetByEmail(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)

	// Setup test data
	account := createUserTestAccount(t, db)
	email := "test@example.com"
	user := &models.User{
		Email:            email,
		DefaultAccountID: &account.ID,
	}

	err := repo.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Retrieve by email
	retrieved, err := repo.GetUserByEmail(email)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to retrieve user, got nil")
	}

	if retrieved.Email != email {
		t.Errorf("Expected email to be '%s', got '%s'", email, retrieved.Email)
	}

	if retrieved.ID != user.ID {
		t.Errorf("Expected ID to be %s, got %s", user.ID, retrieved.ID)
	}
}

func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)

	// Try to get a non-existent user
	retrieved, err := repo.GetUserByEmail("nonexistent@example.com")
	if err != gorm.ErrRecordNotFound {
		t.Errorf("Expected ErrRecordNotFound, got: %v", err)
	}

	if retrieved != nil {
		t.Error("Expected nil result when user not found")
	}
}

func TestUserRepository_UpdatePassword(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)

	// Setup test data
	account := createUserTestAccount(t, db)
	oldPassword := "old_hashed_password"
	user := &models.User{
		Email:            "test@example.com",
		Password:         &oldPassword,
		DefaultAccountID: &account.ID,
	}

	err := repo.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Verify initial password
	var retrieved models.User
	err = db.First(&retrieved, "id = ?", user.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
	}

	if retrieved.Password == nil || *retrieved.Password != "old_hashed_password" {
		t.Errorf("Expected password to be 'old_hashed_password', got '%v'", retrieved.Password)
	}

	// Update password
	newHashedPassword := "new_hashed_password"
	err = repo.UpdatePassword(user.ID, newHashedPassword)
	if err != nil {
		t.Fatalf("Expected no error on UpdatePassword, got: %v", err)
	}

	// Verify password was updated
	err = db.First(&retrieved, "id = ?", user.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve user after update: %v", err)
	}

	if retrieved.Password == nil {
		t.Fatal("Expected password to be set")
	}

	if *retrieved.Password != "new_hashed_password" {
		t.Errorf("Expected password to be 'new_hashed_password', got '%s'", *retrieved.Password)
	}
}

func TestUserRepository_UpdatePassword_NonExistentUser(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)

	// Try to update password for a non-existent user
	err := repo.UpdatePassword(uuid.New(), "some_password")
	if err != nil {
		t.Errorf("Expected no error when updating non-existent user, got: %v", err)
	}
}

func TestUserRepository_ClearMustChangePassword(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)

	// Setup test data - new users have MustChangePassword = true by default
	account := createUserTestAccount(t, db)
	user := &models.User{
		Email:              "test@example.com",
		MustChangePassword: true,
		DefaultAccountID:   &account.ID,
	}

	err := repo.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Verify initial MustChangePassword is true
	var retrieved models.User
	err = db.First(&retrieved, "id = ?", user.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
	}

	if !retrieved.MustChangePassword {
		t.Error("Expected MustChangePassword to be true initially")
	}

	// Clear MustChangePassword
	err = repo.ClearMustChangePassword(user.ID)
	if err != nil {
		t.Fatalf("Expected no error on ClearMustChangePassword, got: %v", err)
	}

	// Verify MustChangePassword was cleared
	err = db.First(&retrieved, "id = ?", user.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve user after clearing flag: %v", err)
	}

	if retrieved.MustChangePassword {
		t.Error("Expected MustChangePassword to be false after clearing")
	}
}

func TestUserRepository_ClearMustChangePassword_AlreadyFalse(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)

	// Setup test data - user with MustChangePassword already false
	account := createUserTestAccount(t, db)
	user := &models.User{
		Email:              "test@example.com",
		MustChangePassword: false,
		DefaultAccountID:   &account.ID,
	}

	err := repo.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Clear MustChangePassword (idempotent operation)
	err = repo.ClearMustChangePassword(user.ID)
	if err != nil {
		t.Fatalf("Expected no error on ClearMustChangePassword, got: %v", err)
	}

	// Verify MustChangePassword is still false
	var retrieved models.User
	err = db.First(&retrieved, "id = ?", user.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
	}

	if retrieved.MustChangePassword {
		t.Error("Expected MustChangePassword to be false")
	}
}

func TestUserRepository_ClearMustChangePassword_NonExistentUser(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)

	// Try to clear flag for a non-existent user
	err := repo.ClearMustChangePassword(uuid.New())
	if err != nil {
		t.Errorf("Expected no error when clearing flag for non-existent user, got: %v", err)
	}
}

func TestUserRepository_UpdatePassword_WithClearMustChangePassword(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)

	// Setup test data - simulate a new user who must change password
	account := createUserTestAccount(t, db)
	tempPassword := "temp_password_hash"
	user := &models.User{
		Email:              "newuser@example.com",
		Password:           &tempPassword,
		MustChangePassword: true,
		DefaultAccountID:   &account.ID,
	}

	err := repo.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Simulate the password change flow: update password + clear flag
	newPassword := "new_password_hash"
	err = repo.UpdatePassword(user.ID, newPassword)
	if err != nil {
		t.Fatalf("Failed to update password: %v", err)
	}

	err = repo.ClearMustChangePassword(user.ID)
	if err != nil {
		t.Fatalf("Failed to clear MustChangePassword: %v", err)
	}

	// Verify both changes were applied
	var retrieved models.User
	err = db.First(&retrieved, "id = ?", user.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
	}

	if retrieved.Password == nil || *retrieved.Password != "new_password_hash" {
		t.Errorf("Expected password to be 'new_password_hash', got '%v'", retrieved.Password)
	}

	if retrieved.MustChangePassword {
		t.Error("Expected MustChangePassword to be false after password change")
	}
}
