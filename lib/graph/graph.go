package graph

import (
	"github.com/sid-technologies/pilum/lib/errors"
)

// Node represents a node in the dependency graph.
type Node struct {
	Name      string
	DependsOn []string
}

// Graph represents a directed dependency graph.
type Graph struct {
	nodes        map[string]*Node
	edges        map[string][]string // dependency -> dependents
	reverseEdges map[string][]string // dependent -> dependencies
}

// New creates a new dependency graph.
func New() *Graph {
	return &Graph{
		nodes:        make(map[string]*Node),
		edges:        make(map[string][]string),
		reverseEdges: make(map[string][]string),
	}
}

// AddNode adds a node to the graph with its dependencies.
func (g *Graph) AddNode(name string, dependsOn []string) {
	g.nodes[name] = &Node{
		Name:      name,
		DependsOn: dependsOn,
	}

	// Build edges: for each dependency, this node is a dependent
	for _, dep := range dependsOn {
		g.edges[dep] = append(g.edges[dep], name)
		g.reverseEdges[name] = append(g.reverseEdges[name], dep)
	}
}

// TopologicalSort returns nodes in dependency order (dependencies first).
// Returns an error if a circular dependency is detected.
func (g *Graph) TopologicalSort() ([]string, error) {
	// Calculate in-degree, counting only dependencies that are actually in this
	// graph. A dependency outside it is satisfied by definition -- it is not
	// part of this run -- and matches how CalculateDepths already treats them.
	// Counting it here would leave the node permanently above zero in-degree
	// and report a deliberate subset deploy as a circular dependency.
	inDegree := make(map[string]int)

	for name := range g.nodes {
		degree := 0

		for _, dep := range g.nodes[name].DependsOn {
			if _, exists := g.nodes[dep]; exists {
				degree++
			}
		}

		inDegree[name] = degree
	}

	// Find nodes with no dependencies (in-degree 0)
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		// Dequeue
		name := queue[0]
		queue = queue[1:]
		sorted = append(sorted, name)

		// For each dependent of this node, reduce in-degree
		for _, dependent := range g.edges[name] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// If we didn't process all nodes, there's a cycle
	if len(sorted) != len(g.nodes) {
		// Find nodes involved in the cycle
		var cycleNodes []string
		for name, degree := range inDegree {
			if degree > 0 {
				cycleNodes = append(cycleNodes, name)
			}
		}
		return nil, errors.New("circular dependency detected involving: %v", cycleNodes)
	}

	return sorted, nil
}

// GetDependents returns all nodes that depend on the given node (direct and transitive).
func (g *Graph) GetDependents(name string) []string {
	visited := make(map[string]bool)
	var dependents []string

	var visit func(n string)
	visit = func(n string) {
		for _, dependent := range g.edges[n] {
			if !visited[dependent] {
				visited[dependent] = true
				dependents = append(dependents, dependent)
				visit(dependent)
			}
		}
	}

	visit(name)
	return dependents
}

// GetDependencies returns all nodes that the given node depends on (direct and transitive).
func (g *Graph) GetDependencies(name string) []string {
	visited := make(map[string]bool)
	var dependencies []string

	var visit func(n string)
	visit = func(n string) {
		for _, dep := range g.reverseEdges[n] {
			if !visited[dep] {
				visited[dep] = true
				dependencies = append(dependencies, dep)
				visit(dep)
			}
		}
	}

	visit(name)
	return dependencies
}

// PropagateChanges takes a set of changed nodes and returns all nodes that
// should be considered changed (including transitive dependents).
func (g *Graph) PropagateChanges(changed map[string]bool) map[string]bool {
	result := make(map[string]bool)

	// Copy original changed nodes
	for name := range changed {
		result[name] = true
	}

	// For each changed node, mark all dependents as changed
	for name := range changed {
		for _, dependent := range g.GetDependents(name) {
			result[dependent] = true
		}
	}

	return result
}

// ValidateDependencies checks that all dependencies reference existing nodes.
// Returns an error if any dependency references a non-existent node.
func (g *Graph) ValidateDependencies() error {
	for name, node := range g.nodes {
		for _, dep := range node.DependsOn {
			if _, exists := g.nodes[dep]; !exists {
				return errors.New("service '%s' depends on '%s' which does not exist", name, dep)
			}
		}
	}
	return nil
}

// CalculateDepths returns the dependency depth of each node.
// Depth 0 = no in-graph dependencies.
// Depth N = max(dependency depths) + 1.
// Dependencies outside the graph are ignored (treated as satisfied).
// Returns an error if a circular dependency is detected.
func (g *Graph) CalculateDepths() (map[string]int, error) {
	depths := make(map[string]int, len(g.nodes))
	inProgress := make(map[string]bool)

	for name := range g.nodes {
		if _, err := g.calculateNodeDepth(name, depths, inProgress); err != nil {
			return nil, err
		}
	}
	return depths, nil
}

func (g *Graph) calculateNodeDepth(name string, cache map[string]int, inProgress map[string]bool) (int, error) {
	if d, ok := cache[name]; ok {
		return d, nil
	}
	if inProgress[name] {
		return 0, errors.New("circular dependency detected involving: %s", name)
	}

	node, exists := g.nodes[name]
	if !exists {
		cache[name] = 0
		return 0, nil
	}

	inProgress[name] = true
	maxDep := -1
	for _, dep := range node.DependsOn {
		if _, inGraph := g.nodes[dep]; !inGraph {
			continue // External dep, treat as already satisfied
		}
		depDepth, err := g.calculateNodeDepth(dep, cache, inProgress)
		if err != nil {
			return 0, err
		}
		if depDepth > maxDep {
			maxDep = depDepth
		}
	}
	delete(inProgress, name)

	depth := 0
	if maxDep >= 0 {
		depth = maxDep + 1
	}
	cache[name] = depth
	return depth, nil
}

// HasNode returns true if the graph contains a node with the given name.
func (g *Graph) HasNode(name string) bool {
	_, exists := g.nodes[name]
	return exists
}

// NodeCount returns the number of nodes in the graph.
func (g *Graph) NodeCount() int {
	return len(g.nodes)
}
