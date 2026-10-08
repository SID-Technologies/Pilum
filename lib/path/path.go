package path

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sid-technologies/pilum/lib/errors"
)

var ProjectConfig = []string{
	"package.json",
	"cdk.json",
	"tsconfig.json",
	".gitignore",
	"go.mod",
	"Cargo.toml",
}

func FindProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "error getting current working directory")
	}

	dir := currentDir
	for {
		// check for project config files in the current directory
		for _, config := range ProjectConfig {
			res, err := os.Stat(filepath.Join(dir, config))
			if err == nil && !res.IsDir() {
				return dir, nil
			}
		}
		// move up one directory
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			errMsg := fmt.Sprintf("no project configuration found in path hierarchy %s", currentDir)
			return "", errors.New("%s", errMsg)
		}
		dir = parentDir
	}
}

// RepoRoot returns the nearest directory at or above dir that holds .git (a
// directory, or a file in worktrees), or dir itself outside a repository.
// Searches for workspace files stop here, so a lockfile in some unrelated
// parent directory is never picked up.
func RepoRoot(dir string) string {
	for current := dir; ; {
		_, err := os.Stat(filepath.Join(current, ".git"))
		if err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return dir
		}
		current = parent
	}
}
