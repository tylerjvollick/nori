package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// PromptString prints a prompt and reads a line from the reader.
func PromptString(reader *bufio.Reader, prompt string) (string, error) {
	fmt.Printf("%s: ", prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// PromptStringDefault prints a prompt with a default value and reads a line.
// If the user presses Enter without typing, the default is returned.
func PromptStringDefault(reader *bufio.Reader, prompt, defaultVal string) (string, error) {
	fmt.Printf("%s [%s]: ", prompt, defaultVal)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	val := strings.TrimSpace(input)
	if val == "" {
		return defaultVal, nil
	}
	return val, nil
}

// PromptPassword prints a prompt and reads a password without echoing.
func PromptPassword(prompt string) (string, error) {
	fmt.Printf("%s: ", prompt)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // newline after hidden input
	if err != nil {
		return "", err
	}
	return string(password), nil
}

// PromptConfirm prints a yes/no prompt and returns true for "y" or "yes".
func PromptConfirm(reader *bufio.Reader, prompt string) (bool, error) {
	fmt.Printf("%s [y/N]: ", prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	val := strings.ToLower(strings.TrimSpace(input))
	return val == "y" || val == "yes", nil
}

// PromptInt prints a prompt and reads an integer from the reader.
// Returns the default value if the user presses Enter without typing.
func PromptInt(reader *bufio.Reader, prompt string, defaultVal int) (int, error) {
	fmt.Printf("%s [%d]: ", prompt, defaultVal)
	input, err := reader.ReadString('\n')
	if err != nil {
		return 0, err
	}
	val := strings.TrimSpace(input)
	if val == "" {
		return defaultVal, nil
	}
	var n int
	if _, err := fmt.Sscanf(val, "%d", &n); err != nil {
		return 0, fmt.Errorf("invalid number: %s", val)
	}
	return n, nil
}
