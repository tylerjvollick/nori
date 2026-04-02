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

func TestLoad_MissingAccountName(t *testing.T) {
	// Set all required vars except account name
	os.Setenv("NORI_JWT_SECRET", "test-secret-key-at-least-32-chars")
	os.Setenv("NORI_ADMIN_EMAIL", "admin@test.com")
	os.Setenv("NORI_ADMIN_PASSWORD", "testpassword123")
	defer func() {
		os.Unsetenv("NORI_JWT_SECRET")
		os.Unsetenv("NORI_ADMIN_EMAIL")
		os.Unsetenv("NORI_ADMIN_PASSWORD")
	}()

	_, err := Load()

	if err == nil {
		t.Fatal("Expected error for missing NORI_ACCOUNT_NAME, got nil")
	}

	expectedMsg := "NORI_ACCOUNT_NAME is required"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
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

func TestValidate_EmptyAccountName(t *testing.T) {
	cfg := &Config{
		JWTSecret:     "valid-secret-key",
		AdminEmail:    "admin@test.com",
		AdminPassword: "password123",
		AccountName:   "",
	}

	err := cfg.Validate()

	if err == nil {
		t.Fatal("Expected error for empty AccountName, got nil")
	}
}
