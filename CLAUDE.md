# CLAUDE.md - Pilum

## What is Pilum?

Pilum (the Roman javelin) is a **cloud-agnostic deployment CLI** that lets you define a service once and deploy it to any cloud provider. Think "GoReleaser for multi-cloud deployments" - but faster, more flexible, and without the static folder limitations.

**Why Pilum exists:**
- **GoReleaser limitations** - Static folders and relative directories in Docker builds were problematic; Pilum solves this
- **Multithreaded execution** - Parallel builds and deploys via worker queues make it significantly faster
- **Provider portability** - Swap between ECS ↔ Cloud Run ↔ Azure Container Apps by changing one line in your config
- **Declarative deployments** - Terraform defines infrastructure; Pilum defines how your code gets deployed to it

**The cooking metaphor:**
- **Recipes** define deployment workflows (steps to execute)
- **Ingredients** are the plugins that implement cloud-specific operations
- **Chef** orchestrates recipe execution
- **Services** are discovered via `service.yaml` files

## Quick Reference

```bash
# Development
make install-tools    # Install pre-commit hooks
make lint             # Run all linters
go test ./...         # Run tests

# CLI Commands
pilum add <template>      # Scaffold a new service from template
pilum list                # List available templates
pilum build [services]    # Build services
pilum deploy [services]   # Full deploy pipeline
pilum check               # Validate service configurations
pilum dry-run             # Preview what would execute
```

## Project Structure

```
pilum/
├── main.go                    # Entry point
├── service.yaml               # Example service configuration
├── cmd/                       # CLI commands (Cobra)
│   ├── root.go               # Root command with banner
│   ├── add.go                # Add service templates
│   ├── build.go              # Build services
│   ├── deploy.go             # Full deploy pipeline
│   ├── publish.go            # Publish artifacts
│   ├── push.go               # Push to registry
│   ├── check.go              # Validate configs
│   ├── dry-run.go            # Preview execution
│   ├── list.go               # List templates
│   ├── delete.go             # Delete build artifacts
│   ├── completion.go         # Shell completions
│   └── helpers.go            # Flag binding utilities
│
├── lib/                       # Core libraries
│   ├── types/                # Type definitions
│   │   ├── config.go         # Config, FlagArg, ConfigFile
│   │   └── enums.go          # TemplateType enum
│   ├── configs/              # Configuration management
│   │   ├── client.go         # Main config client (factory)
│   │   ├── discovery.go      # Find config files
│   │   ├── loader.go         # Parse TOML configs
│   │   └── registry.go       # Store/retrieve configs
│   ├── service_info/         # Service metadata
│   │   ├── service_info.go   # ServiceInfo struct
│   │   ├── get_services.go   # Service discovery
│   │   └── helpers.go        # Utilities
│   ├── recepie/              # Recipe definitions
│   │   ├── recepie.go        # Recipe struct
│   │   └── loader.go         # YAML recipe loader
│   ├── worker_queue/         # Parallel execution
│   │   ├── worker_queue.go   # Work queue manager
│   │   ├── command_worker.go # Command executor
│   │   └── task_info.go      # Task tracking
│   ├── chef/                 # Recipe orchestration
│   │   └── chef.go           # ExecuteRecipe function
│   ├── output/               # CLI output formatting
│   │   ├── header.go         # ASCII banner with gradients
│   │   ├── colors.go         # ANSI color codes
│   │   └── *.go              # Various output formatters
│   ├── writer/               # File operations
│   │   └── writer.go         # Template-based file writing
│   ├── flags/                # Flag parsing
│   │   └── flags.go          # Custom flag parser
│   ├── errors/               # Error handling
│   │   └── errors.go         # Wrapped errors with context
│   ├── registry/             # Command registry
│   │   └── command_registry.go
│   └── utils/                # Utilities
│       └── path.go           # Path helpers
│
├── ingredients/               # Cloud provider plugins
│   ├── build/                # Generic build
│   ├── docker/               # Docker build/push
│   ├── gcp/                  # Google Cloud Run
│   ├── aws/                  # AWS Lambda (planned)
│   └── homebrew/             # Homebrew packaging (planned)
│
├── recepies/                  # Deployment recipes (YAML)
│   ├── homebrew-recepie.yaml
│   ├── aws-lambda-recepie.yaml
│   └── gcp-cloud-run-recepie.yaml
│
├── _configs/                  # Embedded config templates (TOML)
├── _ingredients/              # Embedded ingredient templates
└── .github/workflows/         # CI/CD pipelines
```

