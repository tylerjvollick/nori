package formula

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// Formula file extensions. TOML is preferred, JSON is legacy fallback.
const (
	FormulaExtTOML = ".formula.toml"
	FormulaExtJSON = ".formula.json"
	FormulaExt     = FormulaExtJSON // Legacy alias for backwards compatibility
)

// Parser handles loading and resolving formulas.
//
// NOTE: Parser is NOT thread-safe. Create a new Parser per goroutine or
// synchronize access externally. The cache and resolving maps have no
// internal synchronization.
type Parser struct {
	// searchPaths are directories to search for formulas (in order).
	searchPaths []string

	// cache stores loaded formulas by name.
	cache map[string]*Formula

	// resolvingSet tracks formulas currently being resolved (for cycle detection).
	resolvingSet map[string]bool

	// resolvingChain tracks the order of formulas being resolved (for error messages).
	resolvingChain []string
}

// NewParser creates a new formula parser.
// searchPaths are directories to search for formulas when resolving extends.
// Default paths are: .beads/formulas, ~/.beads/formulas, $GT_ROOT/.beads/formulas
func NewParser(searchPaths ...string) *Parser {
	paths := searchPaths
	if len(paths) == 0 {
		paths = defaultSearchPaths()
	}
	return &Parser{
		searchPaths:    paths,
		cache:          make(map[string]*Formula),
		resolvingSet:   make(map[string]bool),
		resolvingChain: nil,
	}
}

// defaultSearchPaths returns the default formula search paths.
func defaultSearchPaths() []string {
	var paths []string

	// Project-level formulas
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, ".beads", "formulas"))
	}

	// User-level formulas
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".beads", "formulas"))
	}

	// Orchestrator formulas (via GT_ROOT)
	if gtRoot := os.Getenv("GT_ROOT"); gtRoot != "" {
		paths = append(paths, filepath.Join(gtRoot, ".beads", "formulas"))
	}

	return paths
}

// ParseFile parses a formula from a file path.
// Detects format from extension: .formula.toml or .formula.json
func (p *Parser) ParseFile(path string) (*Formula, error) {
	// Check cache first
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	if cached, ok := p.cache[absPath]; ok {
		return cached, nil
	}

	// Read and parse the file
	// #nosec G304 -- absPath comes from controlled search paths or explicit user input
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	// Detect format from extension
	var formula *Formula
	if strings.HasSuffix(path, FormulaExtTOML) {
		formula, err = p.ParseTOML(data)
	} else {
		formula, err = p.Parse(data)
	}
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	formula.Source = absPath

	// Set source tracing info on all steps (gt-8tmz.18)
	SetSourceInfo(formula)

	p.cache[absPath] = formula

	// Also cache by name for extends resolution
	p.cache[formula.Formula] = formula

	return formula, nil
}

// Parse parses a formula from JSON bytes.
func (p *Parser) Parse(data []byte) (*Formula, error) {
	var formula Formula
	if err := json.Unmarshal(data, &formula); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}

	// Set defaults
	if formula.Version == 0 {
		formula.Version = 1
	}
	if formula.Type == "" {
		formula.Type = TypeWorkflow
	}

	return &formula, nil
}

// ParseTOML parses a formula from TOML bytes.
func (p *Parser) ParseTOML(data []byte) (*Formula, error) {
	var formula Formula
	if err := toml.Unmarshal(data, &formula); err != nil {
		return nil, fmt.Errorf("toml: %w", err)
	}

	// Set defaults
	if formula.Version == 0 {
		formula.Version = 1
	}
	if formula.Type == "" {
		formula.Type = TypeWorkflow
	}

	return &formula, nil
}

