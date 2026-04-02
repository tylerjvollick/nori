package services

import (
	"testing"
)

func TestStepActivityDescription(t *testing.T) {
	tests := []struct {
		name      string
		action    string
		stepTitle string
		expected  string
	}{
		{
			name:      "started",
			action:    "started",
			stepTitle: "Cut Mortises",
			expected:  `Step "Cut Mortises" started`,
		},
		{
			name:      "paused",
			action:    "paused",
			stepTitle: "Apply Finish",
			expected:  `Step "Apply Finish" paused`,
		},
		{
			name:      "resumed",
			action:    "resumed",
			stepTitle: "Sand to 220",
			expected:  `Step "Sand to 220" resumed`,
		},
		{
			name:      "completed",
			action:    "completed",
			stepTitle: "Glue Up",
			expected:  `Step "Glue Up" completed`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stepActivityDescription(tt.action, tt.stepTitle)
			if got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
			}
		})
	}
}
