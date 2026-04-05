package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"golang.org/x/term"
)

// Client is a simple HTTP client for CLI commands that adds the Authorization header.
type Client struct {
	ServerURL  string
	Token      string
	httpClient *http.Client
}

// NewClient creates a new CLI HTTP client from stored credentials.
// Uses ActiveToken() to prefer API key over JWT.
func NewClient(creds *Credentials) *Client {
	return &Client{
		ServerURL: creds.ServerURL,
		Token:     creds.ActiveToken(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithURL creates a new CLI HTTP client with just a server URL (no auth token).
// Used for the login flow before credentials exist.
func NewClientWithURL(serverURL string) *Client {
	return &Client{
		ServerURL: serverURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest sends an HTTP request with the Authorization header set.
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	url := c.ServerURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// Post sends a POST request with JSON body and returns the response.
func (c *Client) Post(path string, body interface{}) (*http.Response, error) {
	return c.doRequest(http.MethodPost, path, body)
}

// Get sends a GET request and returns the response.
func (c *Client) Get(path string) (*http.Response, error) {
	return c.doRequest(http.MethodGet, path, nil)
}

// ReadJSON reads and unmarshals a JSON response body. Caller is responsible for
// closing the response body after this call.
func ReadJSON(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	return nil
}

// IsUnauthorized returns true if the response status is 401 Unauthorized.
func IsUnauthorized(resp *http.Response) bool {
	return resp.StatusCode == http.StatusUnauthorized
}

// isInteractiveFunc is a variable so it can be overridden in tests.
var isInteractiveFunc = defaultIsInteractive

// IsInteractive returns true if stdin is a terminal (not piped/redirected).
// Used to decide whether to prompt for re-authentication on 401.
func IsInteractive() bool {
	return isInteractiveFunc()
}

func defaultIsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
