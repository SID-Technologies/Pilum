# Pilum

A cloud-agnostic build and deployment CLI. Define your service once, deploy to any cloud provider.

```bash
pilum deploy --tag=v1.0.0
```

**Think "GoReleaser for multi-cloud deployments"** - Pilum handles the build → push → deploy pipeline while your infrastructure-as-code (Terraform, Pulumi) defines the actual resources.

## Features

- **Recipe-driven deployments** - Define reusable deployment workflows in YAML
- **Recipe-driven validation** - Each recipe declares required fields, no Go code per provider
- **Multi-cloud support** - GCP Cloud Run, AWS Lambda, Azure Container Apps, Homebrew, and more
- **Parallel execution** - Deploy multiple services concurrently with step barriers
- **Step filtering** - Run only build steps, only deploy steps, or custom tag combinations
- **Dry-run mode** - Preview commands before executing
- **Beautiful CLI** - Animated spinners, colored output, clear progress

## Installation

### Homebrew

```bash
brew tap sid-technologies/pilum
brew install pilum
```

### From source

```bash
go install github.com/sid-technologies/pilum@latest
```

## Quick Start

### 1. Create a service configuration

Create a `service.yaml` in your project:

```yaml
name: my-api
provider: gcp
project: my-gcp-project
region: us-central1

build:
  language: go
  version: "1.23"
```

### 2. Validate your configuration

```bash
pilum check
```

This validates your `service.yaml` against the recipe's required fields.

### 3. Deploy

```bash
# Deploy all services
pilum deploy --tag=v1.0.0

# Deploy specific service
pilum deploy my-api --tag=v1.0.0

# Preview without executing
pilum deploy --dry-run --tag=v1.0.0

# Build and push only (no deploy)
pilum publish --tag=v1.0.0
```

## Recipes

Pilum uses recipes to define deployment workflows. Each recipe defines:
- **Steps** - Ordered commands to execute
- **Required fields** - What your `service.yaml` must contain

### Built-in Recipes

| Recipe | Provider | Required Fields |
|--------|----------|-----------------|
| `gcp-cloud-run` | `gcp` | `project`, `region`, `name` |
| `aws-lambda` | `aws` | `region`, `project` |
| `homebrew` | `homebrew` | `name`, `project` |

### Custom Recipes

Create your own recipes in `recepies/`:

```yaml
name: my-recipe
description: My deployment workflow
provider: my-provider

required_fields:
  - name: cluster
    description: Kubernetes cluster name
    type: string
  - name: namespace
    description: Target namespace
    type: string
    default: default  # Optional default

steps:
  - name: build
    command: go build -o dist/app .
    execution_mode: root
    timeout: 300

  - name: deploy
    command: kubectl apply -f k8s/
    execution_mode: root
    timeout: 60
```

See [recepies/README.md](recepies/README.md) for full documentation.

## CLI Reference

### Commands

| Command | Description |
|---------|-------------|
| `pilum deploy [services...]` | Full pipeline: build → push → deploy |
| `pilum build [services...]` | Build only (excludes deploy-tagged steps) |
| `pilum publish [services...]` | Build and push to registry (no deploy) |
| `pilum push [services...]` | Push images to container registry |
| `pilum check` | Validate services against recipe requirements |
| `pilum list` | List discovered services |
| `pilum dry-run` | Preview commands without executing |

### Flags

| Flag | Description |
|------|-------------|
| `--tag` | Version tag for the deployment |
| `--dry-run` | Preview commands without executing |
| `--debug` | Enable debug logging |
| `--timeout` | Command timeout in seconds (default: 300) |
| `--retries` | Number of retries on failure (default: 0) |
| `--max-workers` | Parallel worker count (default: 4) |
| `--only-tags` | Only run steps with these tags (e.g., `--only-tags=deploy`) |
| `--exclude-tags` | Skip steps with these tags (e.g., `--exclude-tags=deploy`) |

### Examples

```bash
# Deploy all services
pilum deploy --tag=v1.0.0

# Deploy specific service
pilum deploy my-api --tag=v1.0.0

# Build only (skip deployment)
pilum build --tag=v1.0.0

# Build and push, but don't deploy
pilum publish --tag=v1.0.0

# Run only deploy-tagged steps (assumes images exist)
pilum deploy --only-tags=deploy --tag=v1.0.0

# Preview what would run
pilum deploy --dry-run --tag=v1.0.0

# Validate all service configurations
pilum check
```

## Project Structure

```
my-project/
├── recepies/
│   ├── gcp-cloud-run-recepie.yaml
│   └── aws-lambda-recepie.yaml
├── services/
│   ├── api/
│   │   ├── service.yaml
│   │   └── main.go
│   └── worker/
│       ├── service.yaml
│       └── main.go
```

## How It Works

1. **Discovery** - Pilum finds all `service.yaml` files in your project
2. **Validation** - Each service is validated against its recipe's required fields
3. **Matching** - Services are matched to recipes based on `provider` field
4. **Orchestration** - Steps execute in order, services run in parallel within steps

```
Step 1: build
  ├── api-gateway     ✓ (1.2s)
  ├── user-service    ✓ (0.9s)
  └── payment-service ✓ (1.1s)

Step 2: push
  ├── api-gateway     ✓ (2.1s)
  ├── user-service    ✓ (1.8s)
  └── payment-service ✓ (2.0s)

Step 3: deploy
  ├── api-gateway     ✓ (3.2s)
  ├── user-service    ✓ (2.9s)
  └── payment-service ✓ (3.1s)
```

## Architecture

| Component | Purpose |
|-----------|---------|
| `cmd/` | CLI commands (Cobra) |
| `lib/recepie/` | Recipe loading and validation |
| `lib/registry/` | Step handler registration |
| `lib/orchestrator/` | Parallel execution engine |
| `ingredients/` | Cloud-specific command generators |
| `recepies/` | Deployment workflow definitions |

For detailed architecture documentation, see [docs/architecture.md](docs/architecture.md).

## Documentation

- [Adding a New Provider](docs/adding-a-provider.md) - Step-by-step guide with Lambda, Fargate, and Azure examples
- [Architecture](docs/architecture.md) - Internal design and extension points
- [Troubleshooting](docs/troubleshooting.md) - Common issues and solutions
- [Recipe Reference](recepies/README.md) - Full recipe configuration guide

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for:

- Development setup
- Code style guidelines
- How to add new providers
- Pull request process

## License

[BSL 1.1](LICENSE) - Converts to Apache 2.0 on December 3, 2028.
