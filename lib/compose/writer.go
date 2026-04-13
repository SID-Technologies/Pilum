package compose

import (
	"os"
	"path/filepath"

	"github.com/sid-technologies/pilum/lib/errors"

	"gopkg.in/yaml.v3"
)

// WriteComposeFile marshals the File to YAML and writes it to outputPath.
func WriteComposeFile(file *File, outputPath string) error {
	data, err := yaml.Marshal(file)
	if err != nil {
		return errors.Wrap(err, "error marshaling YAML")
	}

	dir := filepath.Dir(outputPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return errors.Wrap(err, "error creating output directory")
		}
	}

	return os.WriteFile(outputPath, data, 0o644)
}
