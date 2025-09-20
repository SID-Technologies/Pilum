package configs

import (
	"os"

	"github.com/BurntSushi/toml"
	"github.com/rs/zerolog/log"
	"github.com/sid-technologies/centurion/lib/types"
)

type Loader struct {
	discovery *Discovery
}

func NewLoaderClient(discovery *Discovery) *Loader {
	return &Loader{
		discovery: discovery,
	}
}

func (l *Loader) LoadConfigs() (map[string]types.Config, error) {
	configs := make(map[string]types.Config)

	paths, err := l.discovery.FindConfigs()
	if err != nil {
		return nil, err
	}

	for _, path := range paths {
		config, err := LoadConfigFromFile(path)
		if err != nil {
			log.Warn().Err(err).Msgf("error loading config from file %s: %v", path, err)
			continue
		}
		configs[config.Name] = config
	}

	return configs, nil
}

func LoadConfigFromFile(path string) (types.Config, error) {
	var config types.Config

	data, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}

	err = toml.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}
