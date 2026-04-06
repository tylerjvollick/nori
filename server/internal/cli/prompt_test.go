package cli

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptString(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("hello world\n"))
	result, err := PromptString(reader, "Test")
	require.NoError(t, err)
	assert.Equal(t, "hello world", result)
}

func TestPromptString_Trimmed(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("  spaces  \n"))
	result, err := PromptString(reader, "Test")
	require.NoError(t, err)
	assert.Equal(t, "spaces", result)
}

func TestPromptStringDefault_UsesDefault(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	result, err := PromptStringDefault(reader, "Test", "default-val")
	require.NoError(t, err)
	assert.Equal(t, "default-val", result)
}

func TestPromptStringDefault_UsesInput(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("custom\n"))
	result, err := PromptStringDefault(reader, "Test", "default-val")
	require.NoError(t, err)
	assert.Equal(t, "custom", result)
}

func TestPromptConfirm_Yes(t *testing.T) {
	tests := []string{"y\n", "Y\n", "yes\n", "YES\n", "Yes\n"}
	for _, input := range tests {
		reader := bufio.NewReader(strings.NewReader(input))
		result, err := PromptConfirm(reader, "Continue?")
		require.NoError(t, err)
		assert.True(t, result, "expected true for input %q", input)
	}
}

func TestPromptConfirm_No(t *testing.T) {
	tests := []string{"n\n", "N\n", "no\n", "\n", "anything\n"}
	for _, input := range tests {
		reader := bufio.NewReader(strings.NewReader(input))
		result, err := PromptConfirm(reader, "Continue?")
		require.NoError(t, err)
		assert.False(t, result, "expected false for input %q", input)
	}
}

func TestPromptInt_Default(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	result, err := PromptInt(reader, "WIP limit", 3)
	require.NoError(t, err)
	assert.Equal(t, 3, result)
}

func TestPromptInt_CustomValue(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("5\n"))
	result, err := PromptInt(reader, "WIP limit", 1)
	require.NoError(t, err)
	assert.Equal(t, 5, result)
}

func TestPromptInt_InvalidInput(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("abc\n"))
	_, err := PromptInt(reader, "WIP limit", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid number")
}
