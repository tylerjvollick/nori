package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
)

func TestResolveSOPVersion_NilInputs(t *testing.T) {
	// When no explicit SOP and no default on the ticket type, result should be nil
	svc := &TicketSOPService{}
	ticketType := &models.TicketType{
		ID:   uuid.New(),
		Name: "Build",
	}

	result, err := svc.ResolveSOPVersion(nil, ticketType)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result when no SOP template is set, got %+v", result)
	}
}

func TestValidateParentChild_NilParent(t *testing.T) {
	parent := &models.Ticket{
		ID:             uuid.New(),
		TicketNumber:   "BUILD-001",
		ParentTicketID: nil, // no parent — valid
	}

	if err := ValidateParentChild(parent); err != nil {
		t.Errorf("expected no error for a ticket without a parent, got: %v", err)
	}
}

func TestValidateParentChild_AlreadyHasParent(t *testing.T) {
	grandparentID := uuid.New()
	parent := &models.Ticket{
		ID:             uuid.New(),
		TicketNumber:   "BUILD-002",
		ParentTicketID: &grandparentID, // already a child — nesting not allowed
	}

	err := ValidateParentChild(parent)
	if err == nil {
		t.Fatal("expected error for a ticket that already has a parent, got nil")
	}

	expected := `cannot nest tickets more than one level: parent ticket BUILD-002 already has a parent`
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestSOPCopyDescription(t *testing.T) {
	tests := []struct {
		name      string
		stepCount int
		expected  string
	}{
		{
			name:      "single step",
			stepCount: 1,
			expected:  "Copied 1 step from linked SOP",
		},
		{
			name:      "multiple steps",
			stepCount: 5,
			expected:  "Copied 5 steps from linked SOP",
		},
		{
			name:      "zero steps",
			stepCount: 0,
			expected:  "Copied 0 steps from linked SOP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sopCopyDescription(tt.stepCount)
			if got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
			}
		})
	}
}
