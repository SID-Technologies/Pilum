package configs

import (
	configs "github.com/sid-technologies/centurion/_configs"
	ingredients "github.com/sid-technologies/centurion/_ingredients"
)

type Client struct {
	Loader    *Loader
	Discovery *Discovery
	Registry  *Registry
}

func NewClient() (*Client, error) {
	ingredientsPath, err := ingredients.GetPath()
	if err != nil {
		return nil, err
	}

	configsPath, err := configs.GetPath()
	if err != nil {
		return nil, err
	}

	discovery := NewDiscovery(configsPath, ingredientsPath)
	loader := NewLoaderClient(discovery)
	registry, err := NewRegistry(loader)
	if err != nil {
		return nil, err
	}

	return &Client{
		Loader:    loader,
		Discovery: discovery,
		Registry:  registry,
	}, nil
}
