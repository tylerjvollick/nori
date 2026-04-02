package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestAccountModelFields(t *testing.T) {
	// Test that Account struct has all required fields
	createdByUserID := uuid.New()
	name := "Test Account"

	account := Account{
		ID:              uuid.New(),
		Name:            &name,
		Plan:            Trial,
		CreatedByUserID: createdByUserID,
	}

	if account.ID == uuid.Nil {
		t.Error("Expected ID to be set")
	}

	if account.Name == nil || *account.Name != name {
		t.Errorf("Expected Name to be '%s', got '%v'", name, account.Name)
	}

	if account.Plan != Trial {
		t.Errorf("Expected Plan to be '%s', got '%s'", Trial, account.Plan)
	}

	if account.CreatedByUserID != createdByUserID {
		t.Errorf("Expected CreatedByUserID to be '%s', got '%s'", createdByUserID, account.CreatedByUserID)
	}
}

func TestPlanEnum(t *testing.T) {
	tests := []struct {
		name     string
		plan     Plan
		expected string
	}{
		{
			name:     "trial plan",
			plan:     Trial,
			expected: "trial",
		},
		{
			name:     "paid plan",
			plan:     Paid,
			expected: "paid",
		},
		{
			name:     "enterprise plan",
			plan:     Enterprise,
			expected: "enterprise",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.plan) != tt.expected {
				t.Errorf("Expected plan to be '%s', got '%s'", tt.expected, string(tt.plan))
			}
		})
	}
}

func TestAccountWithAllFields(t *testing.T) {
	id := uuid.New()
	createdByUserID := uuid.New()
	name := "Complete Account"

	account := Account{
		ID:              id,
		Name:            &name,
		Plan:            Paid,
		CreatedByUserID: createdByUserID,
	}

	// Verify all fields are set correctly
	if account.ID != id {
		t.Error("ID mismatch")
	}

	if account.Name == nil || *account.Name != name {
		t.Error("Name mismatch")
	}

	if account.Plan != Paid {
		t.Error("Plan mismatch")
	}

	if account.CreatedByUserID != createdByUserID {
		t.Error("CreatedByUserID mismatch")
	}
}

func TestAccountWithMinimalFields(t *testing.T) {
	// Test account with only required fields
	createdByUserID := uuid.New()

	account := Account{
		ID:              uuid.New(),
		Plan:            Trial,
		CreatedByUserID: createdByUserID,
	}

	if account.ID == uuid.Nil {
		t.Error("Expected ID to be set")
	}

	if account.Name != nil {
		t.Error("Expected Name to be nil when not set")
	}

	if account.Plan != Trial {
		t.Errorf("Expected Plan to be '%s', got '%s'", Trial, account.Plan)
	}

	if account.CreatedByUserID != createdByUserID {
		t.Error("CreatedByUserID mismatch")
	}
}

func TestAccountPlanDefaults(t *testing.T) {
	// Test that Plan can be set to different values
	tests := []struct {
		name     string
		plan     Plan
		expected Plan
	}{
		{
			name:     "default to trial",
			plan:     Trial,
			expected: Trial,
		},
		{
			name:     "set to paid",
			plan:     Paid,
			expected: Paid,
		},
		{
			name:     "set to enterprise",
			plan:     Enterprise,
			expected: Enterprise,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := Account{
				Plan:            tt.plan,
				CreatedByUserID: uuid.New(),
			}

			if account.Plan != tt.expected {
				t.Errorf("Expected Plan to be '%s', got '%s'", tt.expected, account.Plan)
			}
		})
	}
}

func TestAccountNameOptional(t *testing.T) {
	// Test that Name field is optional (can be nil)
	account := Account{
		ID:              uuid.New(),
		CreatedByUserID: uuid.New(),
		Plan:            Trial,
	}

	if account.Name != nil {
		t.Error("Expected Name to be nil when not set")
	}

	// Test setting Name
	name := "My Account"
	account.Name = &name

	if account.Name == nil {
		t.Error("Expected Name to be set")
	}

	if *account.Name != name {
		t.Errorf("Expected Name to be '%s', got '%s'", name, *account.Name)
	}
}