// Resolve fully resolves a formula, processing extends and expansions.
// Returns a new formula with all inheritance applied.
func (p *Parser) Resolve(formula *Formula) (*Formula, error) {
	// Check for cycles
	if p.resolvingSet[formula.Formula] {
		// Build the cycle chain for a clear error message
		chain := append(p.resolvingChain, formula.Formula)
		return nil, fmt.Errorf("circular extends detected: %s", strings.Join(chain, " -> "))
	}
	p.resolvingSet[formula.Formula] = true
	p.resolvingChain = append(p.resolvingChain, formula.Formula)
	defer func() {
		delete(p.resolvingSet, formula.Formula)
		p.resolvingChain = p.resolvingChain[:len(p.resolvingChain)-1]
	}()

	// If no extends, just validate and return
	if len(formula.Extends) == 0 {
		if err := formula.Validate(); err != nil {
			return nil, err
		}
		return formula, nil
	}

	// Build merged formula from parents
	merged := &Formula{
		Formula:     formula.Formula,
		Description: formula.Description,
		Version:     formula.Version,
		Type:        formula.Type,
		Source:      formula.Source,
		Vars:        make(map[string]*VarDef),
		Steps:       nil,
		Compose:     nil,
	}

	// Apply each parent in order
	for _, parentName := range formula.Extends {
		parent, err := p.loadFormula(parentName)
		if err != nil {
			return nil, fmt.Errorf("extends %s: %w", parentName, err)
		}

		// Resolve parent recursively
		parent, err = p.Resolve(parent)
		if err != nil {
			return nil, fmt.Errorf("resolve parent %s: %w", parentName, err)
		}

		// Merge parent vars (parent vars are inherited, child overrides)
		for name, varDef := range parent.Vars {
			if _, exists := merged.Vars[name]; !exists {
				merged.Vars[name] = varDef
			}
		}

		// Merge parent steps (append, child steps come after)
		merged.Steps = append(merged.Steps, parent.Steps...)

		// Merge parent compose rules
		merged.Compose = mergeComposeRules(merged.Compose, parent.Compose)
	}

	// Apply child overrides
	for name, varDef := range formula.Vars {
		merged.Vars[name] = varDef
	}

	// Merge child steps: override parent steps by ID (preserving position),
	// append new child steps at the end.
	merged.Steps = mergeSteps(merged.Steps, formula.Steps)

	merged.Compose = mergeComposeRules(merged.Compose, formula.Compose)

	// Use child description if set
	if formula.Description != "" {
		merged.Description = formula.Description
	}

	if err := merged.Validate(); err != nil {
		return nil, err
	}

	return merged, nil
}

// loadFormula loads a formula by name from search paths.
// Tries TOML first (.formula.toml), then falls back to JSON (.formula.json).
func (p *Parser) loadFormula(name string) (*Formula, error) {
	// Check cache first
	if cached, ok := p.cache[name]; ok {
		return cached, nil
	}

	// Search for the formula file - try TOML first, then JSON
	extensions := []string{FormulaExtTOML, FormulaExtJSON}
	for _, dir := range p.searchPaths {
		for _, ext := range extensions {
			path := filepath.Join(dir, name+ext)
			if _, err := os.Stat(path); err == nil {
				return p.ParseFile(path)
			}
		}
	}

	return nil, fmt.Errorf("formula %q not found in search paths", name)
}

// LoadByName loads a formula by name from search paths.
// This is the public API for loading formulas used by expansion operators.
func (p *Parser) LoadByName(name string) (*Formula, error) {
	return p.loadFormula(name)
}

// mergeSteps merges child steps into parent steps.
// Child steps with the same ID as a parent step replace the parent step
// in-place (preserving position). Child steps with new IDs are appended.
func mergeSteps(parent, child []*Step) []*Step {
	// Index parent steps by ID for quick lookup
	parentIdx := make(map[string]int, len(parent))
	for i, s := range parent {
		parentIdx[s.ID] = i
	}

	// Copy parent steps (will be modified in-place for overrides)
	result := make([]*Step, len(parent))
	copy(result, parent)

	// Apply child steps
	for _, cs := range child {
		if idx, exists := parentIdx[cs.ID]; exists {
			// Override: replace parent step at same position
			result[idx] = cs
		} else {
			// New step: append at end
			result = append(result, cs)
		}
	}

	return result
}

