package cmd

import (
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/path"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/spf13/cobra"
)

func ListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all discovered services",
		RunE: func(_ *cobra.Command, _ []string) error {
			root, err := path.FindProjectRoot()
			if err != nil {
				return errors.Wrap(err, "error finding project root")
			}

			opts := serviceinfo.DefaultDiscoveryOptions()
			opts.NoGitIgnore = NoGitIgnore()

			services, err := serviceinfo.FindServicesWithOptions(root, opts)
			if err != nil {
				return errors.Wrap(err, "error finding services")
			}

			// Convert to pointer slice for ListServices
			svcPtrs := make([]*serviceinfo.ServiceInfo, len(services))
			for i := range services {
				svcPtrs[i] = &services[i]
			}

			serviceinfo.ListServices(svcPtrs)
			return nil
		},
	}

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(ListCmd())
}
