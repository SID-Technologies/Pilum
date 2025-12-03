package serviceinfo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/output"

	"gopkg.in/yaml.v2"
)

func FindAndFilterServices(root string, filter []string) ([]ServiceInfo, error) {
	services, err := FindServices(root)
	if err != nil {
		return nil, errors.Wrap(err, "error finding services")
	}
	if len(filter) == 0 {
		return services, nil
	}
	output.Debugf("Found %d services before filtering", len(services))
	filtered := FilterServices(filter, services)
	output.Debugf("Filtered services down to %d", len(filtered))

	return filtered, nil
}

func FindServices(root string) ([]ServiceInfo, error) {
	var services []ServiceInfo

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "error walking %s", path)
		}

		if info.IsDir() || filepath.Base(path) != "service.yaml" {
			return nil
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

	if err != nil {
		return nil, errors.Wrap(err, "error walking %s", root)
	}

	return services, nil
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
			output.Warning("Service %s not found", name)
		}
	}

	return services
}

func ListServices(services []*ServiceInfo) {
	output.Header("Found %d services:", len(services))
	for _, svc := range services {
		fmt.Printf("  %s•%s %s\n", output.Primary, output.Reset, svc.Name)
		fmt.Printf("      %sProvider:%s %s\n", output.Muted, output.Reset, svc.Provider)
		fmt.Printf("      %sProject:%s  %s\n", output.Muted, output.Reset, svc.Project)
		fmt.Printf("      %sRegion:%s   %s\n", output.Muted, output.Reset, svc.Region)
		fmt.Printf("      %sService:%s  %s\n", output.Muted, output.Reset, svc.Runtime.Service)
		fmt.Println()
	}
}
