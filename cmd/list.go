package cmd

import (
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/exitcodes"
	"github.com/sid-technologies/pilum/lib/path"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/spf13/cobra"
)

func ListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all discovered services",
		RunE: withJSON(func(cmd *cobra.Command, _ []string) (any, error) {
			root, err := path.FindProjectRoot()
			if err != nil {
				return nil, exitcodes.WithCode(exitcodes.NoServices,
					errors.Wrap(err, "error finding project root"))
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

			// Convert to pointer slice for ListServices
			svcPtrs := make([]*serviceinfo.ServiceInfo, len(services))
			for i := range services {
				svcPtrs[i] = &services[i]
			}

			// Text output (suppressed in JSON mode)
			serviceinfo.ListServices(svcPtrs)

			// Build structured data for JSON
			type jsonService struct {
				Name     string `json:"name"`
				Provider string `json:"provider"`
				Project  string `json:"project"`
				Region   string `json:"region,omitempty"`
				Type     string `json:"type,omitempty"`
				Path     string `json:"path"`
			}
			result := make([]jsonService, len(services))
			for i, s := range services {
				result[i] = jsonService{
					Name:     s.Name,
					Provider: s.Provider,
					Project:  s.Project,
					Region:   s.Region,
					Type:     s.Template,
					Path:     s.Path,
				}
			}
			return result, nil
		}),
	}

	cmd.Flags().StringP("env", "e", "", "Environment to apply (merges overrides from environments block)")
	cmd.Flags().String("provider", "", "Filter services by provider (e.g., gcp, aws, azure)")

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(ListCmd())
}
