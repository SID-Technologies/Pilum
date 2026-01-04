# CLAUDE.md - Pilum

> Last updated: 2025-12-06

## What is Pilum?

Pilum is a **cloud-agnostic deployment CLI** - define a service once, deploy to any cloud provider. Think "GoReleaser for multi-cloud deployments" but faster and more flexible.

**Why Pilum exists:**
- **GoReleaser limitations** - Static folders and relative directories in Docker builds were problematic
- **Multithreaded execution** - Parallel builds and deploys via worker queues
- **Provider portability** - Swap between ECS ↔ Cloud Run ↔ Azure Container Apps by changing one line
- **Declarative deployments** - Terraform defines infrastructure; Pilum defines how code gets deployed to it

**The cooking metaphor:**
- **Recipes** (`recepies/`) - Deployment workflows with required fields and ordered steps
- **Ingredients** (`ingredients/`) - Cloud-specific command generators
- **Services** - Discovered via `pilum.yaml` files

## Quick Reference

```bash
# Development
make build            # Build to dist/
make lint             # Run golangci-lint
go test ./...         # Run tests

# CLI Commands
pilum list                     # List discovered services (alias: ls)
pilum check [services]         # Validate configs against recipes (alias: validate)
pilum build [services]         # Build services (aliases: b, make)
pilum publish [services]       # Build + push images (alias: p)
pilum push [services]          # Push images to registry (alias: ps)
pilum deploy [services]        # Full deploy pipeline (alias: up)
pilum dry-run [services]       # Preview what would execute (alias: dr)
pilum delete-builds [services] # Clean dist/ directories (alias: clean)

# Common Flags
--tag, -t              # Image/release tag (default: latest)
--debug, -d            # Enable debug output
--timeout, -T          # Step timeout in seconds (default: 60)
--max-workers          # Parallel workers (0 = auto)
--recipe-path          # Custom recipe directory (default: ./recepies)
--only-tags            # Run only steps with these tags (comma-separated)
--exclude-tags         # Skip steps with these tags (comma-separated)
```

## Key Directories

| Directory | Purpose |
|-----------|---------|
| `cmd/` | CLI commands (Cobra) |
| `lib/errors/` | Custom error wrapping |
| `lib/orchestrator/` | Step execution engine |
| `lib/output/` | CLI output formatting |
| `lib/recepie/` | Recipe loading and validation |
| `lib/registry/` | Step handler registration |
| `lib/service_info/` | Service configuration parsing |
| `lib/worker_queue/` | Parallel execution worker pool |
| `ingredients/` | Cloud provider command generators (gcp, docker, homebrew) |
| `recepies/` | Deployment recipe YAML files |

## Architecture

### Recipe-Driven Validation

Recipes define both workflows AND validation. No Go code per provider:

```yaml
# recepies/homebrew-recepie.yaml
name: homebrew
provider: homebrew

required_fields:
  - name: name
    description: Binary and formula name
    type: string
  - name: project
    description: GitHub org for release URLs
    type: string

steps:
  - name: build binaries
    execution_mode: root
    timeout: 300
```

When `pilum check` runs, it validates each service against its recipe's `required_fields`.

### Handler Registry

Step names are matched to handlers in `lib/registry/commands.go`:

```go
// Provider-specific handler
registry.Register("deploy", "gcp", func(ctx StepContext) any {
    return gcp.GenerateGCPDeployCommand(ctx.Service, ctx.ImageName)
})

// Generic handler (any provider)
registry.Register("build", "", func(ctx StepContext) any {
    cmd, _ := build.GenerateBuildCommand(ctx.Service, ctx.Registry, ctx.Tag)
    return cmd
})
```

Provider-specific handlers take precedence over generic ones.

### Adding a New Provider

1. **Create recipe YAML** in `recepies/` with `required_fields` and `steps`
2. **Register handlers** in `lib/registry/commands.go` (optional - can use explicit commands)
3. **Create ingredient** in `ingredients/<provider>/` for command generation (optional)

See `recepies/README.md` for full guide.

## Service Configuration (`pilum.yaml`)

```yaml
name: my-service
provider: gcp           # Matches recipe by provider field
project: my-gcp-project
region: us-central1

build:
  language: go
  version: "1.23"
  cmd: "go build -o ./dist"
  env_vars:
    CGO_ENABLED: '0'
  flags:
    ldflags: ["-s", "-w"]
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
