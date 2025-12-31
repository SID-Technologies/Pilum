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
