# Pilum

A multi-service deployment orchestrator. Deploy to any cloud with simple YAML recipes.

```
pilum deploy --tag=v1.0.0
```

## Features

- **Recipe-driven deployments** - Define reusable deployment workflows in YAML
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
type: gcp-cloudrun

build:
  language: go
  version: "1.23"

deploy:
  project: my-gcp-project
  region: us-central1
```

### 2. Deploy

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

Pilum uses recipes to define deployment workflows. Each recipe is a series of steps that run in order, with services executing in parallel within each step.

### Built-in Recipes

| Recipe | Description |
|--------|-------------|
| `gcp-cloudrun` | Build, push to GCR, deploy to Cloud Run |
| `aws-lambda` | Build, package, deploy to Lambda |
| `homebrew` | Build binaries, create archives, update tap |

### Custom Recipes

Create your own recipes in a `recipes/` directory:

```yaml
name: my-custom-recipe
description: My deployment workflow

steps:
  - name: build
    command: go build -o dist/app .
    execution_mode: root
    timeout: 300

  - name: test
    command: go test ./...
    execution_mode: root
    timeout: 120

  - name: deploy
    command: kubectl apply -f k8s/
    execution_mode: root
    timeout: 60
    tags:
      - deploy
```

## CLI Reference

```
pilum deploy [services...] [flags]    Deploy services
pilum publish [services...] [flags]   Build and push (skip deploy steps)
pilum add <service>                   Add a new service
pilum list                            List discovered services

Flags:
  --tag          Version tag for the deployment
  --dry-run      Preview commands without executing
  --debug        Enable debug logging
  --timeout      Command timeout in seconds (default: 300)
  --retries      Number of retries on failure (default: 0)
  --recipe-path  Path to recipes directory (default: ./recipes)
```

## Project Structure

```
my-project/
├── recipes/
│   ├── gcp-cloudrun.yaml
│   └── aws-lambda.yaml
├── services/
│   ├── api/
│   │   ├── service.yaml
│   │   └── main.go
│   └── worker/
│       ├── service.yaml
│       └── main.go
└── pilum.yaml          # Optional global config
```

## How It Works

1. **Discovery** - Pilum finds all `service.yaml` files in your project
2. **Matching** - Each service is matched to a recipe based on its `type`
3. **Orchestration** - Steps execute in order, services run in parallel within steps
4. **Barriers** - All services must complete a step before the next step begins

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

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

MIT
