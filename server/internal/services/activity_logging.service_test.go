package services

import (
	"testing"
)

func TestTicketCreatedDescription(t *testing.T) {
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
			got := ticketCreatedDescription(tt.ticketNumber, tt.hasSOP)
			if got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestStatusChangeDesc(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := statusChangeDesc(tt.oldName, tt.newName)
			if got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestStepActivityDesc(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stepActivityDesc(tt.action, tt.stepTitle)
			if got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestAssignmentChangeDescription(t *testing.T) {
	tests := []struct {
		name        string
		oldAssignee string
		newAssignee string
		expected    string
	}{
		{
			name:        "new assignment",
			oldAssignee: "",
			newAssignee: "Tyler",
			expected:    "Assigned to Tyler",
		},
		{
			name:        "unassignment",
			oldAssignee: "Tyler",
			newAssignee: "",
			expected:    "Unassigned from Tyler",
		},
		{
			name:        "reassignment",
			oldAssignee: "Tyler",
			newAssignee: "Sarah",
			expected:    "Reassigned from Tyler to Sarah",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := assignmentChangeDescription(tt.oldAssignee, tt.newAssignee)
			if got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestCommentDescription(t *testing.T) {
	tests := []struct {
		name     string
		comment  string
		expected string
	}{
		{
			name:     "short comment",
			comment:  "Looks good",
			expected: "Looks good",
		},
		{
			name:     "long comment is truncated",
			comment:  "This is a very long comment that exceeds one hundred characters and should be truncated with an ellipsis at the end to keep descriptions concise",
			expected: "This is a very long comment that exceeds one hundred characters and should be truncated with an elli...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commentDescription(tt.comment)
			if got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestLinkAddedDescription(t *testing.T) {
	got := linkAddedDescription("blocks")
	expected := "Link added: blocks"
	if got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}
}

func TestCostLoggedDescription(t *testing.T) {
	tests := []struct {
		name        string
		costType    string
		amountCents int
		expected    string
	}{
		{
			name:        "labor cost",
			costType:    "Labor",
			amountCents: 5000,
			expected:    "Labor cost logged: $50.00",
		},
		{
			name:        "material cost",
			costType:    "Material",
			amountCents: 1299,
			expected:    "Material cost logged: $12.99",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := costLoggedDescription(tt.costType, tt.amountCents)
			if got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestInterruptionDescription(t *testing.T) {
	tenMin := 600
	tests := []struct {
		name            string
		stepTitle       string
		durationSeconds *int
		expected        string
	}{
		{
			name:            "without duration",
			stepTitle:       "Cut Mortises",
			durationSeconds: nil,
			expected:        `Step "Cut Mortises" paused for interruption`,
		},
		{
			name:            "with duration",
			stepTitle:       "Cut Mortises",
			durationSeconds: &tenMin,
			expected:        `Step "Cut Mortises" paused for interruption (10 min)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := interruptionDescription(tt.stepTitle, tt.durationSeconds)
			if got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
			}
		})
	}
}
