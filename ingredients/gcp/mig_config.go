package gcp

import (
	"strconv"
	"strings"

	"github.com/sid-technologies/pilum/lib/configutil"
)

// Defaults for rolling updates. Max unavailable 0 means an old instance is
// only removed once its replacement is up, so capacity never drops.
const (
	defaultMIGWaitTimeout = 600
	defaultZonalSurge     = "1"
	// A regional MIG's fixed max surge must be 0 or at least its zone count.
	// Three covers the default distribution across three zones.
	defaultRegionalSurge = "3"
)

// MIGConfig holds the `mig:` block of a gcp-mig-container service. Pilum
// doesn't create the MIG; these fields identify one that already exists.
type MIGConfig struct {
	Name         string // Managed instance group name
	Zone         string // Zone, for a zonal MIG
	Region       string // Region, for a regional MIG (exclusive with Zone)
	TemplateBase string // New templates are named <base>-<tag>

	// ImageMetadataKey is the metadata key the startup script reads the
	// image from. Defaults to DefaultImageMetadataKey.
	ImageMetadataKey string

	// TemplateRegion makes new templates regional (in this region) instead of
	// global. Must match the scope of the MIG's current template.
	TemplateRegion string

	MaxSurge       string // Fixed number or percent, e.g. "1" or "20%"
	MaxUnavailable string // Fixed number or percent
	WaitTimeout    int    // Seconds to wait for the group to become stable

	// VM overrides. Each is optional: unset means the new template keeps the
	// current template's value, so a MIG set up elsewhere is left alone.
	MachineType    string   // e.g. "e2-small"
	Network        string   // VPC network name or resource path
	Subnet         string   // Subnetwork name or resource path, in the MIG's region
	ExternalIP     *bool    // false removes the external IP; true adds an ephemeral one
	NetworkTags    []string // Replaces the template's tags when set (firewall rules target these)
	SetNetworkTags bool     // network_tags was present, even if empty
	ServiceAccount string   // Email the VMs run as
}

// SubnetRegion is the region a bare subnet name resolves in: the MIG's region,
// or the region its zone is in.
func (c MIGConfig) SubnetRegion() string {
	if c.Region != "" {
		return c.Region
	}
	i := strings.LastIndex(c.Zone, "-")
	if i < 0 {
		return ""
	}
	return c.Zone[:i]
}

// ParseMIGConfig extracts the `mig:` block, filling in rollout defaults.
func ParseMIGConfig(config map[string]any) MIGConfig {
	m := configutil.MapFromAny(config["mig"])

	cfg := MIGConfig{
		Name:             configutil.GetString(m, "name", ""),
		Zone:             configutil.GetString(m, "zone", ""),
		Region:           configutil.GetString(m, "region", ""),
		TemplateBase:     configutil.GetString(m, "template_base", ""),
		ImageMetadataKey: configutil.GetString(m, "image_metadata_key", DefaultImageMetadataKey),
		TemplateRegion:   configutil.GetString(m, "template_region", ""),
		MaxSurge:         configutil.GetString(m, "max_surge", ""),
		MaxUnavailable:   configutil.GetString(m, "max_unavailable", "0"),
		WaitTimeout:      configutil.GetInt(m, "wait_timeout", defaultMIGWaitTimeout),
		MachineType:      configutil.GetString(m, "machine_type", ""),
		Network:          configutil.GetString(m, "network", ""),
		Subnet:           configutil.GetString(m, "subnet", ""),
		ServiceAccount:   configutil.GetString(m, "service_account", ""),
	}

	_, hasExternalIP := m["external_ip"]
	if hasExternalIP {
		externalIP := configutil.GetBool(m, "external_ip", false)
		cfg.ExternalIP = &externalIP
	}
	_, cfg.SetNetworkTags = m["network_tags"]
	cfg.NetworkTags = configutil.GetStringSlice(m, "network_tags")

	// YAML `max_surge: 2` arrives as an int, not a string.
	surge := configutil.GetInt(m, "max_surge", -1)
	if surge >= 0 {
		cfg.MaxSurge = strconv.Itoa(surge)
	}
	unavailable := configutil.GetInt(m, "max_unavailable", -1)
	if unavailable >= 0 {
		cfg.MaxUnavailable = strconv.Itoa(unavailable)
	}

	if cfg.MaxSurge == "" {
		cfg.MaxSurge = defaultZonalSurge
		if cfg.Zone == "" && cfg.Region != "" {
			cfg.MaxSurge = defaultRegionalSurge
		}
	}

	return cfg
}

// LocationFlags returns --zone or --region for gcloud instance-group commands.
func (c MIGConfig) LocationFlags() []string {
	if c.Zone != "" {
		return []string{"--zone", c.Zone}
	}
	if c.Region != "" {
		return []string{"--region", c.Region}
	}
	return nil
}
