package registry

import (
	"fmt"

	"github.com/sid-technologies/pilum/ingredients/azure"
	"github.com/sid-technologies/pilum/ingredients/build"
	"github.com/sid-technologies/pilum/ingredients/cloudflare"
	"github.com/sid-technologies/pilum/ingredients/docker"
	"github.com/sid-technologies/pilum/ingredients/gcp"
	"github.com/sid-technologies/pilum/ingredients/homebrew"
	"github.com/sid-technologies/pilum/ingredients/npm"
)

// RegisterDefaultHandlers registers all built-in step handlers.
// Handlers are registered with exact step names matching recipe definitions.
func RegisterDefaultHandlers(reg *CommandRegistry) {
	registerGCPCloudRunHandlers(reg)
	registerGCPCloudRunJobHandlers(reg)
	registerAzureContainerAppsHandlers(reg)
	registerHomebrewHandlers(reg)
	registerNpmHandlers(reg)
	registerCloudflareHandlers(reg)
}

// registerGCPCloudRunHandlers registers handlers for GCP Cloud Run recipe steps.
// Step names must match exactly: "build binary", "build docker image", etc.
func registerGCPCloudRunHandlers(reg *CommandRegistry) {
	// Step 1: Build binary
	reg.Register("build binary", "", func(ctx StepContext) any {
		cmd, _ := build.GenerateBuildCommand(ctx.Service, ctx.Registry, ctx.Tag)
		return cmd
	})

	// Step 2: Build Docker image
	reg.Register("build docker image", "", func(ctx StepContext) any {
		templatePath := fmt.Sprintf("%s/%s", ctx.TemplatePath, ctx.Service.Template)
		return docker.GenerateDockerBuildCommand(ctx.Service, ctx.ImageName, templatePath)
	})

	// Step 3: Publish to registry (push Docker image)
	reg.Register("publish to registry", "", func(ctx StepContext) any {
		return docker.GenerateDockerPushCommand(ctx.ImageName)
	})

	// Step 4: Deploy to Cloud Run (GCP-specific)
	reg.Register("deploy to cloud run", "gcp", func(ctx StepContext) any {
		return gcp.GenerateGCPDeployCommand(ctx.Service, ctx.ImageName)
	})
}

// registerGCPCloudRunJobHandlers registers handlers for GCP Cloud Run Job recipe steps.
// Step names must match exactly: "deploy job", "execute job".
// Build/push steps are handled by generic handlers registered in registerGCPCloudRunHandlers.
func registerGCPCloudRunJobHandlers(reg *CommandRegistry) {
	// Step 4: Deploy job (create/update the Cloud Run Job definition)
	reg.Register("deploy job", "gcp", func(ctx StepContext) any {
		return gcp.GenerateDeployJobCommand(ctx.Service, ctx.ImageName)
	})

	// Step 5: Execute job (conditionally run the job after deploy)
	reg.Register("execute job", "gcp", func(ctx StepContext) any {
		return gcp.GenerateExecuteJobCommand(ctx.Service)
	})
}

// registerHomebrewHandlers registers handlers for Homebrew recipe steps.
// Step names must match exactly: "build binaries", "create archives", etc.
func registerHomebrewHandlers(reg *CommandRegistry) {
	const outputDir = "dist"

	// Step 1: Build binaries for all platforms
	reg.Register("build binaries", "homebrew", func(ctx StepContext) any {
		return homebrew.GenerateBuildCommand(ctx.Service, ctx.Tag, outputDir)
	})

	// Step 2: Create tar.gz archives
	reg.Register("create archives", "homebrew", func(ctx StepContext) any {
		return homebrew.GenerateArchiveCommand(ctx.Service, ctx.Tag, outputDir)
	})

	// Step 3: Generate SHA256 checksums
	reg.Register("generate checksums", "homebrew", func(_ StepContext) any {
		return homebrew.GenerateChecksumCommand(outputDir)
	})

	// Step 4: Update Homebrew formula
	reg.Register("update formula", "homebrew", func(ctx StepContext) any {
		formulaPath := fmt.Sprintf("%s/%s.rb", outputDir, ctx.Service.Name)
		return homebrew.GenerateFormulaCommand(ctx.Service, ctx.Tag, outputDir, formulaPath)
	})

	// Step 5: Push formula to Homebrew tap
	reg.Register("push to tap", "homebrew", func(ctx StepContext) any {
		formulaPath := fmt.Sprintf("%s/%s.rb", outputDir, ctx.Service.Name)
		return homebrew.GenerateTapPushCommand(ctx.Service, ctx.Tag, formulaPath)
	})
}

// registerNpmHandlers registers handlers for npm package recipe steps.
// Step names must match exactly: "install dependencies", "set version", etc.
func registerNpmHandlers(reg *CommandRegistry) {
	// Step 1: Install dependencies
	reg.Register("install dependencies", "npm", func(_ StepContext) any {
		return npm.GenerateInstallCommand()
	})

	// Step 2: Set version from tag
	reg.Register("set version", "npm", func(ctx StepContext) any {
		return npm.GenerateSetVersionCommand(ctx.Tag)
	})

	// Step 3: Build package (optional, returns nil if no build cmd)
	reg.Register("build package", "npm", func(ctx StepContext) any {
		return npm.GenerateBuildCommand(ctx.Service)
	})

	// Step 4: Publish to npm registry
	reg.Register("publish package", "npm", func(ctx StepContext) any {
		return npm.GeneratePublishCommand(ctx.Service)
	})
}

// registerAzureContainerAppsHandlers registers handlers for Azure Container Apps recipe steps.
// Build/push steps are handled by generic handlers registered in registerGCPCloudRunHandlers.
func registerAzureContainerAppsHandlers(reg *CommandRegistry) {
	// Step 4: Deploy to Azure Container Apps
	reg.Register("deploy to container app", "azure", func(ctx StepContext) any {
		return azure.GenerateDeployCommand(ctx.Service, ctx.ImageName)
	})
}

// registerCloudflareHandlers registers handlers for Cloudflare Pages recipe steps.
// Step names must match exactly: "install dependencies", "build site", "deploy to pages".
func registerCloudflareHandlers(reg *CommandRegistry) {
	// Step 1: Install dependencies
	reg.Register("install dependencies", "cloudflare", func(ctx StepContext) any {
		return cloudflare.GenerateInstallCommand(ctx.Service)
	})

	// Step 2: Build static site (optional, returns nil if no build cmd)
	reg.Register("build site", "cloudflare", func(ctx StepContext) any {
		return cloudflare.GenerateBuildCommand(ctx.Service)
	})

	// Step 3: Deploy to Cloudflare Pages
	reg.Register("deploy to pages", "cloudflare", func(ctx StepContext) any {
		return cloudflare.GenerateDeployCommand(ctx.Service, ctx.Tag)
	})
}
