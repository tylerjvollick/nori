package services

import (
	"testing"
)

func TestStatusChangeDescription(t *testing.T) {
	tests := []struct {
		name     string
		oldName  string
		newName  string
		expected string
	}{
		{
			name:     "simple transition",
			oldName:  "Queued",
			newName:  "In Progress",
			expected: `Status changed from "Queued" to "In Progress"`,
		},
		{
			name:     "terminal transition",
			oldName:  "QC",
			newName:  "Done",
			expected: `Status changed from "QC" to "Done"`,
		},
		{
			name:     "reverse transition",
			oldName:  "Done",
			newName:  "In Progress",
			expected: `Status changed from "Done" to "In Progress"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The service formats the description using fmt.Sprintf with %q
			got := statusChangeDescription(tt.oldName, tt.newName)
			if got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestTicketCreationDescription(t *testing.T) {
	tests := []struct {
		name         string
		ticketNumber string
		hasSOP       bool
		expected     string
	}{
		{
			name:         "without SOP",
			ticketNumber: "BUILD-001",
			hasSOP:       false,
			expected:     "Ticket BUILD-001 created",
		},
		{
			name:         "with SOP",
			ticketNumber: "BUILD-042",
			hasSOP:       true,
			expected:     "Ticket BUILD-042 created with linked SOP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ticketCreationDescription(tt.ticketNumber, tt.hasSOP)
			if got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
			}
		})
	}
}
