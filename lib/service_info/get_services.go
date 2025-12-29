package serviceinfo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/suggest"

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

		// Expand multi-region services into separate instances
		expanded := ExpandMultiRegion(*svc)
		services = append(services, expanded...)

		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "error walking %s", root)
	}

	return services, nil
}

func FilterServices(names []string, found []ServiceInfo) []ServiceInfo {
	// Build lookup structures:
	// - byDisplayName: exact match for "service (region)" format
	// - byBaseName: all services with that base name (for multi-region matching)
	byDisplayName := make(map[string]*ServiceInfo)
	byBaseName := make(map[string][]ServiceInfo)
	var allNames []string

	for i := range found {
		svc := &found[i]
		displayName := svc.DisplayName()
		byDisplayName[displayName] = svc
		byBaseName[svc.Name] = append(byBaseName[svc.Name], *svc)
		allNames = append(allNames, svc.Name)
		if displayName != svc.Name {
			allNames = append(allNames, displayName)
		}
	}

	var services []ServiceInfo
	matched := make(map[string]bool) // track what we've added to avoid duplicates

	for _, name := range names {
		foundMatch := false

		// First, try exact display name match (e.g., "global-api (us-central1)")
		if svc, ok := byDisplayName[name]; ok {
			key := svc.DisplayName()
			if !matched[key] {
				services = append(services, *svc)
				matched[key] = true
			}
			foundMatch = true
		}

		// Second, try base name match (e.g., "global-api" matches all regions)
		if instances, ok := byBaseName[name]; ok {
			for _, svc := range instances {
				key := svc.DisplayName()
				if !matched[key] {
					services = append(services, svc)
					matched[key] = true
				}
			}
			foundMatch = true
		}

		if !foundMatch {
			suggestion := suggest.FormatSuggestion(name, allNames)
			if suggestion != "" {
				output.Warning("Service '%s' not found - %s", name, suggestion)
			} else {
				output.Warning("Service '%s' not found", name)
			}
		}
	}

	return services
}

func ListServices(services []*ServiceInfo) {
	output.Header("Found %d services:", len(services))
	for _, svc := range services {
		fmt.Printf("  %s•%s %s\n", output.Primary, output.Reset, svc.DisplayName())
		fmt.Printf("      %sProvider:%s %s\n", output.Muted, output.Reset, svc.Provider)
		fmt.Printf("      %sProject:%s  %s\n", output.Muted, output.Reset, svc.Project)
		if !svc.IsMultiRegion {
			// Only show region separately if not already in display name
			fmt.Printf("      %sRegion:%s   %s\n", output.Muted, output.Reset, svc.Region)
		}
		fmt.Printf("      %sService:%s  %s\n", output.Muted, output.Reset, svc.Runtime.Service)
		fmt.Println()
	}
}
