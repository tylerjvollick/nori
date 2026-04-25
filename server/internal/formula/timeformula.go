package formula

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// timeUnitPattern matches a number (int or float) followed by a time unit suffix.
var timeUnitPattern = regexp.MustCompile(`^([0-9]*\.?[0-9]+)\s*(s|m|h)$`)

// batchSizePattern matches {{batch_size}} in a formula string.
var batchSizePattern = regexp.MustCompile(`\{\{batch_size\}\}`)

// ParseTimeString converts a human-readable duration string to seconds.
// Supported formats: "30s" (seconds), "5m" (minutes), "2.5h" (hours), "300" (plain seconds).
func ParseTimeString(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty time string")
	}

	// Plain integer (seconds).
	if n, err := strconv.Atoi(s); err == nil {
		return n, nil
	}

	m := timeUnitPattern.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid time string %q", s)
	}

	val, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in time string %q: %w", s, err)
	}

	switch m[2] {
	case "s":
		return int(math.Round(val)), nil
	case "m":
		return int(math.Round(val * 60)), nil
	case "h":
		return int(math.Round(val * 3600)), nil
	default:
		return 0, fmt.Errorf("unknown time unit %q", m[2])
	}
}

// EvalTimeFormula evaluates a time estimation formula string against a given
// batch size, returning the result in seconds.
//
// Supported formula formats:
//   - "=30m"                         → flat time (constant regardless of batch)
//   - "5m * {{batch_size}}"          → variable time
//   - "30m + (5m * {{batch_size}})"  → setup + variable
//   - "300"                          → plain seconds (flat)
//   - "=300"                         → plain seconds (flat)
//
// Returns nil for a nil or empty formula.
func EvalTimeFormula(formula *string, batchSize int) (*int, error) {
	if formula == nil {
		return nil, nil
	}
	f := strings.TrimSpace(*formula)
	if f == "" {
		return nil, nil
	}

	// Strip leading '=' for flat-time shorthand.
	if strings.HasPrefix(f, "=") {
		f = strings.TrimSpace(f[1:])
	}

	// Substitute {{batch_size}} with the actual value.
	f = batchSizePattern.ReplaceAllString(f, strconv.Itoa(batchSize))

	result, err := evalTimeExpr(f)
	if err != nil {
		return nil, fmt.Errorf("evaluating time formula %q: %w", *formula, err)
	}
	return &result, nil
}

// evalTimeExpr evaluates a simple arithmetic expression with time strings.
// Supports: addition (+), multiplication (*), and parentheses.
// Operates left-to-right respecting standard precedence (* before +).
func evalTimeExpr(expr string) (int, error) {
	p := &timeExprParser{input: strings.TrimSpace(expr), pos: 0}
	result, err := p.parseAddSub()
	if err != nil {
		return 0, err
	}
	if p.pos < len(p.input) {
		return 0, fmt.Errorf("unexpected character at position %d: %q", p.pos, string(p.input[p.pos]))
	}
	return result, nil
}

type timeExprParser struct {
	input string
	pos   int
}

func (p *timeExprParser) skipWhitespace() {
	for p.pos < len(p.input) && (p.input[p.pos] == ' ' || p.input[p.pos] == '\t') {
		p.pos++
	}
}

// parseAddSub handles + and - (lowest precedence).
func (p *timeExprParser) parseAddSub() (int, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return 0, err
	}

	for {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			break
		}
		op := p.input[p.pos]
		if op != '+' && op != '-' {
			break
		}
		p.pos++
		right, err := p.parseMulDiv()
		if err != nil {
			return 0, err
		}
		if op == '+' {
			left += right
		} else {
			left -= right
		}
	}
	return left, nil
}

// parseMulDiv handles * and / (higher precedence).
func (p *timeExprParser) parseMulDiv() (int, error) {
	left, err := p.parseAtom()
	if err != nil {
		return 0, err
	}

	for {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			break
		}
		op := p.input[p.pos]
		if op != '*' && op != '/' {
			break
		}
		p.pos++
		right, err := p.parseAtom()
		if err != nil {
			return 0, err
		}
		if op == '*' {
			left *= right
		} else {
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			left /= right
		}
	}
	return left, nil
}

// parseAtom handles parentheses, time strings, and plain numbers.
func (p *timeExprParser) parseAtom() (int, error) {
	p.skipWhitespace()

	if p.pos >= len(p.input) {
		return 0, fmt.Errorf("unexpected end of expression")
	}

	// Parenthesized sub-expression.
	if p.input[p.pos] == '(' {
		p.pos++ // consume '('
		val, err := p.parseAddSub()
		if err != nil {
			return 0, err
		}
		p.skipWhitespace()
		if p.pos >= len(p.input) || p.input[p.pos] != ')' {
			return 0, fmt.Errorf("expected ')' at position %d", p.pos)
		}
		p.pos++ // consume ')'
		return val, nil
	}

	// Read a token: number (possibly with decimal) optionally followed by a time unit.
	start := p.pos
	for p.pos < len(p.input) {
		c := p.input[p.pos]
		if c >= '0' && c <= '9' || c == '.' {
			p.pos++
		} else {
			break
		}
	}

	// Check for time unit suffix (s, m, h) immediately after the number.
	if p.pos < len(p.input) {
		c := p.input[p.pos]
		if c == 's' || c == 'm' || c == 'h' {
			p.pos++
		}
	}

	token := p.input[start:p.pos]
	if token == "" {
		return 0, fmt.Errorf("expected number or time string at position %d", p.pos)
	}

	return ParseTimeString(token)
}
