# CLAUDE.md - Pilum

## What is Pilum?

Pilum is a **cloud-agnostic deployment CLI** - define a service once, deploy to any cloud provider. Think "GoReleaser for multi-cloud deployments" but faster and more flexible.

**Why Pilum exists:**
- **GoReleaser limitations** - Static folders and relative directories in Docker builds were problematic
- **Multithreaded execution** - Parallel builds and deploys via worker queues
- **Provider portability** - Swap between ECS ↔ Cloud Run ↔ Azure Container Apps by changing one line
- **Declarative deployments** - Terraform defines infrastructure; Pilum defines how code gets deployed to it

**The cooking metaphor:**
- **Recipes** (`recepies/`) - Deployment workflows (ordered steps)
- **Ingredients** (`ingredients/`) - Cloud-specific operation plugins
- **Services** - Discovered via `service.yaml` files

## Quick Reference

```bash
# Development
make build            # Build to dist/
make lint             # Run golangci-lint
go test ./...         # Run tests

# CLI Commands
pilum add <template>      # Scaffold a new service
pilum list                # List available templates
pilum build [services]    # Build services
pilum deploy [services]   # Full deploy pipeline
pilum check               # Validate service configurations
pilum dry-run             # Preview what would execute
```

## Key Directories

| Directory | Purpose |
|-----------|---------|
| `cmd/` | CLI commands (Cobra) |
| `lib/` | Core libraries (configs, output, errors, worker_queue, etc.) |
| `ingredients/` | Cloud provider plugins (gcp, aws, docker, build) |
| `recepies/` | Deployment recipe YAML files |
| `_configs/` | Embedded config templates (TOML) |
| `_ingredients/` | Embedded ingredient templates |

## Service Configuration (`service.yaml`)

Services are discovered recursively. Define once, deploy anywhere:

```yaml
name: my-service
type: gcp-cloud-run     # gcp-cloud-run, aws-lambda, aws-ecs, azure-container-apps

build:
  language: go
  version: "1.23"
  cmd: "go build -o ./dist"
  env_vars:
    CGO_ENABLED: '0'
  flags:
    ldflags: ["-s", "-w"]

# Switch providers by changing `type` - build config stays identical
```

## Recipes (`recepies/*.yaml`)

Ordered steps for deployment workflows:

```yaml
name: homebrew
provider: homebrew

steps:
  - name: build
    execution_mode: root
    timeout: 60
  - name: archive
    command: ["tar", "-czf", "${name}.tar.gz", "${name}"]
```

## Development Guidelines

- **Linting**: Strict golangci-lint v2 (see `.golangci.yaml`)
- **Output**: Use `lib/output` package, never `log` or raw `fmt.Print` in library code
- **Errors**: Wrap with context using `lib/errors`
- **Pre-commit**: Run `make install-tools` to set up hooks

## Dependencies

| Package | Purpose |
|---------|---------|
| `spf13/cobra` | CLI framework |
| `spf13/viper` | Configuration |
| `gopkg.in/yaml.v3` | YAML parsing |
| `BurntSushi/toml` | TOML parsing |

## Naming Conventions

- Commands: `cmd/<action>.go` (verb-based)
- Libraries: `lib/<domain>/` (noun-based)
- Plugins: `ingredients/<provider>/`
- Recipes: `recepies/<type>-recepie.yaml` (note: "recepie" spelling is intentional)