## Key Concepts

### Service Configuration (`service.yaml`)

The core idea: define your service once, deploy anywhere. Services are discovered recursively via `service.yaml` files:

```yaml
name: my-service
type: gcp-cloud-run     # Where to deploy: gcp-cloud-run, aws-lambda, aws-ecs, azure-container-apps

build:
  language: go
  version: "1.23"
  cmd: "go build -o ./dist"
  env_vars:
    CGO_ENABLED: '0'
    GOOS: linux
    GOARCH: amd64
  flags:
    ldflags:
      - "-s"
      - "-w"

# To switch providers, just change `type` - the build stays the same
# type: aws-ecs         # Now deploys to ECS instead of Cloud Run
```

**Provider portability**: If your service is containerized, switching from Cloud Run to ECS is a one-line change. The build config, env vars, and secrets stay identical.

### Recipes (`recepies/*.yaml`)

Recipes define ordered steps for deployment workflows:

```yaml
name: homebrew
description: Package for Homebrew
provider: homebrew
service: package

steps:
  - name: build
    execution_mode: root
    timeout: 60

  - name: archive
    command: ["tar", "-czf", "${name}.tar.gz", "${name}"]
    timeout: 30
```

### Template Configs (`_configs/*.toml`)

Define scaffolding templates with required flags and files to copy:

```toml
[config]
name = "template-name"
type = "construct"

[[options]]
name = "project-name"
flag = "--project-name"
type = "string"
required = true

[[files]]
path = "source/template.yaml"
outputPath = "output/result.yaml"
```

## Architecture Patterns

| Pattern | Location | Purpose |
|---------|----------|---------|
| Command | `cmd/` | Cobra CLI subcommands |
| Factory | `configs.Client` | Create config subsystem |
| Registry | `configs.Registry` | Store/retrieve configs |
| Plugin | `ingredients/` | Cloud provider abstraction |
| Worker Queue | `worker_queue/` | Parallel task execution |
| Template | `writer/` | File generation with Go templates |
| Discovery | `configs.Discovery` | Find files by pattern |

## Development Guidelines

### Code Style

- **Linting**: Strict golangci-lint with 40+ rules (see `.golangci.yaml`)
- **Logging**: Use `zerolog`, never `fmt.Print` in library code
- **Errors**: Wrap with context using `lib/errors`
- **Imports**: Ordered by gci (standard, external, internal)

### Testing

```bash
go test -timeout=5m -race ./...
```

### Pre-commit Hooks

```bash
make install-tools  # Install hooks
make lint           # Run manually
```

Hooks enforce: formatting, import ordering, linting, go mod tidy.

## Dependencies

| Package | Purpose |
|---------|---------|
| `spf13/cobra` | CLI framework |
| `spf13/viper` | Configuration management |
| `rs/zerolog` | Structured logging |
| `pkg/errors` | Error wrapping |
| `BurntSushi/toml` | TOML parsing |
| `gopkg.in/yaml.v3` | YAML parsing |

## How It Differs from Alternatives

| Tool | Purpose | Pilum's Advantage |
|------|---------|----------------------|
| **GoReleaser** | Release automation | Handles static folders, relative Docker paths; multithreaded |
| **Terraform** | Infrastructure provisioning | Pilum handles deployment TO infra, not creating it |
| **Docker Compose** | Local multi-container | Pilum is for cloud deployments with provider abstraction |
| **Kubernetes** | Container orchestration | Pilum deploys TO K8s/Cloud Run/ECS without K8s lock-in |

## Current Status

**Branch**: `feat(core)-baseline-cli`

Implemented:
- CLI framework with commands
- Config discovery and loading
- Service discovery via `service.yaml`
- Template scaffolding (`add` command)
- Recipe YAML loading
- Worker queue for parallel execution
- Output formatting with colors

In Progress:
- Chef recipe execution engine
- Cloud provider ingredients (GCP, AWS, Azure)
- Full deploy pipeline integration

## Naming Conventions

- Commands: `cmd/<action>.go` (verb-based: add, build, deploy)
- Libraries: `lib/<domain>/` (noun-based: configs, writer, errors)
- Plugins: `ingredients/<provider>/` (by cloud provider)
- Recipes: `recepies/<type>-recepie.yaml` (note: "recepie" spelling is intentional)
