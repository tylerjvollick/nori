package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tylerjvollick/nori/internal/cli"
)

func TestConfigCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "config" {
			found = true
			// Verify subcommands
			subCmds := cmd.Commands()
			setFound := false
			showFound := false
			for _, sub := range subCmds {
				if sub.Name() == "set" {
					setFound = true
				}
				if sub.Name() == "show" {
					showFound = true
				}
			}
			assert.True(t, setFound, "config set subcommand should be registered")
			assert.True(t, showFound, "config show subcommand should be registered")
			break
		}
	}
	assert.True(t, found, "config command should be registered on root")
}

func TestConfigSet_APIKey(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// First, create some existing credentials
	creds := &cli.Credentials{
		ServerURL:   "http://localhost:8080",
		AccessToken: "existing-jwt",
		UserID:      "user-123",
		UserEmail:   "test@example.com",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	// Set API key via the command function
	err := setAPIKey("nori_abc123def456789012345678901234567890123456789012345678901234")
	require.NoError(t, err)

	// Verify credentials were updated
	loaded, err := cli.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "nori_abc123def456789012345678901234567890123456789012345678901234", loaded.APIKey)
	assert.Equal(t, "existing-jwt", loaded.AccessToken, "JWT should be preserved")
	assert.Equal(t, "http://localhost:8080", loaded.ServerURL, "server URL should be preserved")
	assert.Equal(t, "test@example.com", loaded.UserEmail, "email should be preserved")
}

func TestConfigSet_APIKey_InvalidPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := setAPIKey("invalid-key-no-nori-prefix")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must start with 'nori_'")
}

func TestConfigSet_APIKey_NoExistingCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Set server URL first (required for valid credentials)
	err := setServerURL("http://localhost:8080")
	require.NoError(t, err)

	// Set API key
	err = setAPIKey("nori_abc123def456789012345678901234567890123456789012345678901234")
	require.NoError(t, err)

	// Verify file was created with the API key
	loaded, err := cli.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "nori_abc123def456789012345678901234567890123456789012345678901234", loaded.APIKey)
	assert.Equal(t, "http://localhost:8080", loaded.ServerURL)
}

func TestConfigSet_ServerURL(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create existing credentials
	creds := &cli.Credentials{
		ServerURL:   "http://localhost:8080",
		AccessToken: "existing-jwt",
		UserID:      "user-123",
		UserEmail:   "test@example.com",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	err := setServerURL("https://nori.example.com")
	require.NoError(t, err)

	loaded, err := cli.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "https://nori.example.com", loaded.ServerURL)
	assert.Equal(t, "existing-jwt", loaded.AccessToken, "JWT should be preserved")
}

func TestConfigSet_ServerURL_Normalization(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := &cli.Credentials{
		ServerURL:   "http://localhost:8080",
		AccessToken: "jwt",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	err := setServerURL("nori.example.com/")
	require.NoError(t, err)

	loaded, err := cli.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "http://nori.example.com", loaded.ServerURL, "URL should be normalized")
}

func TestConfigSet_UnknownKey(t *testing.T) {
	err := runConfigSet(nil, []string{"unknown-key", "value"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown config key")
}

func TestConfigSet_Space(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create existing credentials
	creds := &cli.Credentials{
		ServerURL:   "http://localhost:8080",
		AccessToken: "existing-jwt",
		UserID:      "user-123",
		UserEmail:   "test@example.com",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	err := setSpace("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)

	loaded, err := cli.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", loaded.SpaceID)
	assert.Equal(t, "existing-jwt", loaded.AccessToken, "JWT should be preserved")
	assert.Equal(t, "http://localhost:8080", loaded.ServerURL, "server URL should be preserved")
}

func TestConfigSet_Space_NoExistingCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := setSpace("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)

	// File should exist with space set
	loaded, err := cli.LoadCredentialsRaw()
	require.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", loaded.SpaceID)
}

func TestConfigShow_WithSpace(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := &cli.Credentials{
		ServerURL:   "http://localhost:8080",
		AccessToken: "jwt-token-12345678",
		UserID:      "user-123",
		UserEmail:   "test@example.com",
		SpaceID:     "550e8400-e29b-41d4-a716-446655440000",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	// Should not error
	err := runConfigShow(nil, nil)
	require.NoError(t, err)
}

func TestConfigShow(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := &cli.Credentials{
		ServerURL:   "http://localhost:8080",
		AccessToken: "jwt-token-12345678",
		UserID:      "user-123",
		UserEmail:   "test@example.com",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	// Should not error
	err := runConfigShow(nil, nil)
	require.NoError(t, err)
}

func TestConfigShow_WithAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := &cli.Credentials{
		ServerURL:   "http://localhost:8080",
		AccessToken: "jwt-token-12345678",
		APIKey:      "nori_abc123def456",
		UserID:      "user-123",
		UserEmail:   "test@example.com",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	// Should not error
	err := runConfigShow(nil, nil)
	require.NoError(t, err)
}

func TestConfigShow_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := runConfigShow(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no configuration found")
}

func TestConfigSet_APIKey_PreservesFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := &cli.Credentials{
		ServerURL:   "http://localhost:8080",
		AccessToken: "jwt",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	err := setAPIKey("nori_abc123def456789012345678901234567890123456789012345678901234")
	require.NoError(t, err)

	// Verify file permissions are still 0600
	path := filepath.Join(tmpDir, ".config", "nori", "credentials")
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestConfigShow_AuthMethodDisplay(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// JWT only
	creds := &cli.Credentials{
		ServerURL:   "http://localhost:8080",
		AccessToken: "jwt-token-12345678",
		UserEmail:   "test@example.com",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	loaded, err := cli.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "JWT", loaded.AuthMethod())

	// Add API key — should switch to "API key"
	creds.APIKey = "nori_abc123def456"
	require.NoError(t, cli.SaveCredentials(creds))

	loaded, err = cli.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "API key", loaded.AuthMethod())
}

func TestCredentialFile_APIKeyPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Save credentials with API key
	creds := &cli.Credentials{
		ServerURL: "http://localhost:8080",
		APIKey:    "nori_abc123def456",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	// Read the raw file to verify JSON structure
	path := filepath.Join(tmpDir, ".config", "nori", "credentials")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, "nori_abc123def456", raw["apiKey"])
}
