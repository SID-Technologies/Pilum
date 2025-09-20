package cdk

import (
	"errors"
	"os"
	"path/filepath"
)

func GetPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.New("error getting current working directory")
	}

	configs_path := filepath.Join(cwd, "_configs")
	return configs_path, nil
}
