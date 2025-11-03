package utils

import (
	"fmt"
	"strings"
)

const (
	// Base characters for lexicographic ordering
	minChar = 'a'
	maxChar = 'z'
	midChar = 'm' // Roughly middle of the alphabet
)

// normalizeOrder converts any invalid order strings to use only lowercase letters
// This handles legacy orders that may contain digits or other characters
// IMPORTANT: Empty strings are returned as-is (they have special meaning for begin/end positioning)
func normalizeOrder(s string) string {
	// Empty strings indicate beginning/end positions - preserve them!
	if s == "" {
		return ""
	}

	// Check if string contains only valid lowercase letters
	valid := true
	for _, char := range s {
		if char < minChar || char > maxChar {
			valid = false
			break
		}
	}

	if valid {
		return s
	}

	// Invalid order detected - convert to valid characters
	// Digits sort before letters in ASCII, so convert them
	var result strings.Builder
	for _, char := range s {
		if char >= '0' && char <= '9' {
			// Map digits 0-9 to letters a-j
			result.WriteRune(minChar + (char - '0'))
		} else if char >= minChar && char <= maxChar {
			result.WriteRune(char)
		} else if char >= 'A' && char <= 'Z' {
			// Convert uppercase to lowercase
			result.WriteRune(char + 32)
		} else {
			// Invalid character, use midChar
			result.WriteRune(midChar)
		}
	}

	if result.Len() == 0 {
		return string(midChar)
	}

	return result.String()
}

// GenerateOrderBetween generates a lexicographic order string between two existing orders
// If before is empty, generates an order before after
// If after is empty, generates an order after before
// If both are empty, generates a default starting order
func GenerateOrderBetween(before, after string) string {
	// Normalize inputs to handle legacy/invalid orders
	before = normalizeOrder(before)
	after = normalizeOrder(after)

	// Initial case - no items exist yet
	if before == "" && after == "" {
		return string(midChar)
	}

	// Inserting at the beginning
	if before == "" {
		return generateBefore(after)
	}

	// Inserting at the end
	if after == "" {
		return generateAfter(before)
	}

	// Inserting between two items
	return generateBetween(before, after)
}

// generateBefore creates an order string that comes before the given string
// Returns empty string if we can't generate a value before 'a' (edge case requiring rebalance)
// Note: input should already be normalized by caller
func generateBefore(s string) string {
	if len(s) == 0 {
		return string(minChar)
	}

	firstChar := rune(s[0])

	// If it's a lowercase letter and greater than 'a'
	if firstChar > minChar && firstChar <= maxChar {
		// Can decrement the first character
		return string(firstChar - 1)
	}

	// First char is already 'a' (minChar)
	if firstChar == minChar {
		// For single 'a', we can't go lower - need rebalancing
		if len(s) == 1 {
			return ""
		}

		// For multi-character strings starting with 'a' (like "aam", "aaa", etc.)
		// Try to find a character we can decrement
		for i := 1; i < len(s); i++ {
			char := rune(s[i])
			if char > minChar {
				// Found a character we can decrement
				// Build: prefix + decremented char
				return s[:i] + string(char-1)
			}
		}

		// All characters are 'a' (like "aaa") - can't go lower, need rebalancing
		return ""
	}

	// Character is less than 'a' or greater than 'z' - shouldn't happen after normalization
	// Use fractional approach as fallback
	return string(minChar) + string(minChar) + string(midChar)
}

// generateAfter creates an order string that comes after the given string
func generateAfter(s string) string {
	if len(s) == 0 {
		return string(maxChar)
	}

	lastChar := rune(s[len(s)-1])
	if lastChar < maxChar {
		// Can increment the last character
		return s[:len(s)-1] + string(lastChar+1)
	}

	// Last char is 'z', need to append
	return s + string(midChar)
}

// generateBetween creates an order string between two existing strings
// Note: inputs should already be normalized by caller
func generateBetween(before, after string) string {
	// Ensure inputs are properly ordered
	if before >= after {
		// Invalid input - before should be less than after
		// Return a value that would sort between them by appending
		return before + string(minChar)
	}

	// Make both strings the same length by padding with 'a'
	maxLen := len(before)
	if len(after) > maxLen {
		maxLen = len(after)
	}

	beforePadded := padRight(before, maxLen, minChar)
	afterPadded := padRight(after, maxLen, minChar)

	// Find the first position where they differ
	var result strings.Builder
	for i := 0; i < maxLen; i++ {
		beforeChar := rune(beforePadded[i])
		afterChar := rune(afterPadded[i])

		if beforeChar == afterChar {
			result.WriteRune(beforeChar)
			continue
		}

		// Found difference - try to insert between
		diff := afterChar - beforeChar
		if diff > 1 {
			// Can insert a character between them
			midPoint := beforeChar + diff/2
			// Ensure we stay within lowercase letter range
			if midPoint >= minChar && midPoint <= maxChar {
				result.WriteRune(midPoint)
			} else {
				// Fallback: use adjacent approach
				result.WriteRune(beforeChar)
				result.WriteRune(midChar)
			}
			break
		}

		// Adjacent characters, need to go deeper
		result.WriteRune(beforeChar)
		// Append midpoint character
		result.WriteRune(midChar)
		break
	}

	// Handle edge case where we got through entire loop
	if result.Len() == 0 {
		return before + string(midChar)
	}

	return result.String()
}

// padRight pads a string to the specified length with the given character
func padRight(s string, length int, pad rune) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(string(pad), length-len(s))
}

// RebalanceOrders regenerates order strings for a list of items to keep them simple
// Useful when order strings get too long after many reorderings
// Returns a map of index -> new order value
func RebalanceOrders(count int) map[int]string {
	result := make(map[int]string)

	if count == 0 {
		return result
	}

	// Generate evenly spaced order strings
	for i := 0; i < count; i++ {
		result[i] = generateOrderForIndex(i, count)
	}

	return result
}

// generateOrderForIndex generates an order string for a given index in a total count
func generateOrderForIndex(index, total int) string {
	if total <= 26 {
		// Simple case - single character
		return string(minChar + rune(index))
	}

	// For larger counts, use multiple characters
	// This is a simple implementation - could be optimized for large lists
	base := 26
	result := ""
	n := index

	for {
		result = string(minChar+rune(n%base)) + result
		n = n / base
		if n == 0 {
			break
		}
		n-- // Adjust for base-26 without zero
	}

	return result
}

// ShouldRebalance checks if order strings are getting too long and should be rebalanced
func ShouldRebalance(orders []string) bool {
	const maxReasonableLength = 10

	for _, order := range orders {
		if len(order) > maxReasonableLength {
			return true
		}
	}

	return false
}

// ValidateOrder checks if an order string is valid
func ValidateOrder(order string) error {
	if len(order) == 0 {
		return fmt.Errorf("order cannot be empty")
	}

	for _, char := range order {
		// Only allow lowercase letters (a-z)
		// Digits are no longer supported as they cause sorting issues
		if char < minChar || char > maxChar {
			return fmt.Errorf("order contains invalid character: %c (must be %c-%c)", char, minChar, maxChar)
		}
	}

	return nil
}
