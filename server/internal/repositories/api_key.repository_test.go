package repositories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

func TestAPIKeyRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	// Setup test data
	account := createTestAccount(t, db)
	user := createTestUser(t, db, "test@example.com", account.ID)

	// Create API key
	apiKey := &models.APIKey{
		AccountID:   account.ID,
		Name:        "Test API Key",
		KeyHash:     "hashed_key_value",
		IsActive:    true,
		CreatedByID: user.ID,
	}

	err := repo.Create(apiKey)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the key was created
	if apiKey.ID == uuid.Nil {
		t.Error("Expected ID to be set after creation")
	}

	if apiKey.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set after creation")
	}

	// Verify we can retrieve it from the database
	var retrieved models.APIKey
	err = db.First(&retrieved, "id = ?", apiKey.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve created API key: %v", err)
	}

	if retrieved.Name != "Test API Key" {
		t.Errorf("Expected Name to be 'Test API Key', got '%s'", retrieved.Name)
	}

	if retrieved.KeyHash != "hashed_key_value" {
		t.Errorf("Expected KeyHash to be 'hashed_key_value', got '%s'", retrieved.KeyHash)
	}

	if !retrieved.IsActive {
		t.Error("Expected IsActive to be true")
	}

	if retrieved.AccountID != account.ID {
		t.Errorf("Expected AccountID to be %s, got %s", account.ID, retrieved.AccountID)
	}

	if retrieved.CreatedByID != user.ID {
		t.Errorf("Expected CreatedByID to be %s, got %s", user.ID, retrieved.CreatedByID)
	}
}

func TestAPIKeyRepository_Create_UniqueKeyHash(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	// Setup test data
	account := createTestAccount(t, db)
	user := createTestUser(t, db, "test@example.com", account.ID)

	// Create first API key
	apiKey1 := &models.APIKey{
		AccountID:   account.ID,
		Name:        "Test API Key 1",
		KeyHash:     "same_hash_value",
		IsActive:    true,
		CreatedByID: user.ID,
	}
	err := repo.Create(apiKey1)
	if err != nil {
		t.Fatalf("Expected no error on first create, got: %v", err)
	}

	// Attempt to create duplicate (same key hash)
	apiKey2 := &models.APIKey{
		AccountID:   account.ID,
		Name:        "Test API Key 2",
		KeyHash:     "same_hash_value",
		IsActive:    true,
		CreatedByID: user.ID,
	}
	err = repo.Create(apiKey2)
	if err == nil {
		t.Error("Expected error when creating API key with duplicate KeyHash, got nil")
	}
}

func TestAPIKeyRepository_Create_WithExpiresAt(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	// Setup test data
	account := createTestAccount(t, db)
	user := createTestUser(t, db, "test@example.com", account.ID)

	// Create API key with expiration
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	apiKey := &models.APIKey{
		AccountID:   account.ID,
		Name:        "Expiring Key",
		KeyHash:     "hash_with_expiry",
		IsActive:    true,
		CreatedByID: user.ID,
		ExpiresAt:   &expiresAt,
	}

	err := repo.Create(apiKey)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify expiration was set
	var retrieved models.APIKey
	err = db.First(&retrieved, "id = ?", apiKey.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve created API key: %v", err)
	}

	if retrieved.ExpiresAt == nil {
		t.Fatal("Expected ExpiresAt to be set")
	}

	// Allow a small time difference due to rounding
	diff := retrieved.ExpiresAt.Sub(expiresAt)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("Expected ExpiresAt to be around %v, got %v", expiresAt, *retrieved.ExpiresAt)
	}
}

func TestAPIKeyRepository_GetByKeyHash(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	// Setup test data
	account := createTestAccount(t, db)
	user := createTestUser(t, db, "test@example.com", account.ID)

	// Create API key
	apiKey := &models.APIKey{
		AccountID:   account.ID,
		Name:        "Test API Key",
		KeyHash:     "unique_hash_123",
		IsActive:    true,
		CreatedByID: user.ID,
	}
	err := repo.Create(apiKey)
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	// Retrieve by key hash
	retrieved, err := repo.GetByKeyHash("unique_hash_123")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to retrieve API key, got nil")
	}

	if retrieved.ID != apiKey.ID {
		t.Errorf("Expected ID to be %s, got %s", apiKey.ID, retrieved.ID)
	}

	if retrieved.Name != "Test API Key" {
		t.Errorf("Expected Name to be 'Test API Key', got '%s'", retrieved.Name)
	}

	if retrieved.KeyHash != "unique_hash_123" {
		t.Errorf("Expected KeyHash to be 'unique_hash_123', got '%s'", retrieved.KeyHash)
	}
}

