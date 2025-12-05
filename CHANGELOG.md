# Changelog

All notable changes to Pilum will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `CONTRIBUTING.md` with contributor guidelines
- `docs/adding-a-provider.md` with Lambda, Fargate, and Azure examples
- `docs/architecture.md` explaining internal design
- `docs/troubleshooting.md` for common issues
- Improved README with better CLI reference and examples

## [0.1.2] - 2024-12-03

### Fixed
- CI versioning issues (#15)
- CI output formatting (#14)

## [0.1.1] - 2024-12-02

### Added
- Tag-based step filtering with `--only-tags` and `--exclude-tags` flags (#13)
- `build` command that auto-excludes deploy-tagged steps

### Changed
- Build command now uses recipe system for step execution

### Fixed
- Restructured Homebrew config for dogfooding release workflow

## [0.1.0] - 2024-12-01

### Added
- Initial release
- Recipe-driven deployment system
- GCP Cloud Run recipe and ingredient
- AWS Lambda recipe (SAM-based, explicit commands)
- Homebrew release recipe with formula generation
- Parallel service execution with step barriers
- `deploy` command for full pipeline
- `publish` command for build + push (no deploy)
- `push` command for registry push
- `check` command for validating services against recipes
- `list` command for discovering services
- `dry-run` command for previewing commands
- `--debug` flag for verbose output
- Animated spinners and colored CLI output
- Recipe-driven validation (no Go code per provider)
- Variable substitution in explicit commands (`${name}`, `${tag}`, etc.)
- Command registry with provider-specific and generic handlers
- Worker queue for parallel task execution

### Infrastructure
- GitHub Actions workflows for testing, linting, and releases
- Semantic versioning automation
- Pre-commit hooks with golangci-lint
- Self-deploying via Homebrew recipe (dogfooding)

[Unreleased]: https://github.com/sid-technologies/pilum/compare/v0.1.2...HEAD
[0.1.2]: https://github.com/sid-technologies/pilum/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/sid-technologies/pilum/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/sid-technologies/pilum/releases/tag/v0.1.0
