package configs

import (
	"github.com/sid-technologies/centurion/lib/types"
)

type Registry struct {
	configs map[string]types.Config
}

func NewRegistry(loader *Loader) (*Registry, error) {
	configs, err := loader.LoadConfigs()
	if err != nil {
		return nil, err
	}

	return &Registry{
		configs: configs,
	}, nil
}

func (r *Registry) Get(name string) (types.Config, bool) {
	config, exists := r.configs[name]
	return config, exists
}

func (r *Registry) List() []types.Config {
	var list []types.Config
	for _, config := range r.configs {
		list = append(list, config)
	}

	return list
}

func (r *Registry) ListByType(configType types.TemplateType) []types.Config {
	var list []types.Config
	for _, config := range r.configs {
		if config.Type == configType {
			list = append(list, config)
		}
	}
	return list
}
