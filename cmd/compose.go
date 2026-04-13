package cmd

import (
	"fmt"
	"strings"

	"github.com/sid-technologies/pilum/lib/compose"
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/exitcodes"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/path"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/spf13/cobra"
)

// ComposeCmd generates a docker-compose.yml from discovered services.
func ComposeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "compose [services...]",
		Aliases: []string{"docker-compose", "dc"},
		Short:   "Generate a docker-compose.yml for discovered services",
		Long: `Generate a docker-compose.yml from discovered pilum.yaml services.

Infrastructure dependencies (databases, caches) are configured in .pilum.yml
under compose.dependencies. See 'pilum compose --help' for config format.

Examples:
  pilum compose --all                    # All services
  pilum compose auth-service api-service # Specific services
  pilum compose --all --no-deps          # Without infrastructure dependencies
  pilum compose --all --registry ghcr.io/myorg`,
		RunE: withJSON(runCompose),
	}

	cmd.Flags().Bool("all", false, "Include all discovered services")
	cmd.Flags().StringP("registry", "R", "", "Docker registry prefix (overrides pilum.yaml)")
	cmd.Flags().StringP("tag", "t", "latest", "Image tag (default: latest)")
	cmd.Flags().StringP("output", "o", "docker-compose.yml", "Output file path")
	cmd.Flags().StringP("env", "e", "", "Environment to apply")
	cmd.Flags().String("provider", "", "Filter services by provider")
	cmd.Flags().Bool("no-deps", false, "Don't include infrastructure dependencies")
	cmd.Flags().String("env-file", "", "Write a .env template with service URLs (e.g., apps/web/.env.example)")

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(ComposeCmd())
}

func runCompose(cmd *cobra.Command, args []string) (any, error) {
	root, err := path.FindProjectRoot()
	if err != nil {
		return nil, exitcodes.WithCode(exitcodes.NoServices,
			errors.Wrap(err, "error finding project root"))
	}

	all, _ := cmd.Flags().GetBool("all")
	if !all && len(args) == 0 {
		return nil, exitcodes.WithCode(exitcodes.InvalidArgs,
			errors.New("specify --all to include all services or pass service names as arguments"))
	}

	env, _ := cmd.Flags().GetString("env")
	provider, _ := cmd.Flags().GetString("provider")

	opts := serviceinfo.DefaultDiscoveryOptions()
	opts.NoGitIgnore = NoGitIgnore()
	opts.Env = env

	services, err := serviceinfo.FindServicesWithOptions(root, opts)
	if err != nil {
		return nil, exitcodes.WithCode(exitcodes.NoServices,
			errors.Wrap(err, "error finding services"))
	}

	if provider != "" {
		services = serviceinfo.FilterByProvider(services, provider)
	}
	if len(args) > 0 {
		services = serviceinfo.FilterServices(args, services)
	}
	if len(services) == 0 {
		output.Warning("No services found")
		return nil, nil
	}

	// Load infrastructure dependencies from .pilum.yml
	noDeps, _ := cmd.Flags().GetBool("no-deps")
	var deps []compose.Dependency
	if !noDeps {
		deps, err = compose.LoadDependencies()
		if err != nil {
			return nil, exitcodes.WithCode(exitcodes.Config, err)
		}
	}

	registry, _ := cmd.Flags().GetString("registry")
	tag, _ := cmd.Flags().GetString("tag")
	outputPath, _ := cmd.Flags().GetString("output")
	envFile, _ := cmd.Flags().GetString("env-file")

	composeOpts := compose.Options{
		Registry:     registry,
		Tag:          tag,
		Dependencies: deps,
		DisableDeps:  noDeps,
	}

	file := compose.Generate(services, composeOpts)

	if err := compose.WriteComposeFile(file, outputPath); err != nil {
		return nil, errors.Wrap(err, "error writing docker-compose file")
	}

	// Print summary
	output.Success("Docker Compose file generated: %s", outputPath)
	output.Info("Services included: %d", len(services))

	portMap := extractPorts(file)
	for _, s := range services {
		if p, ok := portMap[s.Name]; ok {
			output.Info("  %s → localhost:%d", s.Name, p)
		}
	}

	printDependencySummary(deps, noDeps)
	writeEnvFileIfRequested(envFile, services, portMap)

	output.Info("")
	output.Info("Usage:")
	output.Info("  docker compose -f %s up --build", outputPath)
	output.Info("  docker compose -f %s up -d", outputPath)
	output.Info("  docker compose -f %s down", outputPath)

	serviceNames := make([]string, len(services))
	for i, s := range services {
		serviceNames[i] = s.Name
	}
	return compose.Result{File: outputPath, Services: serviceNames, Ports: portMap}, nil
}

func extractPorts(file *compose.File) map[string]int {
	portMap := make(map[string]int)
	for _, svc := range file.Services {
		if len(svc.Ports) == 0 {
			continue
		}
		parts := strings.SplitN(svc.Ports[0], ":", 2)
		if len(parts) == 2 {
			var port int
			if _, err := fmt.Sscanf(parts[0], "%d", &port); err == nil {
				portMap[svc.ContainerName] = port
			}
		}
	}
	return portMap
}

func printDependencySummary(deps []compose.Dependency, noDeps bool) {
	if noDeps || len(deps) == 0 {
		return
	}
	output.Info("Dependencies: %d", len(deps))
	for _, dep := range deps {
		compose.MergeDefaults(&dep)
		output.Info("  %s (%s) → localhost:%d", dep.Name, dep.Type, dep.Port)
	}
}

func writeEnvFileIfRequested(envFile string, services []serviceinfo.ServiceInfo, portMap map[string]int) {
	if envFile == "" {
		return
	}
	if err := compose.WriteEnvFile(envFile, services, portMap); err != nil {
		output.Warning("Could not write env file: %s", err)
		return
	}
	output.Success("Environment template written: %s", envFile)
}
