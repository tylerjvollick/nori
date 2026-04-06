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
			name: "no deps uses default",
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
			name: "single dep inheritance via Needs",
			steps: []*Step{
				{ID: "cut", BatchSize: intPtr(4)},
				{ID: "sand", Needs: []string{"cut"}},
			},
			defaultBatchSize: 1,
			wantBatchSizes: map[string]int{
				"cut":  4,
				"sand": 4,
			},
		},
		{
			name: "single dep inheritance via DependsOn",
			steps: []*Step{
				{ID: "cut", BatchSize: intPtr(4)},
				{ID: "sand", DependsOn: []string{"cut"}},
			},
			defaultBatchSize: 1,
			wantBatchSizes: map[string]int{
				"cut":  4,
				"sand": 4,
			},
		},
		{
			name: "multi dep agreement inherits",
			steps: []*Step{
				{ID: "cut", BatchSize: intPtr(6)},
				{ID: "mill", BatchSize: intPtr(6)},
				{ID: "sand", Needs: []string{"cut", "mill"}},
			},
			defaultBatchSize: 1,
			wantBatchSizes: map[string]int{
				"cut":  6,
				"mill": 6,
				"sand": 6,
			},
		},
		{
			name: "multi dep disagreement falls back to default",
			steps: []*Step{
				{ID: "cut", BatchSize: intPtr(6)},
				{ID: "mill", BatchSize: intPtr(3)},
				{ID: "sand", Needs: []string{"cut", "mill"}},
			},
			defaultBatchSize: 1,
			wantBatchSizes: map[string]int{
				"cut":  6,
				"mill": 3,
				"sand": 1,
			},
		},
		{
			name: "chain inheritance - A -> B -> C",
			steps: []*Step{
				{ID: "A", BatchSize: intPtr(8)},
				{ID: "B", Needs: []string{"A"}},
				{ID: "C", Needs: []string{"B"}},
			},
			defaultBatchSize: 1,
			wantBatchSizes: map[string]int{
				"A": 8,
				"B": 8,
				"C": 8,
			},
		},
		{
			name: "mixed Needs and DependsOn",
			steps: []*Step{
				{ID: "cut", BatchSize: intPtr(4)},
				{ID: "sand", Needs: []string{"cut"}},
				{ID: "finish", DependsOn: []string{"sand"}},
			},
			defaultBatchSize: 1,
			wantBatchSizes: map[string]int{
				"cut":    4,
				"sand":   4,
				"finish": 4,
			},
		},
		{
			name: "explicit overrides break inheritance chain",
			steps: []*Step{
				{ID: "cut", BatchSize: intPtr(8)},
				{ID: "sand", Needs: []string{"cut"}, BatchSize: intPtr(2)},
				{ID: "finish", Needs: []string{"sand"}},
			},
			defaultBatchSize: 1,
			wantBatchSizes: map[string]int{
				"cut":    8,
				"sand":   2,
				"finish": 2,
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
				"child1": 1, // no deps -> default
				"child2": 1, // dep on child1 which got default
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
				"parent": 1, // no deps -> default
				"child1": 3,
				"child2": 3, // inherits from child1
			},
		},
		{
			name: "dep on unknown step falls back to default",
			steps: []*Step{
				{ID: "sand", Needs: []string{"nonexistent"}},
			},
			defaultBatchSize: 7,
			wantBatchSizes: map[string]int{
				"sand": 7,
			},
		},
		{
			name: "diamond dependency with agreement",
			steps: []*Step{
				{ID: "A", BatchSize: intPtr(4)},
				{ID: "B", Needs: []string{"A"}},
				{ID: "C", Needs: []string{"A"}},
				{ID: "D", Needs: []string{"B", "C"}},
			},
			defaultBatchSize: 1,
			wantBatchSizes: map[string]int{
				"A": 4,
				"B": 4,
				"C": 4,
				"D": 4,
			},
		},
		{
			name: "diamond dependency with disagreement",
			steps: []*Step{
				{ID: "A", BatchSize: intPtr(4)},
				{ID: "B", Needs: []string{"A"}, BatchSize: intPtr(2)},
				{ID: "C", Needs: []string{"A"}}, // inherits 4 from A
				{ID: "D", Needs: []string{"B", "C"}},
			},
			defaultBatchSize: 1,
			wantBatchSizes: map[string]int{
				"A": 4,
				"B": 2,
				"C": 4,
				"D": 1, // B=2, C=4 disagree -> default
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
