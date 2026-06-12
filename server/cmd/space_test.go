package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tylerjvollick/nori/internal/cli"
)

// setSpaceUpdateFlags sets flags on spaceUpdateCmd and registers cleanup to
// reset them so tests do not leak state.
func setSpaceUpdateFlags(t *testing.T, values map[string]string) {
	t.Helper()
	for name, value := range values {
		require.NoError(t, spaceUpdateCmd.Flags().Set(name, value))
	}
	t.Cleanup(func() {
		spaceUpdateDefaultLaborRate = ""
		spaceUpdateClearDefaultLaborRate = false
		spaceUpdateCmd.Flags().VisitAll(func(f *pflag.Flag) {
			f.Changed = false
		})
	})
}

func TestSpaceCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "space" {
			found = true
			break
		}
	}
	assert.True(t, found, "space command should be registered on root")
}

func TestSpaceUpdateSubcommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range spaceCmd.Commands() {
		if cmd.Name() == "update" {
			found = true
			break
		}
	}
	assert.True(t, found, "update subcommand should be registered on space")
}

func TestRunSpaceUpdate_SetDefaultLaborRate(t *testing.T) {
	rate := "50"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/spaces/space-123", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		var body map[string]json.RawMessage
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Contains(t, body, "defaultLaborRate")
		assert.Equal(t, `"50"`, string(body["defaultLaborRate"]))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(spaceItem{
			ID: "space-123", Name: "Shop", Slug: "SHP", DefaultLaborRate: &rate,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	t.Setenv("NORI_SPACE", "space-123")
	setSpaceUpdateFlags(t, map[string]string{"default-labor-rate": "50.00"})
	spaceJSONFlag = false

	err := runSpaceUpdate(spaceUpdateCmd, nil)
	require.NoError(t, err)
}

func TestRunSpaceUpdate_ClearDefaultLaborRate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/spaces/space-123", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		var body map[string]json.RawMessage
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Contains(t, body, "defaultLaborRate")
		assert.Equal(t, "null", string(body["defaultLaborRate"]), "clearing should send explicit null")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(spaceItem{
			ID: "space-123", Name: "Shop", Slug: "SHP", DefaultLaborRate: nil,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	t.Setenv("NORI_SPACE", "space-123")
	setSpaceUpdateFlags(t, map[string]string{"clear-default-labor-rate": "true"})
	spaceJSONFlag = false

	err := runSpaceUpdate(spaceUpdateCmd, nil)
	require.NoError(t, err)
}

func TestRunSpaceUpdate_InvalidRate(t *testing.T) {
	setupCredentials(t, "http://localhost:1") // never reached
	setSpaceUpdateFlags(t, map[string]string{"default-labor-rate": "abc"})

	err := runSpaceUpdate(spaceUpdateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --default-labor-rate")
}

func TestRunSpaceUpdate_NegativeRate(t *testing.T) {
	setupCredentials(t, "http://localhost:1") // never reached
	setSpaceUpdateFlags(t, map[string]string{"default-labor-rate": "-1"})

	err := runSpaceUpdate(spaceUpdateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not be negative")
}

func TestRunSpaceUpdate_NoFlags(t *testing.T) {
	setupCredentials(t, "http://localhost:1") // never reached
	setSpaceUpdateFlags(t, map[string]string{})

	err := runSpaceUpdate(spaceUpdateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nothing to update")
}

func TestRunSpaceUpdate_NoSpaceSelected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be reached without a space")
	}))
	defer server.Close()

	// Credentials without a space ID (setupCredentials sets one).
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	require.NoError(t, cli.SaveCredentials(&cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: "test-token",
		UserID:      "user-123",
		UserEmail:   "test@example.com",
	}))
	t.Setenv("NORI_SPACE", "")
	setSpaceUpdateFlags(t, map[string]string{"default-labor-rate": "50.00"})

	err := runSpaceUpdate(spaceUpdateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no space selected")
}

func TestRunSpaceUpdate_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse{Error: "defaultLaborRate must not be negative"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	t.Setenv("NORI_SPACE", "space-123")
	setSpaceUpdateFlags(t, map[string]string{"default-labor-rate": "50.00"})

	err := runSpaceUpdate(spaceUpdateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "defaultLaborRate must not be negative")
}
