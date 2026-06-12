package config

import (
	"os"
	"testing"
)

func TestLoad_Success(t *testing.T) {
	// Set all required environment variables
	os.Setenv("NORI_JWT_SECRET", "test-secret-key-at-least-32-chars")
	os.Setenv("NORI_ADMIN_EMAIL", "admin@test.com")
	os.Setenv("NORI_ADMIN_PASSWORD", "testpassword123")
	os.Setenv("NORI_ACCOUNT_NAME", "Test Account")
	defer func() {
		os.Unsetenv("NORI_JWT_SECRET")
		os.Unsetenv("NORI_ADMIN_EMAIL")
		os.Unsetenv("NORI_ADMIN_PASSWORD")
		os.Unsetenv("NORI_ACCOUNT_NAME")
	}()

	cfg, err := Load()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.JWTSecret != "test-secret-key-at-least-32-chars" {
		t.Errorf("Expected JWTSecret to be 'test-secret-key-at-least-32-chars', got: %s", cfg.JWTSecret)
	}

	if cfg.AdminEmail != "admin@test.com" {
		t.Errorf("Expected AdminEmail to be 'admin@test.com', got: %s", cfg.AdminEmail)
	}

	if cfg.AdminPassword != "testpassword123" {
		t.Errorf("Expected AdminPassword to be 'testpassword123', got: %s", cfg.AdminPassword)
	}

	if cfg.AccountName != "Test Account" {
		t.Errorf("Expected AccountName to be 'Test Account', got: %s", cfg.AccountName)
	}

	if cfg.SkipPasswordChange != false {
		t.Error("Expected SkipPasswordChange to default to false when env var is not set")
	}
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	// Set all required vars except JWT secret
	os.Setenv("NORI_ADMIN_EMAIL", "admin@test.com")
	os.Setenv("NORI_ADMIN_PASSWORD", "testpassword123")
	os.Setenv("NORI_ACCOUNT_NAME", "Test Account")
	defer func() {
		os.Unsetenv("NORI_ADMIN_EMAIL")
		os.Unsetenv("NORI_ADMIN_PASSWORD")
		os.Unsetenv("NORI_ACCOUNT_NAME")
	}()

	_, err := Load()

	if err == nil {
		t.Fatal("Expected error for missing NORI_JWT_SECRET, got nil")
	}

	expectedMsg := "NORI_JWT_SECRET is required"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestLoad_MissingAdminEmail(t *testing.T) {
	// Set all required vars except admin email
	os.Setenv("NORI_JWT_SECRET", "test-secret-key-at-least-32-chars")
	os.Setenv("NORI_ADMIN_PASSWORD", "testpassword123")
	os.Setenv("NORI_ACCOUNT_NAME", "Test Account")
	defer func() {
		os.Unsetenv("NORI_JWT_SECRET")
		os.Unsetenv("NORI_ADMIN_PASSWORD")
		os.Unsetenv("NORI_ACCOUNT_NAME")
	}()

	_, err := Load()

	if err == nil {
		t.Fatal("Expected error for missing NORI_ADMIN_EMAIL, got nil")
	}

	expectedMsg := "NORI_ADMIN_EMAIL is required"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestLoad_MissingAdminPassword(t *testing.T) {
	// Set all required vars except admin password
	os.Setenv("NORI_JWT_SECRET", "test-secret-key-at-least-32-chars")
	os.Setenv("NORI_ADMIN_EMAIL", "admin@test.com")
	os.Setenv("NORI_ACCOUNT_NAME", "Test Account")
	defer func() {
		os.Unsetenv("NORI_JWT_SECRET")
		os.Unsetenv("NORI_ADMIN_EMAIL")
		os.Unsetenv("NORI_ACCOUNT_NAME")
	}()

	_, err := Load()

	if err == nil {
		t.Fatal("Expected error for missing NORI_ADMIN_PASSWORD, got nil")
	}

	expectedMsg := "NORI_ADMIN_PASSWORD is required"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestLoad_MissingAccountName_DefaultsToMyShop(t *testing.T) {
	os.Setenv("NORI_JWT_SECRET", "test-secret-key-at-least-32-chars")
	os.Setenv("NORI_ADMIN_EMAIL", "admin@test.com")
	os.Setenv("NORI_ADMIN_PASSWORD", "testpassword123")
	os.Unsetenv("NORI_ACCOUNT_NAME")
	defer func() {
		os.Unsetenv("NORI_JWT_SECRET")
		os.Unsetenv("NORI_ADMIN_EMAIL")
		os.Unsetenv("NORI_ADMIN_PASSWORD")
	}()

	cfg, err := Load()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.AccountName != "My Shop" {
		t.Errorf("Expected AccountName to default to 'My Shop', got: %s", cfg.AccountName)
	}
}

func TestValidate_Success(t *testing.T) {
	cfg := &Config{
		JWTSecret:     "valid-secret-key",
		AdminEmail:    "admin@test.com",
		AdminPassword: "password123",
		AccountName:   "Test Account",
	}

	err := cfg.Validate()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestValidate_EmptyJWTSecret(t *testing.T) {
	cfg := &Config{
		JWTSecret:     "",
		AdminEmail:    "admin@test.com",
		AdminPassword: "password123",
		AccountName:   "Test Account",
	}

	err := cfg.Validate()

	if err == nil {
		t.Fatal("Expected error for empty JWTSecret, got nil")
	}
}

func TestValidate_EmptyAdminEmail(t *testing.T) {
	cfg := &Config{
		JWTSecret:     "valid-secret-key",
		AdminEmail:    "",
		AdminPassword: "password123",
		AccountName:   "Test Account",
	}

	err := cfg.Validate()

	if err == nil {
		t.Fatal("Expected error for empty AdminEmail, got nil")
	}
}

func TestValidate_EmptyAdminPassword(t *testing.T) {
	cfg := &Config{
		JWTSecret:     "valid-secret-key",
		AdminEmail:    "admin@test.com",
		AdminPassword: "",
		AccountName:   "Test Account",
	}

	err := cfg.Validate()

	if err == nil {
		t.Fatal("Expected error for empty AdminPassword, got nil")
	}
}

func TestValidate_EmptyAccountName_Succeeds(t *testing.T) {
	// AccountName is optional (defaults to "My Shop" in Load)
	cfg := &Config{
		JWTSecret:     "valid-secret-key",
		AdminEmail:    "admin@test.com",
		AdminPassword: "password123",
		AccountName:   "",
	}

	err := cfg.Validate()

	if err != nil {
		t.Fatalf("Expected no error for empty AccountName, got: %v", err)
	}
}

func TestValidate_E2EAccountEnabled_MissingEmail(t *testing.T) {
	cfg := &Config{
		JWTSecret:         "valid-secret-key",
		AdminEmail:        "admin@test.com",
		AdminPassword:     "password123",
		AccountName:       "Test",
		E2EAccountEnabled: true,
		E2EAccountEmail:   "",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for missing E2E email, got nil")
	}
}

func TestValidate_E2EAccountEnabled_MissingPassword(t *testing.T) {
	cfg := &Config{
		JWTSecret:          "valid-secret-key",
		AdminEmail:         "admin@test.com",
		AdminPassword:      "password123",
		AccountName:        "Test",
		E2EAccountEnabled:  true,
		E2EAccountEmail:    "e2e@test.com",
		E2EAccountPassword: "",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for missing E2E password, got nil")
	}
}

func TestValidate_E2EAccountDisabled_NoValidation(t *testing.T) {
	cfg := &Config{
		JWTSecret:         "valid-secret-key",
		AdminEmail:        "admin@test.com",
		AdminPassword:     "password123",
		AccountName:       "Test",
		E2EAccountEnabled: false,
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Expected no error when E2E disabled, got: %v", err)
	}
}

func TestLoad_SkipPasswordChangeTrue(t *testing.T) {
	os.Setenv("NORI_JWT_SECRET", "test-secret-key-at-least-32-chars")
	os.Setenv("NORI_ADMIN_EMAIL", "admin@test.com")
	os.Setenv("NORI_ADMIN_PASSWORD", "testpassword123")
	os.Setenv("NORI_ACCOUNT_NAME", "Test Account")
	os.Setenv("NORI_SKIP_PASSWORD_CHANGE", "true")
	defer func() {
		os.Unsetenv("NORI_JWT_SECRET")
		os.Unsetenv("NORI_ADMIN_EMAIL")
		os.Unsetenv("NORI_ADMIN_PASSWORD")
		os.Unsetenv("NORI_ACCOUNT_NAME")
		os.Unsetenv("NORI_SKIP_PASSWORD_CHANGE")
	}()

	cfg, err := Load()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !cfg.SkipPasswordChange {
		t.Error("Expected SkipPasswordChange to be true when NORI_SKIP_PASSWORD_CHANGE=true")
	}
}

func TestLoad_SkipPasswordChangeFalseByDefault(t *testing.T) {
	os.Setenv("NORI_JWT_SECRET", "test-secret-key-at-least-32-chars")
	os.Setenv("NORI_ADMIN_EMAIL", "admin@test.com")
	os.Setenv("NORI_ADMIN_PASSWORD", "testpassword123")
	os.Setenv("NORI_ACCOUNT_NAME", "Test Account")
	os.Unsetenv("NORI_SKIP_PASSWORD_CHANGE")
	defer func() {
		os.Unsetenv("NORI_JWT_SECRET")
		os.Unsetenv("NORI_ADMIN_EMAIL")
		os.Unsetenv("NORI_ADMIN_PASSWORD")
		os.Unsetenv("NORI_ACCOUNT_NAME")
	}()

	cfg, err := Load()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.SkipPasswordChange {
		t.Error("Expected SkipPasswordChange to be false when NORI_SKIP_PASSWORD_CHANGE is not set")
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"1", true},
		{"yes", true},
		{"Yes", true},
		{"YES", true},
		{"false", false},
		{"False", false},
		{"0", false},
		{"no", false},
		{"", false},
		{"  true  ", true},
		{"anything", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseBool(tt.input)
			if result != tt.expected {
				t.Errorf("parseBool(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidate_SkipPasswordChangeNotRequired(t *testing.T) {
	// SkipPasswordChange is optional - validation should pass regardless of its value
	cfg := &Config{
		JWTSecret:          "valid-secret-key",
		AdminEmail:         "admin@test.com",
		AdminPassword:      "password123",
		AccountName:        "Test Account",
		SkipPasswordChange: false,
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Expected no error with SkipPasswordChange=false, got: %v", err)
	}

	cfg.SkipPasswordChange = true
	err = cfg.Validate()
	if err != nil {
		t.Fatalf("Expected no error with SkipPasswordChange=true, got: %v", err)
	}
}
