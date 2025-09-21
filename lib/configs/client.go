package configs

import (
	configs "github.com/sid-technologies/centurion/_configs"
	ingredients "github.com/sid-technologies/centurion/_ingredients"
	"github.com/sid-technologies/centurion/lib/errors"
)

type Client struct {
	Loader    *Loader
	Discovery *Discovery
	Registry  *Registry
}

func NewClient() (*Client, error) {
	ingredientsPath, err := ingredients.GetPath()
	if err != nil {
		return nil, errors.Wrap(err, "error getting ingredients path: %v")
	}

	configsPath, err := configs.GetPath()
	if err != nil {
		return nil, errors.Wrap(err, "error getting configs path: %v")
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
