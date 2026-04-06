package formula

import (
	"testing"
)

func intPtr(v int) *int {
	return &v
}

func TestResolveBatchSizes(t *testing.T) {
	tests := []struct {
		name             string
		steps            []*Step
		defaultBatchSize int
		// Map of step ID -> expected BatchSize after resolution
		wantBatchSizes map[string]int
	}{
		{
			name: "explicit override kept",
			steps: []*Step{
				{ID: "cut", BatchSize: intPtr(10)},
				{ID: "sand", BatchSize: intPtr(5)},
			},
			defaultBatchSize: 1,
			wantBatchSizes: map[string]int{
				"cut":  10,
				"sand": 5,
			},
		},
		{
			name: "no explicit batch_size uses default",
			steps: []*Step{
				{ID: "cut"},
				{ID: "sand"},
			},
			defaultBatchSize: 6,
			wantBatchSizes: map[string]int{
				"cut":  6,
				"sand": 6,
			},
		},
		{
			name: "deps do not inherit - step gets default not dep value",
			steps: []*Step{
				{ID: "cut", BatchSize: intPtr(4)},
				{ID: "sand", Needs: []string{"cut"}},
			},
			defaultBatchSize: 1,
			wantBatchSizes: map[string]int{
				"cut":  4,
				"sand": 1, // no inheritance — gets default
			},
		},
		{
			name: "deps do not inherit via DependsOn either",
			steps: []*Step{
				{ID: "cut", BatchSize: intPtr(4)},
				{ID: "sand", DependsOn: []string{"cut"}},
			},
			defaultBatchSize: 1,
			wantBatchSizes: map[string]int{
				"cut":  4,
				"sand": 1, // no inheritance — gets default
			},
		},
		{
			name: "multi deps - step gets default regardless",
			steps: []*Step{
				{ID: "cut", BatchSize: intPtr(6)},
				{ID: "mill", BatchSize: intPtr(6)},
				{ID: "sand", Needs: []string{"cut", "mill"}},
			},
			defaultBatchSize: 1,
			wantBatchSizes: map[string]int{
				"cut":  6,
				"mill": 6,
				"sand": 1, // no inheritance — gets default
			},
		},
		{
			name: "chain - no propagation through deps",
			steps: []*Step{
				{ID: "A", BatchSize: intPtr(8)},
				{ID: "B", Needs: []string{"A"}},
				{ID: "C", Needs: []string{"B"}},
			},
			defaultBatchSize: 1,
			wantBatchSizes: map[string]int{
				"A": 8,
				"B": 1, // no inheritance
				"C": 1, // no inheritance
			},
		},
		{
			name: "explicit overrides kept in chain",
			steps: []*Step{
				{ID: "cut", BatchSize: intPtr(8)},
				{ID: "sand", Needs: []string{"cut"}, BatchSize: intPtr(2)},
				{ID: "finish", Needs: []string{"sand"}},
			},
			defaultBatchSize: 1,
			wantBatchSizes: map[string]int{
				"cut":    8,
				"sand":   2,
				"finish": 1, // no inheritance — gets default
			},
		},
		{
			name: "children are resolved recursively",
			steps: []*Step{
				{
					ID:        "parent",
					BatchSize: intPtr(5),
					Children: []*Step{
						{ID: "child1"},
						{ID: "child2", Needs: []string{"child1"}},
					},
				},
			},
			defaultBatchSize: 1,
			wantBatchSizes: map[string]int{
				"parent": 5,
				"child1": 1, // no deps, no explicit -> default
				"child2": 1, // dep on child1 doesn't matter -> default
			},
		},
		{
			name: "children with explicit batch sizes",
			steps: []*Step{
				{
					ID: "parent",
					Children: []*Step{
						{ID: "child1", BatchSize: intPtr(3)},
						{ID: "child2", Needs: []string{"child1"}},
					},
				},
			},
			defaultBatchSize: 1,
			wantBatchSizes: map[string]int{
				"parent": 1, // no explicit -> default
				"child1": 3, // explicit
				"child2": 1, // no inheritance from child1 -> default
			},
		},
		{
			name: "milestone gets default like any other step",
			steps: []*Step{
				{ID: "assemble", BatchSize: intPtr(1)},
				{ID: "done", Type: "milestone", Needs: []string{"assemble"}},
			},
			defaultBatchSize: 4,
			wantBatchSizes: map[string]int{
				"assemble": 1,
				"done":     4, // milestone gets default, not inherited 1
			},
		},
		{
			name: "diamond dependency - all get default",
			steps: []*Step{
				{ID: "A", BatchSize: intPtr(4)},
				{ID: "B", Needs: []string{"A"}},
				{ID: "C", Needs: []string{"A"}},
				{ID: "D", Needs: []string{"B", "C"}},
			},
			defaultBatchSize: 1,
			wantBatchSizes: map[string]int{
				"A": 4,
				"B": 1, // default
				"C": 1, // default
				"D": 1, // default
			},
		},
		{
			name: "real world: per-piece glue-up does not infect spray-finish",
			steps: []*Step{
				{ID: "glue-up", BatchSize: intPtr(1)},
				{ID: "spray-finish", Needs: []string{"glue-up"}},
				{ID: "install-seat", Needs: []string{"spray-finish"}},
			},
			defaultBatchSize: 4,
			wantBatchSizes: map[string]int{
				"glue-up":      1,
				"spray-finish": 4, // gets recipe default, not inherited 1
				"install-seat": 4, // gets recipe default
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResolveBatchSizes(tt.steps, tt.defaultBatchSize)

			// Build a flat map of all steps for checking
			allSteps := buildStepMap(tt.steps)

			for id, wantBS := range tt.wantBatchSizes {
				step, ok := allSteps[id]
				if !ok {
					t.Errorf("step %q not found after resolution", id)
					continue
				}
				if step.BatchSize == nil {
					t.Errorf("step %q: BatchSize is nil, want %d", id, wantBS)
					continue
				}
				if *step.BatchSize != wantBS {
					t.Errorf("step %q: BatchSize = %d, want %d", id, *step.BatchSize, wantBS)
				}
			}

			// Verify every step has a non-nil BatchSize after resolution
			for id, step := range allSteps {
				if step.BatchSize == nil {
					t.Errorf("step %q: BatchSize is nil after resolution (should always be set)", id)
				}
			}
		})
	}
}
