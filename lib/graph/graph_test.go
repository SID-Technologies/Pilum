package graph

import (
	"testing"
)

func TestTopologicalSort(t *testing.T) {
	tests := []struct {
		name      string
		nodes     map[string][]string // node -> dependencies
		wantOrder []string            // expected order (dependencies before dependents)
		wantErr   bool
	}{
		{
			name: "simple chain A -> B -> C",
			nodes: map[string][]string{
				"C": {"B"},
				"B": {"A"},
				"A": {},
			},
			wantOrder: []string{"A", "B", "C"},
			wantErr:   false,
		},
		{
			name: "diamond A -> B,C -> D",
			nodes: map[string][]string{
				"D": {"B", "C"},
				"B": {"A"},
				"C": {"A"},
				"A": {},
			},
			wantOrder: nil, // A must come before B and C, which must come before D
			wantErr:   false,
		},
		{
			name: "no dependencies",
			nodes: map[string][]string{
				"A": {},
				"B": {},
				"C": {},
			},
			wantOrder: nil, // any order is valid
			wantErr:   false,
		},
		{
			name: "circular dependency",
			nodes: map[string][]string{
				"A": {"B"},
				"B": {"C"},
				"C": {"A"},
			},
			wantOrder: nil,
			wantErr:   true,
		},
		{
			name: "self-referencing",
			nodes: map[string][]string{
				"A": {"A"},
			},
			wantOrder: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New()
			for name, deps := range tt.nodes {
				g.AddNode(name, deps)
			}

			order, err := g.TopologicalSort()

			if tt.wantErr {
				if err == nil {
					t.Errorf("TopologicalSort() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("TopologicalSort() unexpected error: %v", err)
				return
			}

			if len(order) != len(tt.nodes) {
				t.Errorf("TopologicalSort() returned %d nodes, want %d", len(order), len(tt.nodes))
				return
			}

			// Verify order is valid (all dependencies come before dependents)
			position := make(map[string]int)
			for i, name := range order {
				position[name] = i
			}

			for name, deps := range tt.nodes {
				for _, dep := range deps {
					if position[dep] >= position[name] {
						t.Errorf("TopologicalSort() invalid order: %s (pos %d) should come before %s (pos %d)",
							dep, position[dep], name, position[name])
					}
				}
			}

			// If specific order is expected, verify it
			if tt.wantOrder != nil {
				for i, want := range tt.wantOrder {
					if order[i] != want {
						t.Errorf("TopologicalSort()[%d] = %s, want %s", i, order[i], want)
					}
				}
			}
		})
	}
}

func TestGetDependents(t *testing.T) {
	g := New()
	g.AddNode("A", []string{})
	g.AddNode("B", []string{"A"})
	g.AddNode("C", []string{"A"})
	g.AddNode("D", []string{"B", "C"})
	g.AddNode("E", []string{"D"})

	tests := []struct {
		name string
		node string
		want []string
	}{
		{
			name: "A has all as dependents",
			node: "A",
			want: []string{"B", "C", "D", "E"},
		},
		{
			name: "B has D and E as dependents",
			node: "B",
			want: []string{"D", "E"},
		},
		{
			name: "E has no dependents",
			node: "E",
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.GetDependents(tt.node)

			if len(got) != len(tt.want) {
				t.Errorf("GetDependents(%s) = %v, want %v", tt.node, got, tt.want)
				return
			}

			// Check all expected dependents are present
			gotSet := make(map[string]bool)
			for _, n := range got {
				gotSet[n] = true
			}
			for _, w := range tt.want {
				if !gotSet[w] {
					t.Errorf("GetDependents(%s) missing %s", tt.node, w)
				}
			}
		})
	}
}

func TestPropagateChanges(t *testing.T) {
	g := New()
	g.AddNode("shared-lib", []string{})
	g.AddNode("auth-api", []string{"shared-lib"})
	g.AddNode("user-api", []string{"shared-lib"})
	g.AddNode("gateway", []string{"auth-api", "user-api"})

	tests := []struct {
		name    string
		changed map[string]bool
		want    map[string]bool
	}{
		{
			name:    "shared-lib changes propagate to all",
			changed: map[string]bool{"shared-lib": true},
			want: map[string]bool{
				"shared-lib": true,
				"auth-api":   true,
				"user-api":   true,
				"gateway":    true,
			},
		},
		{
			name:    "auth-api changes propagate to gateway",
			changed: map[string]bool{"auth-api": true},
			want: map[string]bool{
				"auth-api": true,
				"gateway":  true,
			},
		},
		{
			name:    "gateway changes don't propagate",
			changed: map[string]bool{"gateway": true},
			want: map[string]bool{
				"gateway": true,
			},
		},
		{
			name:    "multiple changes",
			changed: map[string]bool{"auth-api": true, "user-api": true},
			want: map[string]bool{
				"auth-api": true,
				"user-api": true,
				"gateway":  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.PropagateChanges(tt.changed)

			if len(got) != len(tt.want) {
				t.Errorf("PropagateChanges() = %v, want %v", got, tt.want)
				return
			}

			for name := range tt.want {
				if !got[name] {
					t.Errorf("PropagateChanges() missing %s", name)
				}
			}
		})
	}
}

