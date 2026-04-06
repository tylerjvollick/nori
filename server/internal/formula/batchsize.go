// Package formula provides batch size resolution for steps.
//
// ResolveBatchSizes walks the step graph and sets BatchSize on every step.
// Steps with an explicit batch_size keep their value. Steps without batch_size
// receive the recipe-level default. There is no dependency-based inheritance:
// if a step needs to be per-piece, the recipe author must set batch_size = 1
// explicitly on that step.
package formula

// ResolveBatchSizes resolves the effective BatchSize for every step in the
// given slice.
//
//   - Steps with an explicit BatchSize keep their value unchanged.
//   - Steps without an explicit BatchSize receive the defaultBatchSize.
//
// Children are processed recursively; the same rules apply within child step
// hierarchies.
func ResolveBatchSizes(steps []*Step, defaultBatchSize int) {
	for _, step := range steps {
		resolveBatchSize(step, defaultBatchSize)
	}
}

// resolveBatchSize resolves a single step's BatchSize. If the step has an
// explicit value it is kept; otherwise it receives the recipe-level default.
func resolveBatchSize(step *Step, defaultBatchSize int) {
	// Recurse into children first — they form their own sub-graph.
	for _, child := range step.Children {
		resolveBatchSize(child, defaultBatchSize)
	}

	// If this step already has an explicit BatchSize, keep it.
	if step.BatchSize != nil {
		return
	}

	// No explicit value — use the recipe-level default.
	bs := defaultBatchSize
	step.BatchSize = &bs
}
