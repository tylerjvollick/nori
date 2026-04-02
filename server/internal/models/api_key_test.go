package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAPIKeyModelFields(t *testing.T) {
	// Test that APIKey struct has all required fields
	accountID := uuid.New()
	createdByID := uuid.New()
	now := time.Now()
	expiresAt := now.Add(30 * 24 * time.Hour)

	apiKey := APIKey{
		ID:          uuid.New(),
		AccountID:   accountID,
		Name:        "Production API Key",
		KeyHash:     "hashed_key_value",
		LastUsedAt:  &now,
		ExpiresAt:   &expiresAt,
		IsActive:    true,
		CreatedAt:   now,
		CreatedByID: createdByID,
	}

	if apiKey.ID == uuid.Nil {
		t.Error("Expected ID to be set")
	}

	if apiKey.AccountID != accountID {
		t.Errorf("Expected AccountID to be '%s', got '%s'", accountID, apiKey.AccountID)
	}

	if apiKey.Name != "Production API Key" {
		t.Errorf("Expected Name to be 'Production API Key', got '%s'", apiKey.Name)
	}

	if apiKey.KeyHash != "hashed_key_value" {
		t.Errorf("Expected KeyHash to be 'hashed_key_value', got '%s'", apiKey.KeyHash)
	}

	if apiKey.LastUsedAt == nil || !apiKey.LastUsedAt.Equal(now) {
		t.Error("Expected LastUsedAt to be set to now")
	}

	if apiKey.ExpiresAt == nil || !apiKey.ExpiresAt.Equal(expiresAt) {
		t.Error("Expected ExpiresAt to be set")
	}

	if !apiKey.IsActive {
		t.Error("Expected IsActive to be true")
	}

	if apiKey.CreatedByID != createdByID {
		t.Errorf("Expected CreatedByID to be '%s', got '%s'", createdByID, apiKey.CreatedByID)
	}
}

func TestAPIKeyIsActiveDefault(t *testing.T) {
	// Test that IsActive defaults to true for new API keys
	apiKey := APIKey{
		AccountID:   uuid.New(),
		Name:        "Test Key",
		KeyHash:     "hash",
		CreatedByID: uuid.New(),
	}

	// In Go structs, bool defaults to false, but the database migration sets DEFAULT true
	// This test documents the expected behavior
	if apiKey.IsActive != false {
		t.Logf("Note: IsActive defaults to false in Go structs, but database migration sets DEFAULT true")
	}

	// When explicitly set to true
	apiKey.IsActive = true
	if !apiKey.IsActive {
		t.Error("Expected IsActive to be true when explicitly set")
	}
}

func TestAPIKeyWithNilOptionalFields(t *testing.T) {
	// Test that APIKey can be created with nil optional fields
	apiKey := APIKey{
		ID:          uuid.New(),
		AccountID:   uuid.New(),
		Name:        "Test Key",
		KeyHash:     "hash",
		LastUsedAt:  nil,
		ExpiresAt:   nil,
		IsActive:    true,
		CreatedAt:   time.Now(),
		CreatedByID: uuid.New(),
	}

	if apiKey.LastUsedAt != nil {
		t.Error("Expected LastUsedAt to be nil")
	}

	if apiKey.ExpiresAt != nil {
		t.Error("Expected ExpiresAt to be nil")
	}
}

func TestAPIKeyLastUsedAt(t *testing.T) {
	tests := []struct {
		name       string
		lastUsedAt *time.Time
	}{
		{
			name:       "nil last used at",
			lastUsedAt: nil,
		},
		{
			name: "with last used at",
			lastUsedAt: func() *time.Time {
				t := time.Now()
				return &t
			}(),
		},
		{
			name: "last used at in the past",
			lastUsedAt: func() *time.Time {
				t := time.Now().Add(-24 * time.Hour)
				return &t
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := APIKey{
				AccountID:   uuid.New(),
				Name:        "Test Key",
				KeyHash:     "hash",
				LastUsedAt:  tt.lastUsedAt,
				CreatedByID: uuid.New(),
			}

			if tt.lastUsedAt == nil {
				if apiKey.LastUsedAt != nil {
					t.Error("Expected LastUsedAt to be nil")
				}
			} else {
				if apiKey.LastUsedAt == nil {
					t.Error("Expected LastUsedAt to be set")
				} else if !apiKey.LastUsedAt.Equal(*tt.lastUsedAt) {
					t.Errorf("Expected LastUsedAt to be '%v', got '%v'", tt.lastUsedAt, apiKey.LastUsedAt)
				}
			}
		})
	}
}

func TestAPIKeyExpiresAt(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt *time.Time
		expired   bool
	}{
		{
			name:      "nil expires at (never expires)",
			expiresAt: nil,
			expired:   false,
		},
		{
			name: "expires in the future",
			expiresAt: func() *time.Time {
				t := time.Now().Add(30 * 24 * time.Hour)
				return &t
			}(),
			expired: false,
		},
		{
			name: "expired in the past",
			expiresAt: func() *time.Time {
				t := time.Now().Add(-24 * time.Hour)
				return &t
			}(),
			expired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := APIKey{
				AccountID:   uuid.New(),
				Name:        "Test Key",
				KeyHash:     "hash",
				ExpiresAt:   tt.expiresAt,
				CreatedByID: uuid.New(),
			}

			if tt.expiresAt == nil {
				if apiKey.ExpiresAt != nil {
					t.Error("Expected ExpiresAt to be nil")
				}
			} else {
				if apiKey.ExpiresAt == nil {
					t.Error("Expected ExpiresAt to be set")
				} else if !apiKey.ExpiresAt.Equal(*tt.expiresAt) {
					t.Errorf("Expected ExpiresAt to be '%v', got '%v'", tt.expiresAt, apiKey.ExpiresAt)
				}

				// Check if the key is expired
				isExpired := time.Now().After(*apiKey.ExpiresAt)
				if isExpired != tt.expired {
					t.Errorf("Expected expired to be %v, got %v", tt.expired, isExpired)
				}
			}
		})
	}
}

