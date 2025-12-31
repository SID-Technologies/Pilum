package git

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sid-technologies/pilum/lib/errors"
)

// ChangedFiles returns a list of files that have changed since the given reference.
// If ref is empty, it compares against the default branch (main or master).
func ChangedFiles(ref string) ([]string, error) {
	if ref == "" {
		defaultBranch, err := getDefaultBranch()
		if err != nil {
			return nil, err
		}
		ref = defaultBranch
	}

	// Get changed files between ref and HEAD
	// Using merge-base to find common ancestor, then diffing from there
	mergeBase, err := getMergeBase(ref)
	if err != nil {
		// If merge-base fails (e.g., no common ancestor), diff directly
		mergeBase = ref
	}

	// #nosec G204 - mergeBase is derived from git commands, not user input directly
	cmd := exec.Command("git", "diff", "--name-only", mergeBase+"...HEAD")
	output, err := cmd.Output()
	if err != nil {
		// Try without the triple-dot notation (direct diff)
		// #nosec G204 - mergeBase is derived from git commands, not user input directly
		cmd = exec.Command("git", "diff", "--name-only", mergeBase, "HEAD")
		output, err = cmd.Output()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get changed files")
		}
	}

	return parseLines(string(output)), nil
}

// ChangedFilesUncommitted returns files that have uncommitted changes (staged or unstaged).
func ChangedFilesUncommitted() ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get uncommitted changes")
	}

	staged := parseLines(string(output))

	// Also get untracked files
	cmd = exec.Command("git", "ls-files", "--others", "--exclude-standard")
	output, err = cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get untracked files")
	}

	untracked := parseLines(string(output))

	return append(staged, untracked...), nil
}

// ServiceHasChanges checks if a service directory has any changed files.
// servicePath is the relative path to the service directory.
func ServiceHasChanges(servicePath string, changedFiles []string) bool {
	// Normalize the service path
	servicePath = filepath.Clean(servicePath)

	// Check if any changed file is within the service directory
	for _, file := range changedFiles {
		file = filepath.Clean(file)

		// Check if file is in the service directory
		if strings.HasPrefix(file, servicePath+string(filepath.Separator)) || file == servicePath {
			return true
		}

		// Handle case where service is at root (path is ".")
		if servicePath == "." {
			return true
		}
	}

	return false
}

// getDefaultBranch returns the default branch name (main or master).
func getDefaultBranch() (string, error) {
	// Try to get the default branch from remote
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	output, err := cmd.Output()
	if err == nil {
		// Parse output like "refs/remotes/origin/main"
		ref := strings.TrimSpace(string(output))
		parts := strings.Split(ref, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1], nil
		}
	}

	// Fallback: check if main or master exists
	for _, branch := range []string{"main", "master"} {
		cmd = exec.Command("git", "rev-parse", "--verify", branch)
		if err := cmd.Run(); err == nil {
			return branch, nil
		}
	}

	return "", errors.New("could not determine default branch")
}

// getMergeBase returns the merge base between the given ref and HEAD.
func getMergeBase(ref string) (string, error) {
	cmd := exec.Command("git", "merge-base", ref, "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "failed to get merge base")
	}
	return strings.TrimSpace(string(output)), nil
}

// parseLines splits output by newlines and filters empty lines.
func parseLines(output string) []string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

// IsGitRepository checks if the current directory is inside a git repository.
func IsGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}
