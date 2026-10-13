package git

import (
	"os/exec"
	"strings"

	"github.com/sid-technologies/pilum/lib/errors"
)

// TopLevel returns the absolute path of the repository's root directory.
func TopLevel() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", errors.Wrap(err, "failed to find the repository root")
	}
	return strings.TrimSpace(string(out)), nil
}

// ChangedFilesSince returns the files (relative to the repository root)
// changed between ref and HEAD, and the ref it compared against. An empty
// ref means the default branch, preferring origin's copy, which is the one
// a CI checkout has. It errors when the comparison can't be made, e.g. in a
// shallow clone that lacks ref.
func ChangedFilesSince(ref string) ([]string, string, error) {
	if ref == "" {
		defaultBranch, err := getDefaultBranch()
		if err != nil {
			return nil, "", err
		}
		ref = defaultBranch
	}

	// #nosec G204 -- ref is passed as a single argument, never through a shell.
	err := exec.Command("git", "rev-parse", "--verify", "--quiet", ref+"^{commit}").Run()
	if err != nil {
		return nil, ref, errors.Wrap(err, "unknown git ref %s", ref)
	}

	// Diff from the merge base, so changes made on ref since HEAD branched
	// off don't count. Without a merge base (shallow history), diff the two
	// trees directly, which can only report more files, not fewer.
	base := ref
	mergeBase, err := getMergeBase(ref)
	if err == nil {
		base = mergeBase
	}

	// #nosec G204 -- base is a ref or commit, passed as a single argument.
	out, err := exec.Command("git", "diff", "--name-only", "--no-relative", base, "HEAD").Output()
	if err != nil {
		return nil, ref, errors.Wrap(err, "failed to get changed files")
	}

	return parseLines(string(out)), ref, nil
}

// ChangedFilesUncommitted returns files (relative to the repository root)
// that have uncommitted changes, staged or not, and untracked files.
func ChangedFilesUncommitted() ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "--no-relative", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get uncommitted changes")
	}

	staged := parseLines(string(output))

	// Also get untracked files
	cmd = exec.Command("git", "ls-files", "--others", "--exclude-standard", "--full-name")
	output, err = cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get untracked files")
	}

	untracked := parseLines(string(output))

	return append(staged, untracked...), nil
}

// getDefaultBranch returns the default branch to compare against: origin's
// copy (origin/main) when it exists, else a local main or master.
func getDefaultBranch() (string, error) {
	name := ""
	out, err := exec.Command("git", "symbolic-ref", "--short", "refs/remotes/origin/HEAD").Output()
	if err == nil {
		name = strings.TrimPrefix(strings.TrimSpace(string(out)), "origin/")
	}

	candidates := []string{"origin/main", "origin/master", "main", "master"}
	if name != "" {
		candidates = append([]string{"origin/" + name, name}, candidates...)
	}
	for _, branch := range candidates {
		// #nosec G204 -- branch is one of the fixed names above or origin's HEAD.
		err := exec.Command("git", "rev-parse", "--verify", "--quiet", branch+"^{commit}").Run()
		if err == nil {
			return branch, nil
		}
	}

	return "", errors.New("could not determine the default branch; pass --since")
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
