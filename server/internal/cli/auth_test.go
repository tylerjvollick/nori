package cli

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandle401_NotUnauthorized(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusOK}
	err := Handle401(resp)
	assert.NoError(t, err)
}

func TestHandle401_UnauthorizedInteractive(t *testing.T) {
	// Override IsInteractive to return true
	origFunc := isInteractiveFunc
	isInteractiveFunc = func() bool { return true }
	defer func() { isInteractiveFunc = origFunc }()

	resp := &http.Response{StatusCode: http.StatusUnauthorized}
	err := Handle401(resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run 'nori login' to re-authenticate")
	assert.NotContains(t, err.Error(), "non-interactive")
}

func TestHandle401_UnauthorizedNonInteractive(t *testing.T) {
	// Override IsInteractive to return false
	origFunc := isInteractiveFunc
	isInteractiveFunc = func() bool { return false }
	defer func() { isInteractiveFunc = origFunc }()

	resp := &http.Response{StatusCode: http.StatusUnauthorized}
	err := Handle401(resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-interactive mode")
	assert.Contains(t, err.Error(), "nori config set api-key")
}

func TestHandle401_Integration(t *testing.T) {
	// Spin up a server that always returns 401
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	// Override IsInteractive for predictable behavior
	origFunc := isInteractiveFunc
	isInteractiveFunc = func() bool { return true }
	defer func() { isInteractiveFunc = origFunc }()

	client := &Client{
		ServerURL:  server.URL,
		Token:      "expired-token",
		httpClient: server.Client(),
	}

	resp, err := client.Get("/some/endpoint")
	require.NoError(t, err)

	err = Handle401(resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run 'nori login' to re-authenticate")
}

func TestHandle401_NonUnauthorizedErrors(t *testing.T) {
	// 403 should not trigger 401 handling
	resp := &http.Response{StatusCode: http.StatusForbidden}
	err := Handle401(resp)
	assert.NoError(t, err)

	// 500 should not trigger 401 handling
	resp = &http.Response{StatusCode: http.StatusInternalServerError}
	err = Handle401(resp)
	assert.NoError(t, err)
}
