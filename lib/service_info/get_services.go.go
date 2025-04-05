package serviceinfo

import (
	"log"
	"os"
	"path/filepath"

	"github.com/sid-technologies/centurion/lib/errors"
	"gopkg.in/yaml.v2"
)

func FindAndFilterServices(root string, filter []string) ([]ServiceInfo, error) {
	services, err := FindServices(root)
	if err != nil {
		return nil, errors.Wrap(err, "error finding services")
	}
	filtered := FilterServices(filter, services)

	return filtered, nil
}

func FindServices(root string) ([]ServiceInfo, error) {
	var services []ServiceInfo

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Base(path) != "service.yaml" {
			return errors.Wrap(err, "error walking %s", path)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrap(err, "error reading %s", path)
		}

		var config map[string]any
		if err := yaml.Unmarshal(content, &config); err != nil {
			return errors.Wrap(err, "error parsing %s", path)
		}

		_, hasName := config["name"]

		if !hasName {
			return errors.Wrap(err, "error parsing %s", path)
		}

		relPath, _ := filepath.Rel(root, filepath.Dir(path))
		svc := NewServiceInfo(config, relPath)
		services = append(services, *svc)

		return nil
	})

	return services, errors.Wrap(err, "error walking %s", root)
}

func FilterServices(names []string, found []ServiceInfo) []ServiceInfo {
	var serviceMap = make(map[string]*ServiceInfo)
	for _, svc := range found {
		serviceMap[svc.Name] = &svc
	}
	var services []ServiceInfo
	for _, name := range names {
		if svc, ok := serviceMap[name]; ok {
			services = append(services, *svc)
		} else {
			log.Printf("Warning: Service %s not found", name)
		}
	}

	return services
}

func ListServices(services []*ServiceInfo) {
	log.Printf("Found %d services:\n", len(services))
	for _, svc := range services {
		log.Printf("• %s\n", svc.Name)
		log.Printf("    - Provider: %s\n", svc.Provider)
		log.Printf("    - Project: %s\n", svc.Project)
		log.Printf("    - Region: %s\n", svc.Region)
		log.Printf("    - Service: %s\n", svc.Runtime.Service)
		log.Println()
	}
}