func TestValidateDependencies(t *testing.T) {
	tests := []struct {
		name    string
		nodes   map[string][]string
		wantErr bool
	}{
		{
			name: "valid dependencies",
			nodes: map[string][]string{
				"A": {},
				"B": {"A"},
				"C": {"A", "B"},
			},
			wantErr: false,
		},
		{
			name: "missing dependency",
			nodes: map[string][]string{
				"A": {},
				"B": {"A", "missing"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New()
			for name, deps := range tt.nodes {
				g.AddNode(name, deps)
			}

			err := g.ValidateDependencies()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDependencies() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCalculateDepths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		nodes     map[string][]string
		wantDepth map[string]int
		wantErr   bool
	}{
		{
			name: "simple chain A->B->C",
			nodes: map[string][]string{
				"A": {},
				"B": {"A"},
				"C": {"B"},
			},
			wantDepth: map[string]int{"A": 0, "B": 1, "C": 2},
		},
		{
			name: "diamond A->B,C->D",
			nodes: map[string][]string{
				"A": {},
				"B": {"A"},
				"C": {"A"},
				"D": {"B", "C"},
			},
			wantDepth: map[string]int{"A": 0, "B": 1, "C": 1, "D": 2},
		},
		{
			name: "no dependencies",
			nodes: map[string][]string{
				"A": {},
				"B": {},
				"C": {},
			},
			wantDepth: map[string]int{"A": 0, "B": 0, "C": 0},
		},
		{
			name: "complex multi-level",
			nodes: map[string][]string{
				"A": {},
				"B": {"A"},
				"C": {"A"},
				"D": {"B"},
				"E": {"C", "D"},
			},
			wantDepth: map[string]int{"A": 0, "B": 1, "C": 1, "D": 2, "E": 3},
		},
		{
			name: "asymmetric diamond",
			nodes: map[string][]string{
				"A": {},
				"B": {"A"},
				"C": {"B"},
				"D": {"A", "C"},
			},
			// D depends on A (depth 0) and C (depth 2), so D = max(0,2)+1 = 3
			wantDepth: map[string]int{"A": 0, "B": 1, "C": 2, "D": 3},
		},
		{
			name: "wide tree all depend on root",
			nodes: map[string][]string{
				"root": {},
				"a":    {"root"},
				"b":    {"root"},
				"c":    {"root"},
				"d":    {"root"},
			},
			wantDepth: map[string]int{"root": 0, "a": 1, "b": 1, "c": 1, "d": 1},
		},
		{
			name: "deep chain 5 levels",
			nodes: map[string][]string{
				"L0": {},
				"L1": {"L0"},
				"L2": {"L1"},
				"L3": {"L2"},
				"L4": {"L3"},
			},
			wantDepth: map[string]int{"L0": 0, "L1": 1, "L2": 2, "L3": 3, "L4": 4},
		},
		{
			name: "single node",
			nodes: map[string][]string{
				"only": {},
			},
			wantDepth: map[string]int{"only": 0},
		},
		{
			name: "circular dependency",
			nodes: map[string][]string{
				"A": {"B"},
				"B": {"C"},
				"C": {"A"},
			},
			wantErr: true,
		},
		{
			name: "external deps not in graph treated as depth 0",
			nodes: map[string][]string{
				"A": {"external-service"},
				"B": {"A"},
			},
			// "external-service" is not in graph, so A's dep on it is ignored → A depth 0
			wantDepth: map[string]int{"A": 0, "B": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := New()
			for name, deps := range tt.nodes {
				g.AddNode(name, deps)
			}

			depths, err := g.CalculateDepths()

			if tt.wantErr {
				if err == nil {
					t.Errorf("CalculateDepths() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("CalculateDepths() unexpected error: %v", err)
				return
			}

			if len(depths) != len(tt.wantDepth) {
				t.Errorf("CalculateDepths() returned %d depths, want %d", len(depths), len(tt.wantDepth))
				return
			}

			for name, wantD := range tt.wantDepth {
				if gotD, ok := depths[name]; !ok {
					t.Errorf("CalculateDepths() missing depth for %s", name)
				} else if gotD != wantD {
					t.Errorf("CalculateDepths()[%s] = %d, want %d", name, gotD, wantD)
				}
			}
		})
	}
}

func TestNodeCount(t *testing.T) {
	g := New()
	if g.NodeCount() != 0 {
		t.Errorf("NodeCount() on empty graph = %d, want 0", g.NodeCount())
	}

	g.AddNode("A", nil)
	g.AddNode("B", []string{"A"})

	if g.NodeCount() != 2 {
		t.Errorf("NodeCount() = %d, want 2", g.NodeCount())
	}
}

func TestHasNode(t *testing.T) {
	g := New()
	g.AddNode("A", nil)

	if !g.HasNode("A") {
		t.Error("HasNode(A) = false, want true")
	}
	if g.HasNode("B") {
		t.Error("HasNode(B) = true, want false")
	}
}
