// Package formula provides batch size resolution for steps.
//
// ResolveBatchSizes walks the step graph and sets BatchSize on every step.
// Steps with explicit batch_size keep their value. Steps without batch_size
// inherit from their dependency chain (Needs + DependsOn): if all deps agree,
// use that value; if deps disagree or have no deps, use the recipe-level default.
package formula

// ResolveBatchSizes resolves the effective BatchSize for every step in the
// given slice. It walks the dependency graph (Needs + DependsOn) and applies
// inheritance:
//
//   - Steps with an explicit BatchSize keep their value unchanged.
//   - Steps whose dependencies all share the same resolved BatchSize inherit
//     that value.
//   - Steps whose dependencies disagree, or that have no dependencies, receive
//     the defaultBatchSize.
//
// Children are processed recursively; the same rules apply within child step
// hierarchies.
func ResolveBatchSizes(steps []*Step, defaultBatchSize int) {
	// Build a flat map of step ID -> *Step for dependency lookup.
	stepMap := buildStepMap(steps)

	// Resolved tracks which step IDs have already been resolved, preventing
	// infinite loops in cyclic graphs (which shouldn't occur but defensive).
	resolved := make(map[string]bool)

	// Resolve every step, walking deps first (topological-ish via recursion).
	for _, step := range steps {
		resolveBatchSize(step, stepMap, resolved, defaultBatchSize)
	}
}

// resolveBatchSize resolves a single step's BatchSize. It recurses into
// dependencies first so their BatchSize values are available for inheritance.
func resolveBatchSize(step *Step, stepMap map[string]*Step, resolved map[string]bool, defaultBatchSize int) {
	if resolved[step.ID] {
		return
	}
	resolved[step.ID] = true

	// Recurse into children first — they form their own sub-graph.
	for _, child := range step.Children {
		resolveBatchSize(child, stepMap, resolved, defaultBatchSize)
	}

	// If this step already has an explicit BatchSize, keep it.
	if step.BatchSize != nil {
		return
	}

	// Collect all dependency IDs (Needs + DependsOn).
	allDeps := make([]string, 0, len(step.Needs)+len(step.DependsOn))
	allDeps = append(allDeps, step.Needs...)
	allDeps = append(allDeps, step.DependsOn...)

	if len(allDeps) == 0 {
		// No dependencies — use the recipe-level default.
		bs := defaultBatchSize
		step.BatchSize = &bs
		return
	}

	// Resolve all deps first so their BatchSize values are populated.
	for _, depID := range allDeps {
		if dep, ok := stepMap[depID]; ok {
			resolveBatchSize(dep, stepMap, resolved, defaultBatchSize)
		}
	}

	// Check if all deps agree on a single BatchSize value.
	var agreed *int
	unanimous := true

	for _, depID := range allDeps {
		dep, ok := stepMap[depID]
		if !ok {
			// Unknown dep — can't inherit from it.
			unanimous = false
			break
		}
		depBS := dep.BatchSize
		if depBS == nil {
			// Dep didn't resolve (shouldn't happen, but be safe).
			unanimous = false
			break
		}
		if agreed == nil {
			agreed = depBS
		} else if *agreed != *depBS {
			unanimous = false
			break
		}
	}

	if unanimous && agreed != nil {
		bs := *agreed
		step.BatchSize = &bs
	} else {
		bs := defaultBatchSize
		step.BatchSize = &bs
	}
}
