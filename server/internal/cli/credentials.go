package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Credentials holds the stored authentication data for CLI usage.
type Credentials struct {
	ServerURL   string `json:"serverUrl"`
	AccessToken string `json:"accessToken"`
	APIKey      string `json:"apiKey,omitempty"`
	UserID      string `json:"userId"`
	UserEmail   string `json:"userEmail"`
	SpaceID     string `json:"spaceId,omitempty"`
}

// AuthMethod returns a human-readable description of the current authentication method.
func (c *Credentials) AuthMethod() string {
	if c.APIKey != "" {
		return "API key"
	}
	if c.AccessToken != "" {
		return "JWT"
	}
	return "none"
}

// ActiveToken returns the token to use for authentication.
// API key is preferred over JWT if both are configured.
func (c *Credentials) ActiveToken() string {
	if c.APIKey != "" {
		return c.APIKey
	}
	return c.AccessToken
}

// credentialsDir returns the path to the nori config directory (~/.config/nori).
func credentialsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "nori"), nil
}

// credentialsPath returns the full path to the credentials file.
func credentialsPath() (string, error) {
	dir, err := credentialsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials"), nil
}

// SaveCredentials writes credentials to ~/.config/nori/credentials with 0600 permissions.
func SaveCredentials(creds *Credentials) error {
	dir, err := credentialsDir()
	if err != nil {
		return err
	}

	// Create the directory with restrictive permissions (owner-only)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}

	path, err := credentialsPath()
	if err != nil {
		return err
	}

	// Write with restrictive permissions (owner read/write only)
	return os.WriteFile(path, data, 0600)
}

// LoadCredentials reads credentials from ~/.config/nori/credentials.
// Returns an error if the file doesn't exist or can't be read.
// Validates that essential fields (serverURL and at least one auth method) are present.
func LoadCredentials() (*Credentials, error) {
	creds, err := LoadCredentialsRaw()
	if err != nil {
		return nil, err
	}

	if creds.ServerURL == "" {
		return nil, errors.New("incomplete credentials — run 'nori login' to re-authenticate")
	}

	if creds.AccessToken == "" && creds.APIKey == "" {
		return nil, errors.New("incomplete credentials — run 'nori login' or 'nori config set api-key <key>' to authenticate")
	}

	return creds, nil
}

// LoadCredentialsRaw reads credentials from ~/.config/nori/credentials without
// validating completeness. Used by config commands that need to update partial
// credentials (e.g., setting API key before login).
func LoadCredentialsRaw() (*Credentials, error) {
	path, err := credentialsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("not logged in — run 'nori login' first")
		}
		return nil, err
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, errors.New("credentials file is corrupted — run 'nori login' to re-authenticate")
	}

	return &creds, nil
}