// mergeComposeRules merges two compose rule sets.
func mergeComposeRules(base, overlay *ComposeRules) *ComposeRules {
	if overlay == nil {
		return base
	}
	if base == nil {
		return overlay
	}

	result := &ComposeRules{
		BondPoints: append([]*BondPoint{}, base.BondPoints...),
		Hooks:      append([]*Hook{}, base.Hooks...),
		Expand:     append([]*ExpandRule{}, base.Expand...),
		Map:        append([]*MapRule{}, base.Map...),
	}

	// Add overlay bond points (override by ID)
	existingBP := make(map[string]int)
	for i, bp := range result.BondPoints {
		existingBP[bp.ID] = i
	}
	for _, bp := range overlay.BondPoints {
		if idx, exists := existingBP[bp.ID]; exists {
			result.BondPoints[idx] = bp
		} else {
			result.BondPoints = append(result.BondPoints, bp)
		}
	}

	// Add overlay hooks (append, no override)
	result.Hooks = append(result.Hooks, overlay.Hooks...)

	// Add overlay expand rules (append, no override)
	result.Expand = append(result.Expand, overlay.Expand...)

	// Add overlay map rules (append, no override)
	result.Map = append(result.Map, overlay.Map...)

	return result
}

// varPattern matches {{variable}} placeholders.
var varPattern = regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*)\}\}`)

// tomlQuotedVarPattern matches TOML strings that are purely a single variable
// reference: "{{varname}}". Used by SubstituteTOML to unquote integer values.
var tomlQuotedVarPattern = regexp.MustCompile(`"(\{\{[a-zA-Z_][a-zA-Z0-9_]*\}\})"`)

// intPattern matches a pure integer string (optional leading minus).
var intPattern = regexp.MustCompile(`^-?[0-9]+$`)

// ExtractVariables finds all {{variable}} references in a formula.
func ExtractVariables(formula *Formula) []string {
	seen := make(map[string]bool)
	var vars []string

	// Helper to extract vars from a string
	extract := func(s string) {
		matches := varPattern.FindAllStringSubmatch(s, -1)
		for _, match := range matches {
			if len(match) >= 2 && !seen[match[1]] {
				seen[match[1]] = true
				vars = append(vars, match[1])
			}
		}
	}

	// Extract from formula fields
	extract(formula.Description)

	// Extract from steps
	var extractFromStep func(*Step)
	extractFromStep = func(step *Step) {
		extract(step.Title)
		extract(step.Description)
		extract(step.Assignee)
		extract(step.Condition)
		for _, child := range step.Children {
			extractFromStep(child)
		}
	}

	for _, step := range formula.Steps {
		extractFromStep(step)
	}

	return vars
}

// Substitute replaces {{variable}} placeholders with values.
func Substitute(s string, vars map[string]string) string {
	return varPattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name from {{name}}
		name := match[2 : len(match)-2]
		if val, ok := vars[name]; ok {
			return val
		}
		return match // Keep unresolved placeholders
	})
}

// SubstituteTOML replaces {{variable}} placeholders in raw TOML content,
// handling TOML type coercion. When a TOML string value is purely a single
// variable reference (e.g., count = "{{batch_size}}") and the resolved value
// is a pure integer, the quotes are stripped so TOML parses it as an integer
// (e.g., count = 6). This allows recipe authors to use template variables in
// fields that expect integers (like loop count).
func SubstituteTOML(content string, vars map[string]string) string {
	// First pass: unquote pure-variable TOML strings that resolve to integers.
	// "{{batch_size}}" -> 6  (when batch_size = "6")
	result := tomlQuotedVarPattern.ReplaceAllStringFunc(content, func(match string) string {
		// match is e.g., "{{batch_size}}" — extract the inner {{varname}}
		inner := match[1 : len(match)-1] // strip surrounding quotes
		varName := inner[2 : len(inner)-2]
		if val, ok := vars[varName]; ok && intPattern.MatchString(val) {
			return val // unquoted integer
		}
		// Not an integer or unknown var — substitute inside the quotes
		substituted := Substitute(inner, vars)
		return `"` + substituted + `"`
	})

	// Second pass: substitute remaining {{vars}} in all other string contexts.
	result = Substitute(result, vars)

	return result
}

// ValidateVars checks that all required variables are provided
// and all values pass their constraints.
func ValidateVars(formula *Formula, values map[string]string) error {
	var errs []string

	for name, def := range formula.Vars {
		val, provided := values[name]

		// Check required
		if def.Required && !provided {
			errs = append(errs, fmt.Sprintf("variable %q is required", name))
			continue
		}

		// Use default if not provided
		if !provided && def.Default != nil {
			val = *def.Default
		}

		// Skip further validation if no value
		if val == "" {
			continue
		}

		// Check enum constraint
		if len(def.Enum) > 0 {
			found := false
			for _, allowed := range def.Enum {
				if val == allowed {
					found = true
					break
				}
			}
			if !found {
				errs = append(errs, fmt.Sprintf("variable %q: value %q not in allowed values %v", name, val, def.Enum))
			}
		}

		// Check pattern constraint
		if def.Pattern != "" {
			re, err := regexp.Compile(def.Pattern)
			if err != nil {
				errs = append(errs, fmt.Sprintf("variable %q: invalid pattern %q: %v", name, def.Pattern, err))
			} else if !re.MatchString(val) {
				errs = append(errs, fmt.Sprintf("variable %q: value %q does not match pattern %q", name, val, def.Pattern))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("variable validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}

// ApplyDefaults returns a new map with default values filled in.
// PreParseVars extracts variable definitions from raw TOML content and merges
// them with user-provided overrides. This performs a lightweight parse of just
// the [vars] section — it does NOT parse steps or other fields — so it can be
// called before SubstituteTOML to build the variable map needed for TOML-level
// substitution (e.g., converting count = "{{batch_size}}" to count = 6).
func PreParseVars(tomlContent []byte, userVars map[string]string) map[string]string {
	// Lightweight struct that only decodes the vars section.
	var partial struct {
		Vars map[string]*VarDef `toml:"vars"`
	}
	// Ignore errors — if the TOML is malformed, we return what we can.
	_ = toml.Unmarshal(tomlContent, &partial)

	result := make(map[string]string)

	// Apply defaults from the TOML [vars] section.
	for name, def := range partial.Vars {
		if def != nil && def.Default != nil {
			result[name] = *def.Default
		}
	}

	// User overrides take precedence.
	for k, v := range userVars {
		result[k] = v
	}

	return result
}

func ApplyDefaults(formula *Formula, values map[string]string) map[string]string {
	result := make(map[string]string)

	// Copy provided values
	for k, v := range values {
		result[k] = v
	}

	// Apply defaults for missing values
	for name, def := range formula.Vars {
		if _, exists := result[name]; !exists && def.Default != nil {
			result[name] = *def.Default
		}
	}

	return result
}

// SetSourceInfo sets SourceFormula and SourceLocation on all steps in a formula.
// Called after parsing to enable source tracing during cooking (gt-8tmz.18).
func SetSourceInfo(formula *Formula) {
	setSourceInfoRecursive(formula.Steps, formula.Formula, "steps")
	// Also set source info on template steps for expansion formulas
	setSourceInfoRecursive(formula.Template, formula.Formula, "template")
}

// setSourceInfoRecursive recursively sets source info on steps.
func setSourceInfoRecursive(steps []*Step, formulaName, pathPrefix string) {
	for i, step := range steps {
		step.SourceFormula = formulaName
		step.SourceLocation = fmt.Sprintf("%s[%d]", pathPrefix, i)

		if len(step.Children) > 0 {
			childPath := fmt.Sprintf("%s[%d].children", pathPrefix, i)
			setSourceInfoRecursive(step.Children, formulaName, childPath)
		}

		// Handle loop body steps
		if step.Loop != nil && len(step.Loop.Body) > 0 {
			bodyPath := fmt.Sprintf("%s[%d].loop.body", pathPrefix, i)
			setSourceInfoRecursive(step.Loop.Body, formulaName, bodyPath)
		}
	}
}
