// Package aws provides command generators for AWS services.
package aws

import (
	"fmt"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateStatusCommand generates an AWS CLI command to describe a Lambda function.
func GenerateStatusCommand(svc serviceinfo.ServiceInfo) []string {
	cmd := []string{
		"aws", "lambda", "get-function",
		"--function-name", svc.Name,
		"--output", "json",
	}
	if svc.Region != "" {
		cmd = append(cmd, "--region", svc.Region)
	}
	return cmd
}

// GenerateLogsCommand generates an AWS CLI command to tail Lambda logs.
func GenerateLogsCommand(svc serviceinfo.ServiceInfo, follow bool) []string {
	logGroup := fmt.Sprintf("/aws/lambda/%s", svc.Name)
	cmd := []string{
		"aws", "logs", "tail", logGroup,
	}
	if svc.Region != "" {
		cmd = append(cmd, "--region", svc.Region)
	}
	if follow {
		cmd = append(cmd, "--follow")
	}
	return cmd
}
