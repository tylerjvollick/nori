package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoadCredentials(t *testing.T) {
	// Use a temp dir to avoid touching the real ~/.config/nori
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	creds := &Credentials{
		ServerURL:   "http://localhost:8080",
		AccessToken: "test-jwt-token",
		UserID:      "user-123",
		UserEmail:   "test@example.com",
	}

	// Save
	err := SaveCredentials(creds)
	require.NoError(t, err)

	// Verify file exists at expected path
	path := filepath.Join(tmpDir, ".config", "nori", "credentials")
	info, err := os.Stat(path)
	require.NoError(t, err)

	// Verify file permissions are 0600
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Load
	loaded, err := LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, creds.ServerURL, loaded.ServerURL)
	assert.Equal(t, creds.AccessToken, loaded.AccessToken)
	assert.Equal(t, creds.UserID, loaded.UserID)
	assert.Equal(t, creds.UserEmail, loaded.UserEmail)
}

func TestLoadCredentials_NotLoggedIn(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	_, err := LoadCredentials()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

func TestLoadCredentials_CorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	dir := filepath.Join(tmpDir, ".config", "nori")
	require.NoError(t, os.MkdirAll(dir, 0700))

	path := filepath.Join(dir, "credentials")
	require.NoError(t, os.WriteFile(path, []byte("not-json"), 0600))

	_, err := LoadCredentials()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "corrupted")
}

func TestLoadCredentials_IncompleteCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	dir := filepath.Join(tmpDir, ".config", "nori")
	require.NoError(t, os.MkdirAll(dir, 0700))

	// Missing accessToken
	creds := &Credentials{
		ServerURL: "http://localhost:8080",
		UserEmail: "test@example.com",
	}
	data, _ := json.Marshal(creds)
	path := filepath.Join(dir, "credentials")
	require.NoError(t, os.WriteFile(path, data, 0600))

	_, err := LoadCredentials()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "incomplete credentials")
}

func TestSaveCredentials_OverwritesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds1 := &Credentials{
		ServerURL:   "http://localhost:8080",
		AccessToken: "token-1",
		UserID:      "user-1",
		UserEmail:   "user1@example.com",
	}
	require.NoError(t, SaveCredentials(creds1))

	creds2 := &Credentials{
		ServerURL:   "http://localhost:9090",
		AccessToken: "token-2",
		UserID:      "user-2",
		UserEmail:   "user2@example.com",
	}
	require.NoError(t, SaveCredentials(creds2))

	loaded, err := LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "token-2", loaded.AccessToken)
	assert.Equal(t, "http://localhost:9090", loaded.ServerURL)
}

func TestDirectoryPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := &Credentials{
		ServerURL:   "http://localhost:8080",
		AccessToken: "test-token",
		UserID:      "user-123",
		UserEmail:   "test@example.com",
	}
	require.NoError(t, SaveCredentials(creds))

	dirPath := filepath.Join(tmpDir, ".config", "nori")
	info, err := os.Stat(dirPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
}
