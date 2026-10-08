package cmd

import (
	"net/http"
	"strings"
	"time"

	"github.com/sid-technologies/pilum/ingredients/gcp"
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/migtemplate"
	"github.com/sid-technologies/pilum/lib/orchestrator/workers"
	"github.com/sid-technologies/pilum/lib/output"

	"github.com/spf13/cobra"
)

// MIGTemplateCmd is the "create instance template" step of the
// gcp-mig-container recipe. It's a pilum subcommand rather than a gcloud
// command because gcloud can't copy a template (see lib/migtemplate). Hidden:
// it's plumbing for the recipe, not something to run by hand.
func MIGTemplateCmd() *cobra.Command {
	var req migtemplate.Request
	var timeout int
	var networkTags string

	cmd := &cobra.Command{
		Use:    "mig-template",
		Short:  "Create a MIG's next instance template with a new container image",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Only what the service sets is replaced; the rest stays as the
			// current template has it.
			req.Overrides.SetEnv = cmd.Flags().Changed("env")
			req.Overrides.SetSecrets = cmd.Flags().Changed("secret")
			req.Overrides.SetNetworkTags = cmd.Flags().Changed("network-tags")
			for _, tag := range strings.Split(networkTags, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					req.Overrides.NetworkTags = append(req.Overrides.NetworkTags, tag)
				}
			}

			run := func(c []string) (string, error) {
				return workers.CaptureWorker(workers.NewTaskInfo(c, "", req.MIG, "root", nil, nil, timeout, false, 1))
			}
			client := &http.Client{Timeout: 3 * time.Minute}

			result, err := migtemplate.Create(req, run, client)
			if err != nil {
				return errors.Wrap(err, "creating instance template")
			}
			switch {
			case result.Source == req.Name:
				output.Info("MIG %s already runs %s", req.MIG, req.Name)
			case result.Created:
				output.Success("Created %s from %s", req.Name, result.Source)
			default:
				output.Info("Template %s already exists with this image; reusing it", req.Name)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&req.Project, "project", "", "GCP project")
	cmd.Flags().StringVar(&req.MIG, "mig", "", "Managed instance group name")
	cmd.Flags().StringVar(&req.Zone, "zone", "", "Zone of a zonal MIG")
	cmd.Flags().StringVar(&req.Region, "region", "", "Region of a regional MIG")
	cmd.Flags().StringVar(&req.Name, "name", "", "Name of the template to create")
	cmd.Flags().StringVar(&req.Image, "image", "", "Container image the new template runs")
	cmd.Flags().StringVar(&req.ImageKey, "image-key", gcp.DefaultImageMetadataKey, "Metadata key the startup script reads the image from")
	cmd.Flags().StringVar(&req.TemplateRegion, "template-region", "", "Create a regional template here (default: global)")
	cmd.Flags().IntVar(&timeout, "timeout", 60, "Timeout per gcloud query in seconds")

	o := &req.Overrides
	cmd.Flags().StringVar(&o.MachineType, "machine-type", "", "Machine type (default: keep the template's)")
	cmd.Flags().StringVar(&o.Network, "network", "", "VPC network name or path (default: keep)")
	cmd.Flags().StringVar(&o.Subnet, "subnet", "", "Subnetwork name or path (default: keep)")
	cmd.Flags().StringVar(&o.SubnetRegion, "subnet-region", "", "Region a bare --subnet name is in")
	cmd.Flags().StringVar(&o.ExternalIP, "external-ip", "", "true or false (default: keep)")
	cmd.Flags().StringVar(&networkTags, "network-tags", "", "Comma-separated network tags, replacing the template's")
	cmd.Flags().StringVar(&o.ServiceAccount, "service-account", "", "Service account email (default: keep)")
	cmd.Flags().StringArrayVar(&o.Env, "env", nil, "KEY=VALUE container env var, repeatable; replaces container-env")
	cmd.Flags().StringArrayVar(&o.Secrets, "secret", nil, "NAME=secret version resource, repeatable; replaces container-secrets")

	return cmd
}

//nolint:gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(MIGTemplateCmd())
}
