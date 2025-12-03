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
			return "", errors.New(errMsg)
		}
		dir = parentDir
	}
}
