# Pilum

A multi-service deployment orchestrator. Deploy to any cloud with simple YAML recipes.

```
pilum deploy --tag=v1.0.0
```

## Features

- **Recipe-driven deployments** - Define reusable deployment workflows in YAML
- **Recipe-driven validation** - Each recipe declares required fields, no Go code per provider
- **Multi-cloud support** - GCP Cloud Run, AWS Lambda, Kubernetes, Homebrew, and more
- **Parallel execution** - Deploy multiple services concurrently with step barriers
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

```
pilum deploy [services...] [flags]    Deploy services
pilum publish [services...] [flags]   Build and push (skip deploy steps)
pilum check                           Validate services against recipes
pilum add <template>                  Add a new service from template
pilum list                            List available templates
pilum dry-run                         Preview what would execute

Flags:
  --tag          Version tag for the deployment
  --dry-run      Preview commands without executing
  --debug        Enable debug logging
  --timeout      Command timeout in seconds (default: 300)
  --retries      Number of retries on failure (default: 0)
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
| `ingredients/` | Cloud-specific command generators |
| `recepies/` | Deployment workflow definitions |

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

Apache
