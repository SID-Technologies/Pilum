# Pilum

[![Go Tests](https://github.com/sid-technologies/pilum/actions/workflows/ci-pr.yml/badge.svg)](https://github.com/sid-technologies/pilum/actions/workflows/ci-pr.yml)
[![codecov](https://codecov.io/gh/sid-technologies/pilum/branch/main/graph/badge.svg)](https://codecov.io/gh/sid-technologies/pilum)

A cloud-agnostic build and deployment CLI. Define your service once, deploy to any cloud provider.

```bash
pilum deploy --tag=v1.0.0
```

Pilum handles the build → push → deploy pipeline while your infrastructure-as-code (Terraform, Pulumi) defines the actual resources.

## Features

- **Recipe-driven deployments** - Define reusable deployment workflows in YAML
- **Recipe-driven validation** - Each recipe declares required fields, no Go code per provider
- **Multi-cloud support** - GCP Cloud Run, AWS Lambda, Azure Container Apps, Homebrew, and more
- **Parallel execution** - Deploy multiple services concurrently with step barriers
- **Dependency ordering** - Wave-based deployment respects `depends_on` across services
- **Build caching** - Hash-based skip for unchanged services (SHA-256 content hashing)
- **Deployment locks** - Prevent concurrent deploys with automatic stale lock cleanup
- **GitHub commit status** - Post pending/success/failure status to commits from CI
- **Step filtering** - Run only build steps, only deploy steps, or custom tag combinations
- **Environment overrides** - Per-environment config (staging, prod) via `environments` block
- **Git-aware filtering** - Deploy only services with changes since base branch
- **Dry-run mode** - Preview commands before executing
- **JSON output** - Machine-readable output for scripting and AI agents
- **Editor support** - JSON Schema for autocompletion and validation in VS Code, JetBrains, etc.
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

### 1. Initialize a service (optional)

Use the interactive init command to generate a `pilum.yaml`:

```bash
pilum init
```

This walks you through:
1. Selecting a provider (GCP, AWS, Homebrew, etc.)
2. Selecting a service type (Cloud Run, Lambda, etc.)
3. Filling in required and optional fields
4. Choosing a build language (Go, Python, Rust, Node)

### 2. Or create a service configuration manually

Create a `pilum.yaml` in your project:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/SID-Technologies/pilum/main/schemas/pilum.yaml.schema.json
name: my-api
provider: gcp
project: my-gcp-project
region: us-central1
registry_name: gcr.io/my-gcp-project

build:
  language: go
  version: "1.23"
```

### 3. Validate your configuration

```bash
pilum check
```

This validates your `pilum.yaml` against the recipe's required fields.

### 4. Deploy

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

Pilum uses a cooking metaphor: **recipes** define deployment workflows, **ingredients** generate cloud-specific commands.

Each recipe defines:
- **Steps** - Ordered commands to execute (build, push, deploy)
- **Required fields** - What your `pilum.yaml` must contain
- **Optional fields** - Additional configuration options with defaults

### Built-in Recipes

| Recipe | Provider | Description |
|--------|----------|-------------|
| `gcp-cloud-run` | `gcp` | Deploy to Google Cloud Run |
| `gcp-cloud-run-job` | `gcp` | Deploy batch jobs to Google Cloud Run Jobs |
| `aws-lambda` | `aws` | Deploy to AWS Lambda |
| `azure-container-apps` | `azure` | Deploy to Azure Container Apps |
| `cloudflare-pages` | `cloudflare` | Deploy to Cloudflare Pages |
| `homebrew` | `homebrew` | Publish Homebrew formula |
| `npm` | `npm` | Publish npm packages |

Use `pilum recipes` to list all recipes, or `pilum recipes <name>` to see required fields, optional fields, and deployment steps for a specific recipe.

### Custom Recipes

Create your own recipes in `recepies/`:

```yaml
name: my-recipe
description: My deployment workflow
provider: my-provider
service: my-service    # Required - service type identifier

required_fields:
  - name: cluster
    description: Kubernetes cluster name
    type: string
  - name: namespace
    description: Target namespace
    type: string
    default: default  # Optional default

optional_fields:
  - name: replicas
    description: Number of replicas
    type: int
    default: "1"

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

## Build Caching

Pilum caches build results based on content hashes. If no files in a service directory have changed since the last build with the same tag, the build step is skipped automatically.

```bash
# First run builds everything
pilum deploy --tag=v1.0.0

# Second run skips unchanged services
pilum deploy --tag=v1.0.0

# Force rebuild (ignore cache)
pilum deploy --tag=v1.0.0 --no-cache
```

Cache state is stored in `.pilum/build-cache.json`. Hashing uses `git ls-files` to respect `.gitignore`, falling back to filesystem walk for non-git projects.

## Deployment Locks

Pilum prevents concurrent deployments using a lock file at `.pilum/deploy.lock`. If you run `pilum deploy` while another deploy is in progress, the second invocation will fail with details about who holds the lock.

```bash
# Fails if another deploy is running
pilum deploy --tag=v1.0.0
# ✗ deployment locked by PID 12345 on my-host (started 2026-02-13T10:00:00Z, command: deploy)

# Override the lock
pilum deploy --tag=v1.0.0 --force
```

Locks are automatically cleaned up when:
- The holding process has exited (dead PID detection)
- The lock is older than 1 hour (stale timeout)

Dry-run mode does not acquire locks.

## CI Integration

### GitHub Commit Status

Post deployment status to GitHub commits directly from your CI pipeline:

```bash
pilum deploy --tag=v1.0.0 --github-status
```

This posts `pending` before the run, then `success` or `failure` after, using the `pilum/<command>` context (e.g., `pilum/deploy`).

Required environment variables (set automatically in GitHub Actions):
- `GITHUB_ACTIONS=true`
- `GITHUB_TOKEN` - GitHub token with `repo:status` permission
- `GITHUB_SHA` - Commit SHA to update
- `GITHUB_REPOSITORY` - Repository in `owner/repo` format

### GitHub Actions Example

```yaml
- name: Deploy
  run: pilum deploy --tag=${{ github.sha }} --github-status
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Environment Overrides

Define per-environment configuration that merges with your base config:

```yaml
name: api
provider: gcp
project: acme-dev
region: us-central1

environments:
  staging:
    project: acme-staging
  prod:
    project: acme-prod
    region: us-east1
    cloud_run:
      min_instances: 2
```

```bash
# Deploy with prod overrides
pilum deploy --env=prod --tag=v1.0.0
```

Scalar values are replaced, maps are deep-merged, arrays are replaced entirely.

## Dependency Ordering

Services can declare dependencies. Deploy steps execute in waves - dependencies deploy first:

```yaml
# db/pilum.yaml
name: db-migration
provider: gcp

# api/pilum.yaml
name: api
provider: gcp
depends_on:
  - db-migration
```

```
Wave 1: db-migration  ✓ (3.2s)
Wave 2: api           ✓ (2.9s)
```

Build steps always run in flat parallel (no wave ordering). Use `--no-deps` to disable.

## Editor Support

Pilum provides a JSON Schema for `pilum.yaml` that enables autocompletion, validation, and inline documentation in supported editors.

### VS Code (with YAML extension)

Add this comment to the top of your `pilum.yaml`:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/SID-Technologies/pilum/main/schemas/pilum.yaml.schema.json
name: my-service
provider: gcp
```

### JetBrains IDEs

Map the schema URL to `pilum.yaml` files in **Settings > Languages & Frameworks > Schemas and DTDs > JSON Schema Mappings**.

## CLI Reference

### Pipeline Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `pilum deploy [services...]` | `up` | Full pipeline: build, push, and deploy |
| `pilum build [services...]` | `b`, `make` | Build step only |
| `pilum publish [services...]` | `p` | Build and push (no deploy) |
| `pilum push [services...]` | `ps` | Push images to registry only |

### Configuration Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `pilum init` | | Generate a new `pilum.yaml` interactively |
| `pilum check [services...]` | `validate` | Validate configs against recipe requirements |
| `pilum list` | `ls` | List all discovered services |
| `pilum recipes [name]` | `providers` | List recipes or describe a specific one |
| `pilum dry-run [services...]` | `dr` | Preview commands without executing |

### Observability Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `pilum status [services...]` | `st` | Show status of deployed services |
| `pilum logs <service>` | `log` | Show logs for a deployed service |
| `pilum history` | `hist` | Show deployment history |

### Utility Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `pilum delete-builds [services...]` | `clean` | Delete `dist/` directories |
| `pilum completion` | | Generate shell completion script |

### Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--verbose` | `-v` | Stream command output in real-time |
| `--quiet` | `-q` | Minimal output (CI-friendly) |
| `--json` | | Output as JSON for scripting |
| `--no-gitignore` | | Don't read `.gitignore` for ignore patterns |

### Pipeline Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--tag` | `-t` | `latest` | Version tag for the deployment |
| `--dry-run` | `-D` | `false` | Preview commands without executing |
| `--debug` | `-d` | `false` | Enable debug logging |
| `--timeout` | `-T` | `60` | Command timeout in seconds |
| `--retries` | `-r` | `3` | Number of retries on failure |
| `--max-workers` | | `0` (auto) | Maximum parallel workers |
| `--only-tags` | | | Only run steps with these tags |
| `--exclude-tags` | | | Exclude steps with these tags |
| `--env` | `-e` | | Environment to apply (merges overrides) |
| `--provider` | | | Filter services by provider |
| `--only-changed` | | `false` | Only deploy services with changes since base branch |
| `--since` | | | Git ref to compare against (default: main or master) |
| `--no-deps` | | `false` | Disable dependency-based deployment ordering |
| `--force` | `-f` | `false` | Override deployment lock |
| `--no-cache` | | `false` | Disable build caching (force rebuild) |
| `--github-status` | | `false` | Post commit status to GitHub |

### Examples

```bash
# Initialize a new service (interactive)
pilum init

# Validate all service configurations
pilum check

# Explore available recipes and their fields
pilum recipes
pilum recipes gcp-cloud-run

# Deploy all services
pilum deploy --tag=v1.0.0

# Deploy specific service
pilum deploy my-api --tag=v1.0.0

# Deploy with environment overrides
pilum deploy --env=prod --tag=v1.0.0

# Deploy only changed services
pilum deploy --only-changed --tag=v1.0.0

# Deploy only GCP services
pilum deploy --provider=gcp --tag=v1.0.0

# Build only (skip deployment)
pilum build --tag=v1.0.0

# Build and push, but don't deploy
pilum publish --tag=v1.0.0

# Run only deploy-tagged steps (assumes images exist)
pilum deploy --only-tags=deploy --tag=v1.0.0

# Force rebuild (skip cache)
pilum deploy --tag=v1.0.0 --no-cache

# Override an existing deployment lock
pilum deploy --tag=v1.0.0 --force

# Post GitHub commit status from CI
pilum deploy --tag=v1.0.0 --github-status

# Preview what would run
pilum dry-run --tag=v1.0.0

# Check status of deployed services
pilum status
pilum status my-api

# Stream logs
pilum logs my-api --follow

# View deployment history
pilum history --limit=20

# JSON output (for scripting / AI agents)
pilum list --json
pilum recipes gcp-cloud-run --json
pilum status --json
```

## Project Structure

```
my-project/
├── .pilum/                    # State directory (gitignored)
│   ├── history.jsonl          # Deployment history
│   ├── build-cache.json       # Build cache hashes
│   └── deploy.lock            # Deployment lock (temporary)
├── _templates/                # Dockerfile templates
├── recepies/                  # Recipe definitions (YAML)
│   ├── gcp-cloud-run-recepie.yaml
│   └── aws-lambda-recepie.yaml
├── schemas/                   # JSON Schema for editor support
│   └── pilum.yaml.schema.json
├── services/
│   ├── api/
│   │   ├── pilum.yaml
│   │   └── main.go
│   └── worker/
│       ├── pilum.yaml
│       └── main.go
└── .pilumignore               # Additional ignore patterns
```

## How It Works

1. **Discovery** - Pilum finds all `pilum.yaml` files in your project (respects `.gitignore` and `.pilumignore`)
2. **Validation** - Each service is validated against its recipe's required fields
3. **Matching** - Services are matched to recipes based on `provider` field
4. **Locking** - A deployment lock prevents concurrent runs
5. **Orchestration** - Steps execute in order; services run in parallel within steps (build steps skip if cached)
6. **Recording** - Results are saved to `.pilum/history.jsonl`

```
Step 1: build
  ├── api-gateway     ✓ cached (0.0s)
  ├── user-service    ✓ (1.2s)
  └── payment-service ✓ (1.1s)

Step 2: push
  ├── api-gateway     ✓ (2.1s)
  ├── user-service    ✓ (1.8s)
  └── payment-service ✓ (2.0s)

Step 3: deploy (wave-ordered)
  Wave 1: database    ✓ (3.2s)
  Wave 2: api-gateway ✓ (2.9s)
          user-service ✓ (3.1s)
```

## Architecture

| Component | Purpose |
|-----------|---------|
| `cmd/` | CLI commands (Cobra) |
| `lib/recepie/` | Recipe loading and validation |
| `lib/registry/` | Step handler registration |
| `lib/orchestrator/` | Parallel execution engine |
| `lib/service_info/` | Service discovery and filtering |
| `lib/cache/` | Hash-based build caching |
| `lib/lock/` | Deployment lock management |
| `lib/ci/` | CI integrations (GitHub commit status) |
| `lib/history/` | Deployment history recording |
| `lib/git/` | Git-aware change detection |
| `ingredients/` | Cloud-specific command generators |
| `recepies/` | Deployment workflow definitions |
| `schemas/` | JSON Schema for editor support |

## Documentation

Full documentation available at **[pilum.dev/docs](https://pilum.dev/docs/getting-started/introduction/)**

- [Getting Started](https://pilum.dev/docs/getting-started/introduction/) - Introduction and quick start
- [Service Configuration](https://pilum.dev/docs/configuration/service-yaml/) - Full `pilum.yaml` reference
- [Build Config](https://pilum.dev/docs/configuration/build-config/) - Language-specific build settings
- [Multi-Service Monorepo](https://pilum.dev/docs/guides/multi-service-monorepo/) - Managing multiple services
- [How Recipes Work](https://pilum.dev/docs/recipes/how-recipes-work/) - Recipe system deep dive
- [CLI Commands](https://pilum.dev/docs/reference/cli/) - Complete CLI reference
- [Adding a Provider](https://pilum.dev/docs/providers/adding-a-provider/) - Extend Pilum with new providers
- [Troubleshooting](https://pilum.dev/docs/reference/troubleshooting/) - Common issues and solutions

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for:

- Development setup
- Code style guidelines
- How to add new providers
- Pull request process

## License

[BSL 1.1](LICENSE) - Converts to Apache 2.0 on December 3, 2028.
