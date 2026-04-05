package cli

import (
	"fmt"
	"net/http"
)

// Handle401 checks if a response is 401 Unauthorized and takes appropriate action.
// In interactive mode, it prints a message prompting the user to re-authenticate.
// In non-interactive mode (piped stdin, scripts), it returns an error with a clear message.
// Returns nil if the response is not 401 (caller should continue normal processing).
// Returns a non-nil error if the response is 401 (caller should abort).
func Handle401(resp *http.Response) error {
	if resp.StatusCode != http.StatusUnauthorized {
		return nil
	}

	if IsInteractive() {
		return fmt.Errorf("authentication expired or invalid — run 'nori login' to re-authenticate")
	}

	return fmt.Errorf("authentication expired or invalid (non-interactive mode) — re-run 'nori login' or set a valid API key with 'nori config set api-key <key>'")
}
