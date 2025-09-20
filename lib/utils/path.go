package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	current_dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := current_dir
	for {
		// check for project config files in the current directory
		for _, config := range ProjectConfig {
			res, err := os.Stat(filepath.Join(dir, config))
			if err == nil && !res.IsDir() {
				return dir, nil
			}
		}
		// move up one directory
		parent_dir := filepath.Dir(dir)
		if parent_dir == dir {
			err_msg := fmt.Sprintf("no project configuration found in path hierarchy %s", current_dir)
			return "", errors.New(err_msg)
		}
		dir = parent_dir
	}
}