func TestAPIKeyRepository_GetByKeyHash_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	// Try to get a non-existent key
	retrieved, err := repo.GetByKeyHash("nonexistent_hash")
	if err != gorm.ErrRecordNotFound {
		t.Errorf("Expected ErrRecordNotFound, got: %v", err)
	}

	if retrieved != nil {
		t.Error("Expected nil result when key not found")
	}
}

func TestAPIKeyRepository_GetByAccount(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	// Setup test data
	account := createTestAccount(t, db)
	user := createTestUser(t, db, "test@example.com", account.ID)

	// Create multiple API keys for the account
	apiKey1 := &models.APIKey{
		AccountID:   account.ID,
		Name:        "API Key 1",
		KeyHash:     "hash_1",
		IsActive:    true,
		CreatedByID: user.ID,
	}
	err := repo.Create(apiKey1)
	if err != nil {
		t.Fatalf("Failed to create API key 1: %v", err)
	}

	// Wait to ensure different created times
	time.Sleep(10 * time.Millisecond)

	apiKey2 := &models.APIKey{
		AccountID:   account.ID,
		Name:        "API Key 2",
		KeyHash:     "hash_2",
		IsActive:    false,
		CreatedByID: user.ID,
	}
	err = repo.Create(apiKey2)
	if err != nil {
		t.Fatalf("Failed to create API key 2: %v", err)
	}

	// Retrieve all API keys for the account
	apiKeys, err := repo.GetByAccount(account.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(apiKeys) != 2 {
		t.Fatalf("Expected 2 API keys, got %d", len(apiKeys))
	}

	// Verify they are ordered by created_at DESC (newest first)
	if apiKeys[0].Name != "API Key 2" {
		t.Errorf("Expected first key to be 'API Key 2', got '%s'", apiKeys[0].Name)
	}

	if apiKeys[1].Name != "API Key 1" {
		t.Errorf("Expected second key to be 'API Key 1', got '%s'", apiKeys[1].Name)
	}

	// Verify the keys belong to the correct account
	for _, key := range apiKeys {
		if key.AccountID != account.ID {
			t.Errorf("Expected AccountID to be %s, got %s", account.ID, key.AccountID)
		}
	}
}

func TestAPIKeyRepository_GetByAccount_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	// Create an account with no API keys
	account := createTestAccount(t, db)

	// Retrieve API keys for the account
	apiKeys, err := repo.GetByAccount(account.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(apiKeys) != 0 {
		t.Errorf("Expected 0 API keys, got %d", len(apiKeys))
	}
}

func TestAPIKeyRepository_Deactivate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	// Setup test data
	account := createTestAccount(t, db)
	user := createTestUser(t, db, "test@example.com", account.ID)

	// Create an active API key
	apiKey := &models.APIKey{
		AccountID:   account.ID,
		Name:        "Active Key",
		KeyHash:     "hash_to_deactivate",
		IsActive:    true,
		CreatedByID: user.ID,
	}
	err := repo.Create(apiKey)
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	// Deactivate the key
	err = repo.Deactivate(apiKey.ID)
	if err != nil {
		t.Fatalf("Expected no error on deactivate, got: %v", err)
	}

	// Verify it's deactivated
	var retrieved models.APIKey
	err = db.First(&retrieved, "id = ?", apiKey.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve API key: %v", err)
	}

	if retrieved.IsActive {
		t.Error("Expected IsActive to be false after deactivation")
	}
}

func TestAPIKeyRepository_Deactivate_NonExistent(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	// Try to deactivate a non-existent key
	err := repo.Deactivate(uuid.New())
	if err != nil {
		t.Errorf("Expected no error when deactivating non-existent key, got: %v", err)
	}
}

func TestAPIKeyRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	// Setup test data
	account := createTestAccount(t, db)
	user := createTestUser(t, db, "test@example.com", account.ID)

	// Create API key
	apiKey := &models.APIKey{
		AccountID:   account.ID,
		Name:        "Key to Delete",
		KeyHash:     "hash_to_delete",
		IsActive:    true,
		CreatedByID: user.ID,
	}
	err := repo.Create(apiKey)
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	// Delete the key
	err = repo.Delete(apiKey.ID)
	if err != nil {
		t.Fatalf("Expected no error on delete, got: %v", err)
	}

	// Verify it's deleted
	var retrieved models.APIKey
	err = db.First(&retrieved, "id = ?", apiKey.ID).Error
	if err != gorm.ErrRecordNotFound {
		t.Error("Expected record to be deleted")
	}
}

func TestAPIKeyRepository_Delete_NonExistent(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	// Try to delete a non-existent key
	err := repo.Delete(uuid.New())
	if err != nil {
		t.Errorf("Expected no error when deleting non-existent key, got: %v", err)
	}
}

func TestAPIKeyRepository_UpdateLastUsed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	// Setup test data
	account := createTestAccount(t, db)
	user := createTestUser(t, db, "test@example.com", account.ID)

	// Create API key with no last used time
	apiKey := &models.APIKey{
		AccountID:   account.ID,
		Name:        "Key to Use",
		KeyHash:     "hash_to_use",
		IsActive:    true,
		CreatedByID: user.ID,
	}
	err := repo.Create(apiKey)
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	// Verify LastUsedAt is initially nil
	var retrieved models.APIKey
	err = db.First(&retrieved, "id = ?", apiKey.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve API key: %v", err)
	}
	if retrieved.LastUsedAt != nil {
		t.Error("Expected LastUsedAt to be nil initially")
	}

	// Update last used timestamp
	beforeUpdate := time.Now()
	err = repo.UpdateLastUsed(apiKey.ID)
	if err != nil {
		t.Fatalf("Expected no error on UpdateLastUsed, got: %v", err)
	}
	afterUpdate := time.Now()

	// Verify LastUsedAt was set
	err = db.First(&retrieved, "id = ?", apiKey.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve API key: %v", err)
	}

	if retrieved.LastUsedAt == nil {
		t.Fatal("Expected LastUsedAt to be set after update")
	}

	// Verify the timestamp is recent (within the test execution window)
	if retrieved.LastUsedAt.Before(beforeUpdate) || retrieved.LastUsedAt.After(afterUpdate) {
		t.Errorf("Expected LastUsedAt to be between %v and %v, got %v", beforeUpdate, afterUpdate, *retrieved.LastUsedAt)
	}
}

func TestAPIKeyRepository_UpdateLastUsed_Multiple(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	// Setup test data
	account := createTestAccount(t, db)
	user := createTestUser(t, db, "test@example.com", account.ID)

	// Create API key
	apiKey := &models.APIKey{
		AccountID:   account.ID,
		Name:        "Key to Use",
		KeyHash:     "hash_to_use_multiple",
		IsActive:    true,
		CreatedByID: user.ID,
	}
	err := repo.Create(apiKey)
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	// First update
	err = repo.UpdateLastUsed(apiKey.ID)
	if err != nil {
		t.Fatalf("Expected no error on first UpdateLastUsed, got: %v", err)
	}

	var retrieved models.APIKey
	err = db.First(&retrieved, "id = ?", apiKey.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve API key: %v", err)
	}

	if retrieved.LastUsedAt == nil {
		t.Fatal("Expected LastUsedAt to be set after first update")
	}

	firstUpdateTime := *retrieved.LastUsedAt

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Second update
	err = repo.UpdateLastUsed(apiKey.ID)
	if err != nil {
		t.Fatalf("Expected no error on second UpdateLastUsed, got: %v", err)
	}

	err = db.First(&retrieved, "id = ?", apiKey.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve API key: %v", err)
	}

	if retrieved.LastUsedAt == nil {
		t.Fatal("Expected LastUsedAt to be set after second update")
	}

	// Verify the timestamp was updated
	if !retrieved.LastUsedAt.After(firstUpdateTime) {
		t.Errorf("Expected LastUsedAt to be updated from %v to %v", firstUpdateTime, *retrieved.LastUsedAt)
	}
}

func TestAPIKeyRepository_UpdateLastUsed_NonExistent(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	// Try to update a non-existent key
	err := repo.UpdateLastUsed(uuid.New())
	if err != nil {
		t.Errorf("Expected no error when updating non-existent key, got: %v", err)
	}
}
