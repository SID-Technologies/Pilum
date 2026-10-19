// Package aws provides command generators for AWS services.
package aws

import (
	"fmt"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateStatusCommand generates an AWS CLI command to describe a Lambda function.
func GenerateStatusCommand(svc serviceinfo.ServiceInfo) []string {
	cmd := []string{
		"aws", "lambda", "get-function",
		"--function-name", svc.Name,
		"--output", "json",
	}
	return withTarget(cmd, svc)
}

// GenerateLogsCommand generates an AWS CLI command to tail Lambda logs.
func GenerateLogsCommand(svc serviceinfo.ServiceInfo, follow bool) []string {
	function := ParseLambdaConfig(svc).Function
	// An ARN's function name is the segment after "function:".
	_, name, found := strings.Cut(function, ":function:")
	if found {
		function, _, _ = strings.Cut(name, ":")
	}
	logGroup := fmt.Sprintf("/aws/lambda/%s", function)
	cmd := []string{
		"aws", "logs", "tail", logGroup,
	}
	cmd = withTarget(cmd, svc)
	if follow {
		cmd = append(cmd, "--follow")
	}
	return cmd
}
