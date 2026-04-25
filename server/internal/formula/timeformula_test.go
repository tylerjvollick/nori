package formula

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTimeString(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		wantErr  bool
	}{
		{"5m", 300, false},
		{"30m", 1800, false},
		{"1h", 3600, false},
		{"2.5h", 9000, false},
		{"30s", 30, false},
		{"300", 300, false},
		{"0", 0, false},
		{"1.5m", 90, false},
		{"", 0, true},
		{"abc", 0, true},
		{"5x", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseTimeString(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestEvalTimeFormula(t *testing.T) {
	f := func(s string) *string { return &s }

	tests := []struct {
		name      string
		formula   *string
		batchSize int
		expected  *int
		wantErr   bool
	}{
		{
			name:     "nil formula",
			formula:  nil,
			expected: nil,
		},
		{
			name:     "empty formula",
			formula:  f(""),
			expected: nil,
		},
		{
			name:      "flat time with equals sign",
			formula:   f("=30m"),
			batchSize: 6,
			expected:  timeIntPtr(1800),
		},
		{
			name:      "flat time plain seconds",
			formula:   f("=300"),
			batchSize: 10,
			expected:  timeIntPtr(300),
		},
		{
			name:      "plain seconds no equals",
			formula:   f("300"),
			batchSize: 1,
			expected:  timeIntPtr(300),
		},
		{
			name:      "variable formula",
			formula:   f("5m * {{batch_size}}"),
			batchSize: 6,
			expected:  timeIntPtr(1800),
		},
		{
			name:      "setup plus variable",
			formula:   f("30m + (5m * {{batch_size}})"),
			batchSize: 2,
			expected:  timeIntPtr(2400), // 1800 + 600
		},
		{
			name:      "setup plus variable larger batch",
			formula:   f("30m + (5m * {{batch_size}})"),
			batchSize: 6,
			expected:  timeIntPtr(3600), // 1800 + 1800
		},
		{
			name:      "hours in formula",
			formula:   f("1h + (10m * {{batch_size}})"),
			batchSize: 3,
			expected:  timeIntPtr(5400), // 3600 + 1800
		},
		{
			name:      "batch size 1",
			formula:   f("5m * {{batch_size}}"),
			batchSize: 1,
			expected:  timeIntPtr(300),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvalTimeFormula(tt.formula, tt.batchSize)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.expected == nil {
					assert.Nil(t, result)
				} else {
					require.NotNil(t, result)
					assert.Equal(t, *tt.expected, *result)
				}
			}
		})
	}
}

func timeIntPtr(n int) *int {
	return &n
}
