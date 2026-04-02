package services

import (
	"testing"
)

func TestIsValidTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     bool
	}{
		{name: "woodworking_shop", template: "woodworking_shop", want: true},
		{name: "sales", template: "sales", want: true},
		{name: "unknown", template: "metalwork", want: false},
		{name: "empty", template: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidTemplate(tt.template)
			if got != tt.want {
				t.Errorf("IsValidTemplate(%q) = %v, want %v", tt.template, got, tt.want)
			}
		})
	}
}

func TestSalesTicketTypeSeedData(t *testing.T) {
	// Verify the seed data is correctly structured
	if len(salesTicketTypes) != 1 {
		t.Fatalf("expected 1 ticket type, got %d", len(salesTicketTypes))
	}

	expected := []struct {
		name          string
		statusCount   int
		defaultStatus string
		terminalNames []string
	}{
		{name: "Lead", statusCount: 6, defaultStatus: "New", terminalNames: []string{"Won", "Lost"}},
	}

	for i, exp := range expected {
		tt := salesTicketTypes[i]
		if tt.Name != exp.name {
			t.Errorf("ticket type %d: expected name %q, got %q", i, exp.name, tt.Name)
		}
		if len(tt.Statuses) != exp.statusCount {
			t.Errorf("ticket type %q: expected %d statuses, got %d", tt.Name, exp.statusCount, len(tt.Statuses))
		}

		// Verify exactly one default status
		defaultCount := 0
		terminalCount := 0
		terminalNames := map[string]bool{}
		for _, s := range tt.Statuses {
			if s.IsDefault {
				defaultCount++
				if s.Name != exp.defaultStatus {
					t.Errorf("ticket type %q: expected default status %q, got %q", tt.Name, exp.defaultStatus, s.Name)
				}
			}
			if s.IsTerminal {
				terminalCount++
				terminalNames[s.Name] = true
			}
		}

		if defaultCount != 1 {
			t.Errorf("ticket type %q: expected 1 default status, got %d", tt.Name, defaultCount)
		}
		if terminalCount != len(exp.terminalNames) {
			t.Errorf("ticket type %q: expected %d terminal statuses, got %d", tt.Name, len(exp.terminalNames), terminalCount)
		}
		for _, tn := range exp.terminalNames {
			if !terminalNames[tn] {
				t.Errorf("ticket type %q: expected terminal status %q not found", tt.Name, tn)
			}
		}
	}
}

func TestWoodworkingShopSOPCategorySeedData(t *testing.T) {
	expectedNames := []string{"Techniques", "Maintenance", "Setup", "Products"}

	if len(woodworkingShopSOPCategories) != len(expectedNames) {
		t.Fatalf("expected %d SOP categories, got %d", len(expectedNames), len(woodworkingShopSOPCategories))
	}

	for i, expected := range expectedNames {
		if woodworkingShopSOPCategories[i] != expected {
			t.Errorf("SOP category %d: expected %q, got %q", i, expected, woodworkingShopSOPCategories[i])
		}
	}
}

func TestWoodworkingShopTicketTypeSeedData(t *testing.T) {
	// Verify the seed data is correctly structured
	if len(woodworkingShopTicketTypes) != 3 {
		t.Fatalf("expected 3 ticket types, got %d", len(woodworkingShopTicketTypes))
	}

	expected := []struct {
		name           string
		statusCount    int
		defaultStatus  string
		terminalStatus string
	}{
		{name: "Build", statusCount: 4, defaultStatus: "Queued", terminalStatus: "Done"},
		{name: "Maintenance", statusCount: 3, defaultStatus: "Open", terminalStatus: "Done"},
		{name: "Prep", statusCount: 3, defaultStatus: "Todo", terminalStatus: "Done"},
	}

	for i, exp := range expected {
		tt := woodworkingShopTicketTypes[i]
		if tt.Name != exp.name {
			t.Errorf("ticket type %d: expected name %q, got %q", i, exp.name, tt.Name)
		}
		if len(tt.Statuses) != exp.statusCount {
			t.Errorf("ticket type %q: expected %d statuses, got %d", tt.Name, exp.statusCount, len(tt.Statuses))
		}

		// Verify exactly one default status
		defaultCount := 0
		terminalCount := 0
		for _, s := range tt.Statuses {
			if s.IsDefault {
				defaultCount++
				if s.Name != exp.defaultStatus {
					t.Errorf("ticket type %q: expected default status %q, got %q", tt.Name, exp.defaultStatus, s.Name)
				}
			}
			if s.IsTerminal {
				terminalCount++
				if s.Name != exp.terminalStatus {
					t.Errorf("ticket type %q: expected terminal status %q, got %q", tt.Name, exp.terminalStatus, s.Name)
				}
			}
		}

		if defaultCount != 1 {
			t.Errorf("ticket type %q: expected 1 default status, got %d", tt.Name, defaultCount)
		}
		if terminalCount < 1 {
			t.Errorf("ticket type %q: expected at least 1 terminal status, got %d", tt.Name, terminalCount)
		}
	}
}
