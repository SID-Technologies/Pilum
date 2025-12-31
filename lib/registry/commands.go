package registry

import (
	"fmt"

	"github.com/sid-technologies/pilum/ingredients/build"
	"github.com/sid-technologies/pilum/ingredients/docker"
	"github.com/sid-technologies/pilum/ingredients/gcp"
	"github.com/sid-technologies/pilum/ingredients/homebrew"
)

// RegisterDefaultHandlers registers all built-in step handlers.
// Handlers are registered with exact step names matching recipe definitions.
func RegisterDefaultHandlers(reg *CommandRegistry) {
	registerGCPCloudRunHandlers(reg)
	registerHomebrewHandlers(reg)
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
