package serviceinfo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/git"
	"github.com/sid-technologies/pilum/lib/graph"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/suggest"

	"gopkg.in/yaml.v2"
)

// FilterOptions configures how services are filtered.
type FilterOptions struct {
	Names       []string // Service names to filter by
	OnlyChanged bool     // Only include services with git changes
	Since       string   // Git ref to compare against (default: main/master)
	NoGitIgnore bool     // Skip reading .gitignore patterns
}

func FindAndFilterServices(root string, filter []string) ([]ServiceInfo, error) {
	return FindAndFilterServicesWithOptions(root, FilterOptions{Names: filter})
}

func FindAndFilterServicesWithOptions(root string, opts FilterOptions) ([]ServiceInfo, error) {
	discoveryOpts := DefaultDiscoveryOptions()
	discoveryOpts.NoGitIgnore = opts.NoGitIgnore

	services, err := FindServicesWithOptions(root, discoveryOpts)
	if err != nil {
		return nil, errors.Wrap(err, "error finding services")
	}

	output.Debugf("Found %d services before filtering", len(services))

	// Filter by name if specified
	if len(opts.Names) > 0 {
		services = FilterServices(opts.Names, services)
		output.Debugf("Filtered by name to %d services", len(services))
	}

	// Filter by git changes if requested
	if opts.OnlyChanged {
		services, err = FilterByChanges(services, opts.Since)
		if err != nil {
			return nil, err
		}
		output.Debugf("Filtered by changes to %d services", len(services))
	}

	return services, nil
}

// FilterByChanges filters services to only those with git changes since the given ref.
// It also includes services that depend on changed services (transitive dependents).
func FilterByChanges(services []ServiceInfo, since string) ([]ServiceInfo, error) {
	if !git.IsGitRepository() {
		output.Warning("Not a git repository, --only-changed has no effect")
		return services, nil
	}

	changedFiles, err := git.ChangedFiles(since)
	if err != nil {
		return nil, errors.Wrap(err, "failed to detect changed files")
	}

	// Also include uncommitted changes
	uncommitted, err := git.ChangedFilesUncommitted()
	if err != nil {
		output.Debugf("Could not get uncommitted changes: %v", err)
	} else {
		changedFiles = append(changedFiles, uncommitted...)
	}

	if len(changedFiles) == 0 {
		output.Info("No changes detected")
		return nil, nil
	}

	output.Debugf("Changed files: %v", changedFiles)

	// Build dependency graph
	g := graph.New()
	serviceMap := make(map[string]ServiceInfo)
	for _, svc := range services {
		g.AddNode(svc.Name, svc.DependsOn)
		serviceMap[svc.Name] = svc
	}

	// Find directly changed services
	directlyChanged := make(map[string]bool)
	for _, svc := range services {
		if git.ServiceHasChanges(svc.Path, changedFiles) {
			directlyChanged[svc.Name] = true
			output.Debugf("Service %s has direct changes", svc.DisplayName())
		}
	}

	// Propagate changes to dependents
	allChanged := g.PropagateChanges(directlyChanged)

	// Log propagated changes
	for name := range allChanged {
		if !directlyChanged[name] {
			output.Debugf("Service %s included (depends on changed service)", name)
		}
	}

	// Build result list
	var changed []ServiceInfo
	for _, svc := range services {
		if allChanged[svc.Name] {
			changed = append(changed, svc)
		}
	}

	if len(changed) == 0 {
		output.Info("No services have changes")
	} else {
		directCount := len(directlyChanged)
		propagatedCount := len(changed) - directCount
		if propagatedCount > 0 {
			output.Info("Found %d service(s) with changes (%d direct, %d via dependencies)",
				len(changed), directCount, propagatedCount)
		} else {
			output.Info("Found %d service(s) with changes", len(changed))
		}
	}

	return changed, nil
}

// DefaultMaxDepth is the default maximum directory depth to search for services.
// This matches the Python implementation's default of 3, with +1 to account for
// the pilum.yaml file itself being one level deeper.
const DefaultMaxDepth = 4

// DiscoveryOptions configures service discovery behavior.
type DiscoveryOptions struct {
	MaxDepth    int  // Maximum directory depth (-1 for unlimited)
	NoGitIgnore bool // Skip reading .gitignore patterns
}

// DefaultDiscoveryOptions returns the default discovery options.
func DefaultDiscoveryOptions() DiscoveryOptions {
	return DiscoveryOptions{
		MaxDepth:    DefaultMaxDepth,
		NoGitIgnore: false,
	}
}

