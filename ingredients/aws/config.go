package aws

import (
	"github.com/sid-technologies/pilum/lib/configutil"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// Config is the aws: block, shared by every AWS recipe.
type Config struct {
	// AccountID owns the ECR registry images are pushed to. Falls back to
	// registry_name, which AWS services used before the aws: block existed.
	AccountID string
	// Profile is passed as --profile to every aws command. Empty leaves
	// credentials to the CLI's own chain (env vars, AWS_PROFILE, an
	// instance or OIDC role), which is what CI normally uses.
	Profile string
	// CreateRepository creates the ECR repository before the first push
	// when it doesn't exist yet.
	CreateRepository bool
}

// ParseConfig reads the aws: block of a service.
func ParseConfig(svc serviceinfo.ServiceInfo) Config {
	awsMap := configutil.MapFromAny(svc.Config["aws"])
	accountID := configutil.GetString(awsMap, "account_id", "")
	if accountID == "" {
		accountID = svc.RegistryName
	}
	return Config{
		AccountID:        accountID,
		Profile:          configutil.GetString(awsMap, "profile", ""),
		CreateRepository: configutil.GetBool(awsMap, "create_repository", false),
	}
}

// AccountID returns the account that owns the service's ECR registry.
func AccountID(svc serviceinfo.ServiceInfo) string {
	return ParseConfig(svc).AccountID
}

// RegistryHost is the ECR registry for the service's account and region,
// or "" when either is missing.
func RegistryHost(svc serviceinfo.ServiceInfo) string {
	accountID := AccountID(svc)
	if accountID == "" || svc.Region == "" {
		return ""
	}
	return accountID + ".dkr.ecr." + svc.Region + ".amazonaws.com"
}

// withTarget appends the region and profile flags every aws command takes.
func withTarget(cmd []string, svc serviceinfo.ServiceInfo) []string {
	if svc.Region != "" {
		cmd = append(cmd, "--region", svc.Region)
	}
	profile := ParseConfig(svc).Profile
	if profile != "" {
		cmd = append(cmd, "--profile", profile)
	}
	return cmd
}
