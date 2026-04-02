package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tylerjvollick/nori/internal/models"
)

func TestComputeBreakdown(t *testing.T) {
	ticketID := uuid.New()

	tests := []struct {
		name             string
		entries          []models.CostEntry
		expectedTotal    string
		expectedLabor    string
		expectedMaterial string
		expectedOther    string
	}{
		{
			name:             "no entries",
			entries:          []models.CostEntry{},
			expectedTotal:    "0",
			expectedLabor:    "0",
			expectedMaterial: "0",
			expectedOther:    "0",
		},
		{
			name: "labor only",
			entries: []models.CostEntry{
				{TicketID: ticketID, CostType: models.CostTypeLabor, Amount: decimal.NewFromFloat(50.00)},
				{TicketID: ticketID, CostType: models.CostTypeLabor, Amount: decimal.NewFromFloat(75.50)},
			},
			expectedTotal:    "125.5",
			expectedLabor:    "125.5",
			expectedMaterial: "0",
			expectedOther:    "0",
		},
		{
			name: "material only",
			entries: []models.CostEntry{
				{TicketID: ticketID, CostType: models.CostTypeMaterial, Amount: decimal.NewFromFloat(30.00)},
			},
			expectedTotal:    "30",
			expectedLabor:    "0",
			expectedMaterial: "30",
			expectedOther:    "0",
		},
		{
			name: "mixed types",
			entries: []models.CostEntry{
				{TicketID: ticketID, CostType: models.CostTypeLabor, Amount: decimal.NewFromFloat(100.00)},
				{TicketID: ticketID, CostType: models.CostTypeMaterial, Amount: decimal.NewFromFloat(50.00)},
				{TicketID: ticketID, CostType: models.CostTypeConsumable, Amount: decimal.NewFromFloat(10.00)},
				{TicketID: ticketID, CostType: models.CostTypeMarketing, Amount: decimal.NewFromFloat(25.00)},
				{TicketID: ticketID, CostType: models.CostTypeOther, Amount: decimal.NewFromFloat(5.00)},
			},
			expectedTotal:    "190",
			expectedLabor:    "100",
			expectedMaterial: "50",
			expectedOther:    "40",
		},
		{
			name: "consumable and marketing roll up to other",
			entries: []models.CostEntry{
				{TicketID: ticketID, CostType: models.CostTypeConsumable, Amount: decimal.NewFromFloat(15.00)},
				{TicketID: ticketID, CostType: models.CostTypeMarketing, Amount: decimal.NewFromFloat(20.00)},
				{TicketID: ticketID, CostType: models.CostTypeOther, Amount: decimal.NewFromFloat(5.00)},
			},
			expectedTotal:    "40",
			expectedLabor:    "0",
			expectedMaterial: "0",
			expectedOther:    "40",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			breakdown := computeBreakdown(ticketID, tt.entries)

			if breakdown.TotalCost.String() != tt.expectedTotal {
				t.Errorf("TotalCost: expected %s, got %s", tt.expectedTotal, breakdown.TotalCost.String())
			}
			if breakdown.LaborCost.String() != tt.expectedLabor {
				t.Errorf("LaborCost: expected %s, got %s", tt.expectedLabor, breakdown.LaborCost.String())
			}
			if breakdown.MaterialCost.String() != tt.expectedMaterial {
				t.Errorf("MaterialCost: expected %s, got %s", tt.expectedMaterial, breakdown.MaterialCost.String())
			}
			if breakdown.OtherCost.String() != tt.expectedOther {
				t.Errorf("OtherCost: expected %s, got %s", tt.expectedOther, breakdown.OtherCost.String())
			}
			if len(breakdown.Entries) != len(tt.entries) {
				t.Errorf("Entries count: expected %d, got %d", len(tt.entries), len(breakdown.Entries))
			}
		})
	}
}

func TestLaborCostCalculation(t *testing.T) {
	// Verify the math: 3600 seconds (1 hour) at $50/hr = $50
	laborRate := decimal.NewFromFloat(50.00)
	secondsPerHour := decimal.NewFromInt(3600)
	actualSeconds := 3600

	hours := decimal.NewFromInt(int64(actualSeconds)).Div(secondsPerHour)
	amount := hours.Mul(laborRate)

	expectedHours := "1"
	if hours.String() != expectedHours {
		t.Errorf("Expected hours %s, got %s", expectedHours, hours.String())
	}

	expectedAmount := "50"
	if amount.String() != expectedAmount {
		t.Errorf("Expected amount %s, got %s", expectedAmount, amount.String())
	}

	// Verify partial hours: 1800 seconds (0.5 hours) at $50/hr = $25
	actualSeconds = 1800
	hours = decimal.NewFromInt(int64(actualSeconds)).Div(secondsPerHour)
	amount = hours.Mul(laborRate)

	expectedAmount = "25"
	if amount.String() != expectedAmount {
		t.Errorf("Expected amount %s, got %s", expectedAmount, amount.String())
	}

	// Verify arbitrary time: 5400 seconds (1.5 hours) at $45/hr = $67.50
	laborRate = decimal.NewFromFloat(45.00)
	actualSeconds = 5400
	hours = decimal.NewFromInt(int64(actualSeconds)).Div(secondsPerHour)
	amount = hours.Mul(laborRate)

	expectedAmount = "67.5"
	if amount.String() != expectedAmount {
		t.Errorf("Expected amount %s, got %s", expectedAmount, amount.String())
	}
}

func TestMaterialCostCalculation(t *testing.T) {
	// Verify: 10.5 board feet at $8.50/bf = $89.25
	quantity := decimal.NewFromFloat(10.5)
	unitCost := decimal.NewFromFloat(8.50)
	amount := quantity.Mul(unitCost)

	expectedAmount := "89.25"
	if amount.String() != expectedAmount {
		t.Errorf("Expected amount %s, got %s", expectedAmount, amount.String())
	}

	// Verify: 4 each at $12.99/each = $51.96
	quantity = decimal.NewFromFloat(4)
	unitCost = decimal.NewFromFloat(12.99)
	amount = quantity.Mul(unitCost)

	expectedAmount = "51.96"
	if amount.String() != expectedAmount {
		t.Errorf("Expected amount %s, got %s", expectedAmount, amount.String())
	}
}
