package configs

import (
	"os"
	"path/filepath"
)

type Discovery struct {
	configPath string
	cdkPath    string
}

func NewDiscovery(configPath, cdkPath string) *Discovery {
	return &Discovery{
		configPath: configPath,
		cdkPath:    cdkPath,
	}
}

func (d *Discovery) FindConfigs() ([]string, error) {
	var configs []string

	err := filepath.Walk(d.configPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (filepath.Ext(path) == ".toml") {
			configs = append(configs, path)
		}
		return nil
	})

	return configs, err
}
