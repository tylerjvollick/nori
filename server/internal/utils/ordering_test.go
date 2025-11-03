package utils

import (
	"testing"
)

// Test edge case: inserting before step with order 'a'
func TestGenerateOrderBetween_BeforeFirstA(t *testing.T) {
	// Simulates dragging a step to the beginning when first step has order 'a'
	result := GenerateOrderBetween("", "a")

	// Should return empty string to signal rebalancing needed
	if result != "" {
		t.Errorf("Expected empty string for edge case, got: %s", result)
	}
}

// Test edge case: inserting before step with order 'aam'
func TestGenerateOrderBetween_BeforeAAM(t *testing.T) {
	// Simulates dragging a step to the beginning when first step has order 'aam'
	result := GenerateOrderBetween("", "aam")

	// Should be able to generate 'aal' which comes before 'aam'
	if result == "" {
		t.Errorf("Expected non-empty string, got empty")
	}
	if result >= "aam" {
		t.Errorf("Expected order < 'aam', got: %s", result)
	}
}

// Test edge case: inserting before step with order 'aaa'
func TestGenerateOrderBetween_BeforeAAA(t *testing.T) {
	// Simulates dragging a step to the beginning when first step has order 'aaa'
	result := GenerateOrderBetween("", "aaa")

	// All characters are 'a' - should return empty string to signal rebalancing needed
	if result != "" {
		t.Errorf("Expected empty string for edge case (all 'a's), got: %s", result)
	}
}

// Test edge case: inserting after step with order 'z'
func TestGenerateOrderBetween_AfterLastZ(t *testing.T) {
	// Simulates dragging a step to the end when last step has order 'z'
	result := GenerateOrderBetween("z", "")

	// Should return 'zm' (can always append to 'z')
	if result == "" {
		t.Errorf("Expected non-empty string, got empty")
	}
	if result <= "z" {
		t.Errorf("Expected order > 'z', got: %s", result)
	}
}

// Test normal case: inserting between two steps
func TestGenerateOrderBetween_Normal(t *testing.T) {
	tests := []struct {
		before   string
		after    string
		expected string
	}{
		{"a", "c", "b"},  // Middle of range
		{"a", "b", "am"}, // Adjacent
		{"", "b", "a"},   // At beginning
		{"y", "", "z"},   // At end
		{"", "", "m"},    // Empty list
		{"a", "z", "m"},  // Large gap
	}

	for _, tt := range tests {
		result := GenerateOrderBetween(tt.before, tt.after)
		if result != tt.expected {
			t.Errorf("GenerateOrderBetween(%q, %q) = %q, expected %q",
				tt.before, tt.after, result, tt.expected)
		}
	}
}

// Test normalization of legacy orders
func TestNormalizeOrder(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a", "a"},   // Valid lowercase
		{"z", "z"},   // Valid lowercase
		{"0", "a"},   // Digit 0 -> 'a'
		{"9", "j"},   // Digit 9 -> 'j'
		{"M", "m"},   // Uppercase -> lowercase
		{"", ""},     // Empty preserved
		{"9a", "ja"}, // Mixed digit and letter
	}

	for _, tt := range tests {
		result := normalizeOrder(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeOrder(%q) = %q, expected %q",
				tt.input, result, tt.expected)
		}
	}
}
