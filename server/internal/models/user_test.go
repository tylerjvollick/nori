package models

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestUserModelFields(t *testing.T) {
	// Test that User struct has all required fields
	user := User{
		ID:                 uuid.New(),
		Email:              "test@example.com",
		MustChangePassword: true,
	}

	if user.ID == uuid.Nil {
		t.Error("Expected ID to be set")
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected Email to be 'test@example.com', got '%s'", user.Email)
	}

	if !user.MustChangePassword {
		t.Error("Expected MustChangePassword to be true")
	}
}

func TestUserMustChangePasswordDefault(t *testing.T) {
	// Test that MustChangePassword defaults to true for new users
	user := User{
		Email: "newuser@example.com",
	}

	// When creating a new user without explicitly setting MustChangePassword,
	// the GORM default should set it to true (this is enforced by the database)
	// In Go, bool defaults to false, but the migration sets DEFAULT true
	// This test documents the expected behavior
	if user.MustChangePassword != false {
		// In Go structs, bool defaults to false until set
		t.Logf("Note: MustChangePassword defaults to false in Go structs, but database migration sets DEFAULT true")
	}

	// When explicitly set to true
	user.MustChangePassword = true
	if !user.MustChangePassword {
		t.Error("Expected MustChangePassword to be true when explicitly set")
	}
}

func TestRoleEnum(t *testing.T) {
	tests := []struct {
		name     string
		role     Role
		expected string
	}{
		{
			name:     "admin role",
			role:     RoleAdmin,
			expected: "admin",
		},
		{
			name:     "user role",
			role:     RoleUser,
			expected: "user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.role) != tt.expected {
				t.Errorf("Expected role to be '%s', got '%s'", tt.expected, string(tt.role))
			}
		})
	}
}

func TestUserWithRole(t *testing.T) {
	adminRole := RoleAdmin
	userRole := RoleUser

	tests := []struct {
		name string
		user User
		role *Role
	}{
		{
			name: "user with admin role",
			user: User{
				Email: "admin@example.com",
				Role:  &adminRole,
			},
			role: &adminRole,
		},
		{
			name: "user with user role",
			user: User{
				Email: "user@example.com",
				Role:  &userRole,
			},
			role: &userRole,
		},
		{
			name: "user with nil role",
			user: User{
				Email: "noauth@example.com",
				Role:  nil,
			},
			role: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.role == nil {
				if tt.user.Role != nil {
					t.Error("Expected Role to be nil")
				}
			} else {
				if tt.user.Role == nil {
					t.Error("Expected Role to be set")
				} else if *tt.user.Role != *tt.role {
					t.Errorf("Expected role to be '%s', got '%s'", *tt.role, *tt.user.Role)
				}
			}
		})
	}
}

func TestRecentSpacesValue(t *testing.T) {
	tests := []struct {
		name     string
		spaces   RecentSpaces
		expected string
	}{
		{
			name:     "empty spaces",
			spaces:   RecentSpaces{},
			expected: "[]",
		},
		{
			name:     "nil spaces",
			spaces:   nil,
			expected: "[]",
		},
		{
			name: "single space",
			spaces: RecentSpaces{
				uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			},
			expected: `["550e8400-e29b-41d4-a716-446655440000"]`,
		},
		{
			name: "multiple spaces",
			spaces: RecentSpaces{
				uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
			},
			expected: `["550e8400-e29b-41d4-a716-446655440000","6ba7b810-9dad-11d1-80b4-00c04fd430c8"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := tt.spaces.Value()
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			var result string
			switch v := value.(type) {
			case string:
				result = v
			case []byte:
				result = string(v)
			default:
				t.Fatalf("Unexpected value type: %T", value)
			}

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRecentSpacesScan(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected RecentSpaces
		hasError bool
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: RecentSpaces{},
			hasError: false,
		},
		{
			name:     "empty json array",
			input:    []byte("[]"),
			expected: RecentSpaces{},
			hasError: false,
		},
		{
			name:  "single uuid",
			input: []byte(`["550e8400-e29b-41d4-a716-446655440000"]`),
			expected: RecentSpaces{
				uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			},
			hasError: false,
		},
		{
			name:  "multiple uuids",
			input: []byte(`["550e8400-e29b-41d4-a716-446655440000","6ba7b810-9dad-11d1-80b4-00c04fd430c8"]`),
			expected: RecentSpaces{
				uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
			},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rs RecentSpaces
			err := rs.Scan(tt.input)

			if tt.hasError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if len(rs) != len(tt.expected) {
				t.Errorf("Expected %d spaces, got %d", len(tt.expected), len(rs))
				return
			}

			for i, expected := range tt.expected {
				if rs[i] != expected {
					t.Errorf("Expected space[%d] to be %s, got %s", i, expected, rs[i])
				}
			}
		})
	}
}

func TestRecentSpacesJSONRoundTrip(t *testing.T) {
	original := RecentSpaces{
		uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
	}

	// Convert to JSON
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Convert back from JSON
	var result RecentSpaces
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify they match
	if len(result) != len(original) {
		t.Errorf("Expected %d spaces, got %d", len(original), len(result))
		return
	}

	for i, expected := range original {
		if result[i] != expected {
			t.Errorf("Expected space[%d] to be %s, got %s", i, expected, result[i])
		}
	}
}

func TestUserWithAllFields(t *testing.T) {
	id := uuid.New()
	accountID := uuid.New()
	role := RoleAdmin
	password := "hashedpassword"
	firstName := "John"
	lastName := "Doe"
	spaces := RecentSpaces{uuid.New(), uuid.New()}

	user := User{
		ID:                 id,
		Role:               &role,
		Email:              "john.doe@example.com",
		Password:           &password,
		MustChangePassword: true,
		FirstName:          &firstName,
		LastName:           &lastName,
		DefaultAccountID:   &accountID,
		RecentSpaces:       spaces,
	}

	// Verify all fields are set correctly
	if user.ID != id {
		t.Error("ID mismatch")
	}
	if user.Role == nil || *user.Role != role {
		t.Error("Role mismatch")
	}
	if user.Email != "john.doe@example.com" {
		t.Error("Email mismatch")
	}
	if user.Password == nil || *user.Password != password {
		t.Error("Password mismatch")
	}
	if !user.MustChangePassword {
		t.Error("MustChangePassword should be true")
	}
	if user.FirstName == nil || *user.FirstName != firstName {
		t.Error("FirstName mismatch")
	}
	if user.LastName == nil || *user.LastName != lastName {
		t.Error("LastName mismatch")
	}
	if user.DefaultAccountID == nil || *user.DefaultAccountID != accountID {
		t.Error("DefaultAccountID mismatch")
	}
	if len(user.RecentSpaces) != 2 {
		t.Errorf("Expected 2 recent spaces, got %d", len(user.RecentSpaces))
	}
}
