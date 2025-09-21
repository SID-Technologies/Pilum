package registry

import "github.com/rs/zerolog/log"

func RegisterCommands(registry *CommandRegistry) {
	log.Info().Msg("Registering commands")
	log.Info().Msgf("Registered commands: %v", registry)
	// // AWS Lambda commands
	// registry.Register("aws", "lambda", "build", awsLambdaBuild)
	// registry.Register("aws", "lambda", "package", awsLambdaPackage)
	// registry.Register("aws", "lambda", "deploy", awsLambdaDeploy)

	// // GCP Cloud Run commands
	// registry.Register("gcp", "cloudrun", "build", gcpCloudRunBuild)
	// registry.Register("gcp", "cloudrun", "push", gcpCloudRunPush)
	// registry.Register("gcp", "cloudrun", "deploy", gcpCloudRunDeploy)

	// // Docker commands
	// registry.Register("docker", "container", "build", dockerBuild)
	// registry.Register("docker", "container", "push", dockerPush)
}