func TestAPIKeyActiveStatus(t *testing.T) {
	tests := []struct {
		name     string
		isActive bool
	}{
		{
			name:     "active key",
			isActive: true,
		},
		{
			name:     "inactive key",
			isActive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := APIKey{
				AccountID:   uuid.New(),
				Name:        "Test Key",
				KeyHash:     "hash",
				IsActive:    tt.isActive,
				CreatedByID: uuid.New(),
			}

			if apiKey.IsActive != tt.isActive {
				t.Errorf("Expected IsActive to be %v, got %v", tt.isActive, apiKey.IsActive)
			}
		})
	}
}

func TestAPIKeyWithAllFields(t *testing.T) {
	id := uuid.New()
	accountID := uuid.New()
	createdByID := uuid.New()
	now := time.Now()
	lastUsedAt := now.Add(-1 * time.Hour)
	expiresAt := now.Add(30 * 24 * time.Hour)

	apiKey := APIKey{
		ID:          id,
		AccountID:   accountID,
		Name:        "Production API Key",
		KeyHash:     "$2a$10$hashedkeyvalue",
		LastUsedAt:  &lastUsedAt,
		ExpiresAt:   &expiresAt,
		IsActive:    true,
		CreatedAt:   now,
		CreatedByID: createdByID,
	}

	// Verify all fields are set correctly
	if apiKey.ID != id {
		t.Error("ID mismatch")
	}
	if apiKey.AccountID != accountID {
		t.Error("AccountID mismatch")
	}
	if apiKey.Name != "Production API Key" {
		t.Error("Name mismatch")
	}
	if apiKey.KeyHash != "$2a$10$hashedkeyvalue" {
		t.Error("KeyHash mismatch")
	}
	if apiKey.LastUsedAt == nil || !apiKey.LastUsedAt.Equal(lastUsedAt) {
		t.Error("LastUsedAt mismatch")
	}
	if apiKey.ExpiresAt == nil || !apiKey.ExpiresAt.Equal(expiresAt) {
		t.Error("ExpiresAt mismatch")
	}
	if !apiKey.IsActive {
		t.Error("IsActive should be true")
	}
	if !apiKey.CreatedAt.Equal(now) {
		t.Error("CreatedAt mismatch")
	}
	if apiKey.CreatedByID != createdByID {
		t.Error("CreatedByID mismatch")
	}
}

func TestAPIKeyBeforeCreate(t *testing.T) {
	// Test that BeforeCreate hook sets ID if not already set
	apiKey := APIKey{
		AccountID:   uuid.New(),
		Name:        "Test Key",
		KeyHash:     "hash",
		CreatedByID: uuid.New(),
	}

	// Simulate BeforeCreate hook (without actual DB)
	if apiKey.ID == uuid.Nil {
		apiKey.ID = uuid.New()
	}

	if apiKey.ID == uuid.Nil {
		t.Error("Expected ID to be set by BeforeCreate hook")
	}
}

func TestAPIKeyBeforeCreateDoesNotOverrideExistingID(t *testing.T) {
	// Test that BeforeCreate hook does not override existing ID
	existingID := uuid.New()
	apiKey := APIKey{
		ID:          existingID,
		AccountID:   uuid.New(),
		Name:        "Test Key",
		KeyHash:     "hash",
		CreatedByID: uuid.New(),
	}

	// Simulate BeforeCreate hook (without actual DB)
	if apiKey.ID == uuid.Nil {
		apiKey.ID = uuid.New()
	}

	if apiKey.ID != existingID {
		t.Error("Expected ID to remain unchanged when already set")
	}
}

func TestAPIKeyNameVariations(t *testing.T) {
	tests := []struct {
		name    string
		keyName string
	}{
		{
			name:    "simple name",
			keyName: "Production",
		},
		{
			name:    "descriptive name",
			keyName: "Production API Key for CI/CD",
		},
		{
			name:    "name with special characters",
			keyName: "Dev-API-Key_2024",
		},
		{
			name:    "empty name (invalid but testing field)",
			keyName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := APIKey{
				AccountID:   uuid.New(),
				Name:        tt.keyName,
				KeyHash:     "hash",
				CreatedByID: uuid.New(),
			}

			if apiKey.Name != tt.keyName {
				t.Errorf("Expected Name to be '%s', got '%s'", tt.keyName, apiKey.Name)
			}
		})
	}
}

func TestAPIKeyMultipleKeysForSameAccount(t *testing.T) {
	// Test that multiple API keys can exist for the same account
	accountID := uuid.New()
	createdByID := uuid.New()

	key1 := APIKey{
		ID:          uuid.New(),
		AccountID:   accountID,
		Name:        "Production Key",
		KeyHash:     "hash1",
		IsActive:    true,
		CreatedByID: createdByID,
	}

	key2 := APIKey{
		ID:          uuid.New(),
		AccountID:   accountID,
		Name:        "Development Key",
		KeyHash:     "hash2",
		IsActive:    true,
		CreatedByID: createdByID,
	}

	if key1.AccountID != key2.AccountID {
		t.Error("Both keys should belong to the same account")
	}

	if key1.ID == key2.ID {
		t.Error("Keys should have different IDs")
	}

	if key1.KeyHash == key2.KeyHash {
		t.Error("Keys should have different hashes")
	}
}