// FindServices searches for pilum.yaml files with default options.
func FindServices(root string) ([]ServiceInfo, error) {
	return FindServicesWithOptions(root, DefaultDiscoveryOptions())
}

// FindServicesWithDepth searches for pilum.yaml files up to the specified depth.
// A maxDepth of 0 means only search the root directory.
// A maxDepth of -1 means unlimited depth.
func FindServicesWithDepth(root string, maxDepth int) ([]ServiceInfo, error) {
	opts := DefaultDiscoveryOptions()
	opts.MaxDepth = maxDepth
	return FindServicesWithOptions(root, opts)
}

// FindServicesWithOptions searches for pilum.yaml files with the given options.
func FindServicesWithOptions(root string, opts DiscoveryOptions) ([]ServiceInfo, error) {
	var services []ServiceInfo

	// Load ignore patterns
	ignorePatterns := loadIgnorePatterns(root, opts.NoGitIgnore)
	if len(ignorePatterns) > 0 {
		output.Debugf("Loaded %d ignore patterns", len(ignorePatterns))
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "error walking %s", path)
		}

		// Get path relative to root for pattern matching and depth calculation
		relPath, _ := filepath.Rel(root, path)

		// Calculate current depth (root = 0, immediate children = 1, etc.)
		depth := 0
		if relPath != "." {
			depth = strings.Count(relPath, string(filepath.Separator)) + 1
		}

		// Skip directories entirely if we've exceeded max depth
		if info.IsDir() {
			// Check depth limit first (most common case for skipping)
			if opts.MaxDepth >= 0 && depth > opts.MaxDepth {
				return filepath.SkipDir
			}

			if shouldIgnore(relPath, ignorePatterns) {
				output.Debugf("Ignoring directory: %s", relPath)
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Base(path) != "pilum.yaml" {
			return nil
		}

		// Check if this pilum.yaml is in an ignored path
		if shouldIgnore(relPath, ignorePatterns) {
			output.Debugf("Ignoring service: %s", relPath)
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrap(err, "error reading %s", path)
		}

		var config map[string]any
		if err := yaml.Unmarshal(content, &config); err != nil {
			return errors.Wrap(err, "error parsing %s", path)
		}

		_, hasName := config["name"]

		if !hasName {
			return errors.Wrap(err, "error parsing %s", path)
		}

		svcRelPath, _ := filepath.Rel(root, filepath.Dir(path))
		svc := NewServiceInfo(config, svcRelPath)

		// Expand multi-region services into separate instances
		expanded := ExpandMultiRegion(*svc)
		services = append(services, expanded...)

		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "error walking %s", root)
	}

	return services, nil
}

// fallbackIgnorePatterns are used when neither .gitignore nor .pilumignore exist.
var fallbackIgnorePatterns = []string{
	"node_modules",
	".git",
	"vendor",
}

// loadIgnorePatterns reads patterns from .gitignore and .pilumignore files.
// .gitignore provides the base patterns, .pilumignore adds project-specific overrides.
// If noGitIgnore is true, .gitignore is skipped.
// Supports:
// - Directory names: "examples" matches any path containing "examples"
// - Paths: "examples/" matches the examples directory at root
// - Globs: "test-*" matches directories starting with "test-"
// - Comments: lines starting with # are ignored
// - Blank lines are ignored
func loadIgnorePatterns(root string, noGitIgnore bool) []string {
	var patterns []string

	// Load from .gitignore first (base patterns) unless disabled
	if !noGitIgnore {
		patterns = append(patterns, loadPatternsFromFile(filepath.Join(root, ".gitignore"))...)
	}

	// Load from .pilumignore (overrides/additions)
	patterns = append(patterns, loadPatternsFromFile(filepath.Join(root, ".pilumignore"))...)

	// If no patterns loaded, use fallback defaults
	if len(patterns) == 0 {
		return fallbackIgnorePatterns
	}

	return patterns
}

// loadPatternsFromFile reads ignore patterns from a file.
func loadPatternsFromFile(path string) []string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Skip negation patterns (e.g., !important.log) - we don't support these
		if strings.HasPrefix(line, "!") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

// shouldIgnore checks if a path matches any ignore pattern.
func shouldIgnore(path string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}

	// Normalize path separators
	path = filepath.ToSlash(path)

	for _, pattern := range patterns {
		pattern = filepath.ToSlash(pattern)

		// Exact directory match (pattern ends with /)
		if strings.HasSuffix(pattern, "/") {
			dir := strings.TrimSuffix(pattern, "/")
			if path == dir || strings.HasPrefix(path, dir+"/") {
				return true
			}
			continue
		}

		// Check if any path component matches the pattern
		parts := strings.Split(path, "/")
		for _, part := range parts {
			matched, _ := filepath.Match(pattern, part)
			if matched {
				return true
			}
		}

		// Also try matching the full path
		matched, _ := filepath.Match(pattern, path)
		if matched {
			return true
		}
	}

	return false
}

