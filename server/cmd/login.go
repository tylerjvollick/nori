package cmd

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/tylerjvollick/nori/internal/cli"
)

// loginResponse mirrors services.LoginResponse for JSON deserialization.
type loginResponse struct {
	AccessToken        string  `json:"accessToken"`
	UserID             string  `json:"userId"`
	UserEmail          string  `json:"userEmail"`
	FirstName          string  `json:"firstName"`
	LastName           string  `json:"lastName"`
	MustChangePassword bool    `json:"mustChangePassword"`
	ActiveSpaceID      *string `json:"activeSpaceId,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with a Nori server",
	Long:  "Log in to a Nori server by providing the server URL, email, and password. Credentials are stored in ~/.config/nori/credentials for use by subsequent CLI commands.",
	RunE:  runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	// Prompt for server URL
	serverURL, err := promptString(reader, "Server URL")
	if err != nil {
		return fmt.Errorf("failed to read server URL: %w", err)
	}
	serverURL = normalizeURL(serverURL)

	// Prompt for email
	email, err := promptString(reader, "Email")
	if err != nil {
		return fmt.Errorf("failed to read email: %w", err)
	}

	// Prompt for password (hidden)
	password, err := promptPassword("Password")
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	// Attempt login
	client := cli.NewClientWithURL(serverURL)
	resp, err := client.Post("/auth/login", map[string]string{
		"email":    email,
		"password": password,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		return fmt.Errorf("invalid email or password")
	}
	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("login failed: %s", errResp.Error)
		}
		return fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	var loginResp loginResponse
	if err := cli.ReadJSON(resp, &loginResp); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	// Handle mustChangePassword
	if loginResp.MustChangePassword {
		fmt.Println("You must change your password before continuing.")

		newToken, err := handlePasswordChange(reader, client, loginResp.AccessToken)
		if err != nil {
			return err
		}
		loginResp.AccessToken = newToken
	}

	// Save credentials
	creds := &cli.Credentials{
		ServerURL:   serverURL,
		AccessToken: loginResp.AccessToken,
		UserID:      loginResp.UserID,
		UserEmail:   loginResp.UserEmail,
	}
	if err := cli.SaveCredentials(creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Printf("Logged in as %s\n", loginResp.UserEmail)
	return nil
}

// handlePasswordChange prompts for new password and calls POST /auth/change-password.
// Returns the new access token on success.
func handlePasswordChange(reader *bufio.Reader, client *cli.Client, currentToken string) (string, error) {
	// Get the current password (user just typed it — ask again for change flow)
	currentPassword, err := promptPassword("Current password")
	if err != nil {
		return "", fmt.Errorf("failed to read current password: %w", err)
	}

	newPassword, err := promptPassword("New password")
	if err != nil {
		return "", fmt.Errorf("failed to read new password: %w", err)
	}

	confirmPassword, err := promptPassword("Confirm new password")
	if err != nil {
		return "", fmt.Errorf("failed to read password confirmation: %w", err)
	}

	if newPassword != confirmPassword {
		return "", fmt.Errorf("passwords do not match")
	}

	// Use the token from the login response for the authenticated request
	client.Token = currentToken

	resp, err := client.Post("/auth/change-password", map[string]string{
		"currentPassword": currentPassword,
		"newPassword":     newPassword,
	})
	if err != nil {
		return "", fmt.Errorf("failed to change password: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return "", fmt.Errorf("password change failed: %s", errResp.Error)
		}
		return "", fmt.Errorf("password change failed: unauthorized")
	}
	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return "", fmt.Errorf("password change failed: %s", errResp.Error)
		}
		return "", fmt.Errorf("password change failed with status %d", resp.StatusCode)
	}

	var changeResp loginResponse
	if err := cli.ReadJSON(resp, &changeResp); err != nil {
		return "", fmt.Errorf("failed to parse password change response: %w", err)
	}

	fmt.Println("Password changed successfully.")
	return changeResp.AccessToken, nil
}

// promptString prints a prompt and reads a line from the reader.
func promptString(reader *bufio.Reader, prompt string) (string, error) {
	fmt.Printf("%s: ", prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// promptPassword prints a prompt and reads a password without echoing.
func promptPassword(prompt string) (string, error) {
	fmt.Printf("%s: ", prompt)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // newline after hidden input
	if err != nil {
		return "", err
	}
	return string(password), nil
}

// normalizeURL ensures the server URL has a scheme and no trailing slash.
func normalizeURL(url string) string {
	url = strings.TrimSpace(url)
	url = strings.TrimRight(url, "/")
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}
	return url
}
