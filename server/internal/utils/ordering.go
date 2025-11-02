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

// GenerateOrderBetween generates a lexicographic order string between two existing orders
// If before is empty, generates an order before after
// If after is empty, generates an order after before
// If both are empty, generates a default starting order
func GenerateOrderBetween(before, after string) string {
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
func generateBefore(s string) string {
	if len(s) == 0 {
		return string(minChar)
	}

	firstChar := rune(s[0])

	// If it's a digit, we can decrement within digit range
	if firstChar >= '0' && firstChar <= '9' {
		if firstChar > '0' {
			return string(firstChar - 1)
		}
		// Already at '0', prepend and add middle character
		return string('0') + string(midChar)
	}

	// If it's a lowercase letter
	if firstChar > minChar {
		// Can decrement the first character
		return string(firstChar - 1)
	}

	// First char is already 'a' (minChar), use digit '9' which comes before 'a' in PostgreSQL collation
	return string('9')
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
func generateBetween(before, after string) string {
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
			result.WriteRune(beforeChar + diff/2)
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
		// Allow digits (0-9) and lowercase letters (a-z)
		if (char < '0' || char > '9') && (char < minChar || char > maxChar) {
			return fmt.Errorf("order contains invalid character: %c (must be 0-9 or %c-%c)", char, minChar, maxChar)
		}
	}

	return nil
}