func FilterServices(names []string, found []ServiceInfo) []ServiceInfo {
	// Build lookup structures:
	// - byDisplayName: exact match for "service (region)" format
	// - byBaseName: all services with that base name (for multi-region matching)
	byDisplayName := make(map[string]*ServiceInfo)
	byBaseName := make(map[string][]ServiceInfo)
	var allNames []string

	for i := range found {
		svc := &found[i]
		displayName := svc.DisplayName()
		byDisplayName[displayName] = svc
		byBaseName[svc.Name] = append(byBaseName[svc.Name], *svc)
		allNames = append(allNames, svc.Name)
		if displayName != svc.Name {
			allNames = append(allNames, displayName)
		}
	}

	var services []ServiceInfo
	matched := make(map[string]bool) // track what we've added to avoid duplicates

	for _, name := range names {
		foundMatch := false

		// First, try exact display name match (e.g., "global-api (us-central1)")
		if svc, ok := byDisplayName[name]; ok {
			key := svc.DisplayName()
			if !matched[key] {
				services = append(services, *svc)
				matched[key] = true
			}
			foundMatch = true
		}

		// Second, try base name match (e.g., "global-api" matches all regions)
		if instances, ok := byBaseName[name]; ok {
			for _, svc := range instances {
				key := svc.DisplayName()
				if !matched[key] {
					services = append(services, svc)
					matched[key] = true
				}
			}
			foundMatch = true
		}

		if !foundMatch {
			suggestion := suggest.FormatSuggestion(name, allNames)
			if suggestion != "" {
				output.Warning("Service '%s' not found - %s", name, suggestion)
			} else {
				output.Warning("Service '%s' not found", name)
			}
		}
	}

	return services
}

// SortByDependencies sorts services in topological order (dependencies first).
// Services with no dependencies come first, then services that depend on them, etc.
// Returns an error if a circular dependency is detected.
func SortByDependencies(services []ServiceInfo) ([]ServiceInfo, error) {
	if len(services) == 0 {
		return services, nil
	}

	// Check if any service has dependencies - if not, return as-is
	hasDeps := false
	for _, svc := range services {
		if len(svc.DependsOn) > 0 {
			hasDeps = true
			break
		}
	}
	if !hasDeps {
		return services, nil
	}

	// Build dependency graph (using unique names only)
	g := graph.New()
	servicesByName := make(map[string][]ServiceInfo)
	for _, svc := range services {
		if !g.HasNode(svc.Name) {
			g.AddNode(svc.Name, svc.DependsOn)
		}
		servicesByName[svc.Name] = append(servicesByName[svc.Name], svc)
	}

	// Validate that all dependencies exist
	if err := g.ValidateDependencies(); err != nil {
		return nil, err
	}

	// Get topological order
	order, err := g.TopologicalSort()
	if err != nil {
		return nil, err
	}

	// Build sorted result, preserving all services with the same name
	sorted := make([]ServiceInfo, 0, len(services))
	for _, name := range order {
		if svcs, exists := servicesByName[name]; exists {
			sorted = append(sorted, svcs...)
		}
	}

	return sorted, nil
}

// BuildDependencyGraph builds a dependency graph from the given services.
func BuildDependencyGraph(services []ServiceInfo) *graph.Graph {
	g := graph.New()
	for _, svc := range services {
		g.AddNode(svc.Name, svc.DependsOn)
	}
	return g
}

func ListServices(services []*ServiceInfo) {
	output.Header("Found %d services:", len(services))
	for _, svc := range services {
		fmt.Printf("  %s•%s %s\n", output.Primary, output.Reset, svc.DisplayName())
		fmt.Printf("      %sProvider:%s %s\n", output.Muted, output.Reset, svc.Provider)
		fmt.Printf("      %sProject:%s  %s\n", output.Muted, output.Reset, svc.Project)
		if !svc.IsMultiRegion {
			// Only show region separately if not already in display name
			fmt.Printf("      %sRegion:%s   %s\n", output.Muted, output.Reset, svc.Region)
		}
		fmt.Printf("      %sService:%s  %s\n", output.Muted, output.Reset, svc.Runtime.Service)
		fmt.Println()
	}
}
