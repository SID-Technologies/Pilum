package registry

import (
	"fmt"

	"github.com/sid-technologies/pilum/ingredients/build"
	"github.com/sid-technologies/pilum/ingredients/docker"
	"github.com/sid-technologies/pilum/ingredients/gcp"
	"github.com/sid-technologies/pilum/ingredients/homebrew"
)

// RegisterDefaultHandlers registers all built-in step handlers.
func RegisterDefaultHandlers(reg *CommandRegistry) {
	registerDockerHandlers(reg)
	registerBuildHandlers(reg)
	registerDeployHandlers(reg)
	registerHomebrewHandlers(reg)
}

func registerDockerHandlers(reg *CommandRegistry) {
	// Docker build step (must come before generic build to match "docker-build" first)
	reg.Register("docker", "", func(ctx StepContext) any {
		templatePath := fmt.Sprintf("%s/%s", ctx.TemplatePath, ctx.Service.Template)
		return docker.GenerateDockerBuildCommand(ctx.Service, ctx.ImageName, templatePath)
	})

	// Push/publish step
	reg.Register("push", "", func(ctx StepContext) any {
		return docker.GenerateDockerPushCommand(ctx.ImageName)
	})
	reg.Register("publish", "", func(ctx StepContext) any {
		return docker.GenerateDockerPushCommand(ctx.ImageName)
	})
}

func registerBuildHandlers(reg *CommandRegistry) {
	// Generic build step
	reg.Register("build", "", func(ctx StepContext) any {
		cmd, _ := build.GenerateBuildCommand(ctx.Service, ctx.Registry, ctx.Tag)
		return cmd
	})
}

func registerDeployHandlers(reg *CommandRegistry) {
	// GCP deploy step (provider-specific)
	reg.Register("deploy", "gcp", func(ctx StepContext) any {
		return gcp.GenerateGCPDeployCommand(ctx.Service, ctx.ImageName)
	})

	// AWS deploy step (placeholder for future)
	reg.Register("deploy", "aws", func(_ StepContext) any {
		// TODO: Implement AWS deployment
		return nil
	})

	// Azure deploy step (placeholder for future)
	reg.Register("deploy", "azure", func(_ StepContext) any {
		// TODO: Implement Azure deployment
		return nil
	})
}

func registerHomebrewHandlers(reg *CommandRegistry) {
	const outputDir = "dist"

	// Homebrew multi-platform build
	reg.Register("build", "homebrew", func(ctx StepContext) any {
		return homebrew.GenerateBuildCommand(ctx.Service, ctx.Tag, outputDir)
	})

	// Homebrew archive creation
	reg.Register("archive", "homebrew", func(ctx StepContext) any {
		return homebrew.GenerateArchiveCommand(ctx.Service, ctx.Tag, outputDir)
	})

	// Homebrew checksum generation
	reg.Register("checksum", "homebrew", func(_ StepContext) any {
		return homebrew.GenerateChecksumCommand(outputDir)
	})

	// Homebrew formula generation
	reg.Register("formula", "homebrew", func(ctx StepContext) any {
		formulaPath := fmt.Sprintf("../Homebrew-%s/Formula/%s.rb", ctx.Service.Name, ctx.Service.Name)
		return homebrew.GenerateFormulaCommand(ctx.Service, ctx.Tag, outputDir, formulaPath)
	})
}
