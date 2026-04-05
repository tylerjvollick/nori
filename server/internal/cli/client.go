package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a simple HTTP client for CLI commands that adds the Authorization header.
type Client struct {
	ServerURL  string
	Token      string
	httpClient *http.Client
}

// NewClient creates a new CLI HTTP client from stored credentials.
func NewClient(creds *Credentials) *Client {
	return &Client{
		ServerURL: creds.ServerURL,
		Token:     creds.AccessToken,
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

// Post sends a POST request with JSON body and returns the response.
func (c *Client) Post(path string, body interface{}) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := c.ServerURL + path
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
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
