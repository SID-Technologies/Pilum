# Pilum

[![CI](https://github.com/sid-technologies/pilum/actions/workflows/ci-pr.yml/badge.svg)](https://github.com/sid-technologies/pilum/actions/workflows/ci-pr.yml)
[![codecov](https://codecov.io/gh/sid-technologies/pilum/branch/main/graph/badge.svg)](https://codecov.io/gh/sid-technologies/pilum)
[![License](https://img.shields.io/github/license/sid-technologies/pilum)](LICENSE)
[![Release](https://img.shields.io/github/v/release/sid-technologies/pilum)](https://github.com/sid-technologies/pilum/releases)

A cloud-agnostic build and deployment CLI. Define your service once, deploy to any cloud provider.

![Pilum deploy demo](public/demo.gif)

```bash
pilum deploy --tag=v1.0.0
```

Pilum handles the build → push → deploy pipeline while your infrastructure-as-code (Terraform, Pulumi) defines the actual resources.

Pilum is dogfooded as the deployment tool for every [SID Technologies](https://romans.dev) product — dozens of services shipped to Cloud Run, Cloud Run Jobs, Cloudflare Pages, npm, and Homebrew with it, daily, since December 2025.

**Provider status, honestly:**

- **Battle-tested daily**: GCP Cloud Run, GCP Cloud Run Jobs, Cloudflare Pages, npm, Homebrew
- **Implemented and command-level reviewed, less exercised**: AWS Lambda, Azure Container Apps — feedback from real-world use wanted

Not sure Pilum fits your stack? Read [When to Use Pilum (and When Not To)](https://pilum.dev/docs/guides/when-to-use/) — it includes the cases where the answer is no.

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

## Features

- **Recipe-driven deployments** - Define reusable deployment workflows in YAML
- **Recipe-driven validation** - Each recipe declares required fields, no Go code per provider
- **Multi-cloud support** - GCP Cloud Run, AWS Lambda, Azure Container Apps, Homebrew, and more
- **Parallel execution** - Deploy multiple services concurrently with step barriers
- **Dependency ordering** - Wave-based deployment respects `depends_on` across services
- **Deployment locks** - Prevent concurrent deploys with automatic stale lock cleanup
- **GitHub commit status** - Post pending/success/failure status to commits from CI
- **Step filtering** - Run only build steps, only deploy steps, or custom tag combinations
- **Environment overrides** - Per-environment config (staging, prod) via `environments` block
- **Git-aware filtering** - Deploy only services with changes since base branch
- **Dry-run mode** - Preview commands before executing
- **JSON output** - Machine-readable output for scripting and AI agents
- **Editor support** - JSON Schema for autocompletion and validation in VS Code, JetBrains, etc.
- **Beautiful CLI** - Animated spinners, colored output, clear progress

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

This validates your `pilum.yaml` against the recipe's required fields and the [JSON Schema](#editor-support):

- **Errors** (check fails): an unknown key inside a block, such as `cloud_run.cpu_throtling` (with a "did you mean" suggestion), or a value of the wrong type that Pilum would ignore, such as `min_instances: "2"`. `environments` blocks are checked too.
- **Warnings**: an unknown top-level key (custom recipes may read their own; keys a service's recipe names aren't reported), or a number where a string is expected, which Pilum reads as text (`version: 1.20` reads as `1.2`; quote it).

Other commands print the same problems as warnings and carry on.

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
| `gcp-mig-container` | `gcp` | Roll a new container image onto an existing managed instance group (COS + startup script) |
| `aws-lambda` | `aws` | Deploy a zip to an existing Lambda function, publish a version and move its alias |
| `aws-lambda-image` | `aws` | Deploy a container image to an existing Lambda function, publish a version and move its alias |
| `aws-lambda-sam` | `aws` | Deploy a SAM application (`sam build` + `sam deploy`); named `aws-lambda` before v0.12 |
| `aws-ecs` | `aws` | Roll a new image onto an existing Amazon ECS service (Fargate or EC2) |
| `aws-ecr-image` | `aws` | Build a Docker image and publish it to Amazon ECR (no deploy) |
| `azure-container-apps` | `azure` | Deploy to Azure Container Apps |
| `cloudflare-pages` | `cloudflare` | Deploy to Cloudflare Pages |
| `homebrew` | `homebrew` | Publish Homebrew formula |
| `npm` | `npm` | Publish npm packages |

Use `pilum recipes` to list all recipes, or `pilum recipes <name>` to see required fields, optional fields, and deployment steps for a specific recipe.

### Custom Recipes

Create your own recipes in `recipes/`:

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

See [recipes/README.md](recipes/README.md) for full documentation.

## Rollback

Roll a service back to the version before the one currently deployed:

```bash
pilum rollback api --dry-run          # Show what would change
pilum rollback api                    # Asks for confirmation
pilum rollback api --to api-00041-xyz # A specific revision
pilum rollback api --yes              # No prompt (required in CI)
```

- **Cloud Run services** move 100% of traffic to the previous healthy revision. Nothing is rebuilt, and that revision's full config comes back with it. Running rollback again steps back one more revision.
- **Cloud Run jobs** point the job at the image of its previous execution (or `--to <tag>`), keeping the job's current config.
- **Managed instance groups** (`gcp-mig-container`) roll back onto the newest `<template_base>` template older than the current one (or `--to <tag|template>`), with the same surge settings as a deploy, and wait until the group is stable.
- **ECS services** (`aws-ecs`) point the service back at the newest task definition revision before the current one, and wait for the rollout. That revision's whole config comes back with it. Revisions whose rollout failed are skipped: when a deploy fails or the circuit breaker rolls it back, Pilum tags the revision `pilum:rollout=failed` (it needs `ecs:TagResource`). `--to` takes a revision number (`41`), `family:revision`, an ARN, or an image tag.
- **Lambda functions** (`aws-lambda`, `aws-lambda-image`) move the alias (`live`) back to the previous published version. Instant, nothing rebuilt. `--to` takes a version number or the tag it was deployed with.
- The previous version is read from the cloud provider, not from local history, so it works from any machine, including CI.
- The next `pilum deploy` sends traffic to the new revision again: Cloud Run recipes end with a `route traffic to latest` step. This also resets any traffic split set up by hand.

Other providers are not supported yet. The next `pilum deploy` to an ECS service copies the revision it runs, so it builds on what you rolled back to.

## Managed Instance Groups

`gcp-mig-container` ships new versions to a managed instance group that already exists. Pilum doesn't create the MIG, load balancer, health check, firewall rules or Cloud NAT; it builds the image, creates a new instance template for the release, and rolls the group onto it with no downtime.

### How the container runs

The template runs the container the way Google recommends now that the container startup agent is retired: a [Container-Optimized OS](https://cloud.google.com/container-optimized-os/docs) VM whose `startup-script` reads its configuration from instance metadata and runs `docker run`. Templates that still use `gce-container-declaration` (`create-with-container`) are refused, because [GCP stopped creating VMs from them on July 31, 2026](https://docs.cloud.google.com/compute/docs/containers/prepare-for-container-agent-shutdown).

Each release, Pilum copies the MIG's current template and changes only what the service manages:

| Metadata key | Set by Pilum | Holds |
|---|---|---|
| `container-image` (or `mig.image_metadata_key`) | Every release | The image to run. Must exist in the template. |
| `container-env` | When `env_vars` or `mig.env_vars` are set | `KEY=VALUE` lines |
| `container-secrets` | When `secrets` are set | `NAME=projects/<p>/secrets/<s>/versions/<v>` lines. References only, never values |

A startup script that meets the contract:

```bash
#!/bin/bash
# Runs on every boot. Pilum updates the metadata it reads in each release's template.
set -euo pipefail
md() { curl -sf -H 'Metadata-Flavor: Google' "http://metadata.google.internal/computeMetadata/v1/instance/$1"; }

export HOME=/home/app && mkdir -p "$HOME"   # COS: the root filesystem is read-only
IMAGE=$(md attributes/container-image)
ENV_FILE=$HOME/container.env
umask 077
{ md attributes/container-env || true; echo; } > "$ENV_FILE"

# Secrets are read from Secret Manager with the VM's service account,
# which needs roles/secretmanager.secretAccessor on them.
SECRETS=$(md attributes/container-secrets || true)
if [ -n "$SECRETS" ]; then
  TOKEN=$(md service-accounts/default/token | tr -d '\n ' | sed -E 's/.*"access_token":"([^"]+)".*/\1/')
  while IFS='=' read -r name ref; do
    [ -n "$name" ] || continue
    value=$(curl -sf -H "Authorization: Bearer $TOKEN" "https://secretmanager.googleapis.com/v1/$ref:access" \
      | tr -d '\n ' | sed -E 's/.*"data":"([^"]+)".*/\1/' | base64 -d)
    printf '%s=%s\n' "$name" "$value" >> "$ENV_FILE"
  done <<< "$SECRETS"
fi

docker-credential-gcr configure-docker --registries="${IMAGE%%/*}"
docker rm -f app 2>/dev/null || true
docker run -d --name app --restart=always --network host --env-file "$ENV_FILE" "$IMAGE"
```

Env values and secrets must be single-line, since they go through an env file.

### Configuration

```yaml
name: statio-egress-proxy
type: gcp-mig-container
project: statio-499700
region: us-central1
image: us-central1-docker.pkg.dev/statio-499700/statio/statio-egress-proxy  # no tag; it comes from --tag
template: egress-proxy.dockerfile

env_vars:                         # -> container-env
  PORT: "3128"
secrets:                          # -> container-secrets: name, name:version, or a full path
  UPSTREAM_TOKEN: upstream-token:latest

mig:
  name: egress-proxy-mig
  zone: us-central1-a             # or region: for a regional MIG
  template_base: egress-proxy-tpl # each release creates <base>-<tag>

  # Optional. Unset keeps what the current template has.
  machine_type: e2-small
  subnet: proxy-subnet            # in the MIG's region; or projects/<host>/regions/<r>/subnetworks/<s> for Shared VPC
  external_ip: false              # outbound traffic then goes through Cloud NAT
  network_tags: [egress-proxy]    # firewall rules target these
  service_account: egress-proxy@statio-499700.iam.gserviceaccount.com

  # Rollout. Defaults shown.
  max_surge: 1                    # 3 for regional MIGs (must be 0 or at least the zone count)
  max_unavailable: 0
  wait_timeout: 600               # seconds; the deploy fails if the group isn't stable by then
```

VM settings left unset are copied from the current template, so a MIG set up by Terraform or a script keeps its configuration. Settings Pilum does manage replace the template's: removing every env var from `pilum.yaml` leaves the last `container-env` in place, so delete it from the template if you need it gone. There's no per-instance static IP, because MIG instances are replaced on every rollout; for a fixed outbound IP, reserve one on the Cloud NAT gateway.

### Deploying

`pilum deploy statio-egress-proxy --tag v1.4.2` builds and pushes the image, creates `egress-proxy-tpl-v1-4-2` from the current template, starts a rolling update (`--max-surge 1 --max-unavailable 0`), and fails unless the group is stable within `mig.wait_timeout`. Each release needs its own tag; `latest` is rejected. `pilum rollback` moves the group back to the previous template, and `pilum status` shows the template it runs.

## AWS

Every AWS recipe reads a shared `aws:` block:

```yaml
provider: aws
type: aws-ecr-image
region: us-east-1
template: go

aws:
  account_id: "123456789012"   # owns the ECR registry; quote it, YAML drops leading zeros
  profile: prod                # optional: --profile on every aws command
  create_repository: true      # optional: create the ECR repository on first push
```

Images go to `<account_id>.dkr.ecr.<region>.amazonaws.com/<image_name or name>`. Pilum logs docker in to ECR before it pushes, and before it builds when `build.docker_cache` is on, since the cache lives in ECR too. Without `profile`, the aws CLI's own credentials apply: env vars, `AWS_PROFILE`, or an OIDC role from `aws-actions/configure-aws-credentials` in CI. Leave `create_repository` off when Terraform owns the repositories.

Services that set the account in `registry_name` keep working.

### ECS

`aws-ecs` rolls a new image onto an existing ECS service, Fargate or EC2. Terraform (or whoever set it up) owns the cluster, service, load balancer and IAM roles; Pilum ships versions:

```yaml
name: api
provider: aws
type: aws-ecs
region: us-east-1
template: go
aws:
  account_id: "123456789012"
ecs:
  cluster: prod
  # service: api          # default: name
  # container: api        # default: the only container, or the one named like the service
  # cpu: 512              # task size; unset keeps the task definition's
  # memory: 1024
  # desired_count: 2      # unset leaves it to you or autoscaling
  # wait_timeout: 600
env_vars:
  LOG_LEVEL: info         # when set, replaces the container's environment
secrets:
  DB_URL: prod/api/db-url # Secrets Manager secret or SSM parameter, name or ARN
```

Each deploy copies the task definition the service runs now, changes the image and only what `pilum.yaml` sets, registers it as the next revision and points the service at it. Ports, health checks, log configuration, roles and other containers come along unchanged. Pilum then waits for the rollout and fails the deploy if it fails, times out, or is rolled back by the deployment circuit breaker (`aws ecs wait services-stable` calls that a success). The last service events are in the error. A rerun reuses a revision that already matches instead of registering another.

If Terraform manages the task definition too, give it `lifecycle { ignore_changes = [task_definition] }` on the service, so the next `terraform apply` doesn't roll back what Pilum deployed.

`pilum status` shows the revision and running tasks; `pilum logs` tails `ecs.log_group` (default `/ecs/<service>`).

### Lambda

`aws-lambda` (zip) and `aws-lambda-image` (container image) deploy to an existing function. Terraform owns the function, its role and its triggers; Pilum updates the code and what `pilum.yaml` sets, publishes a version, and moves an alias to it:

```yaml
name: thumbnails
provider: aws
type: aws-lambda          # or aws-lambda-image, with template: and aws.account_id
region: us-east-1
build:
  cmd: GOOS=linux GOARCH=arm64 go build -o dist/bootstrap .
lambda:
  # function: thumbnails  # default: name
  # alias: live           # default
  artifact: dist          # aws-lambda: a directory (zipped) or a .zip
  runtime: provided.al2023
  handler: bootstrap
  architecture: arm64     # aws-lambda-image builds linux/arm64 too
  memory: 512
  timeout: 30
  # s3_bucket: my-artifacts  # needed for zips over 50 MB
env_vars:
  LOG_LEVEL: info         # when set, replaces the function's environment
```

**Point triggers at `<function>:live`**, not the bare function name. Then a deploy switches every caller to the new version in one step, and `pilum rollback` moves the alias back instantly. Callers of the bare name run `$LATEST`, which a deploy updates and a rollback doesn't touch. On the first deploy Pilum creates the alias; if Terraform manages it, ignore changes to its `function_version`.

Each step waits for Lambda to finish the update before the next starts, and a failed update fails the deploy before the alias moves. Zips are built deterministically, so an unchanged build publishes no new version. Images are built with `--provenance=false`, since Lambda rejects the image index BuildKit makes for provenance attestations. Lambda has no secret references like ECS, so `secrets:` is refused: read them at run time.

`aws-lambda` used to be the SAM recipe. A service with a `template.yaml` next to it gets a pointer to `type: aws-lambda-sam`.

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

## Cloud Run: Port and HTTP/2

Both options are opt-in; leaving them out passes nothing to `gcloud`, so existing services keep their current settings.

```yaml
cloud_run:
  port: 8080       # --port; always sent with sidecars (default 8080), otherwise only when set
  use_http2: true  # --use-http2: end-to-end HTTP/2 (h2c) to the ingress container
```

With `use_http2`, the container must serve cleartext HTTP/2 (h2c). Removing the key later does not turn HTTP/2 off; run `gcloud run services update <name> --no-use-http2`. Both work for `gcp-cloud-run` and `gcp-cloud-run-from-image`.

## Cloud Run: Service Identity

`service_account` sets the identity the service's revisions run as (`--service-account`). Without it, a new service runs as the project's default compute account, which has no access to your secrets, so a first deploy that mounts `secrets:` fails with "Permission denied on secret".

```yaml
cloud_run:
  service_account: api-runtime@my-project.iam.gserviceaccount.com
```

The account needs `roles/secretmanager.secretAccessor` on each mounted secret, and whoever runs `pilum deploy` needs `roles/iam.serviceAccountUser` on the account. Leaving the key out passes nothing, so an existing service keeps its current identity; removing it later does not revert to the default. Works for `gcp-cloud-run` and `gcp-cloud-run-from-image`, including deploys with `sidecars:` (the identity applies to every container in the revision). Cloud Run jobs use `job.service_account`.

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

## Warming Build Caches

Parallel builds against a cold cache each compile the same shared dependencies: four Go services building at once compile the same modules four times. `build.warm` fills the cache once, before any service builds:

```yaml
# services/*/pilum.yaml (Go)
build:
  language: go
  warm: go build ./...
  cmd: go build -o ./dist .
  env_vars:
    GOOS: linux
    GOARCH: amd64

# apps/*/pilum.yaml (Node)
build:
  language: node
  warm: pnpm install --frozen-lockfile
  cmd: pnpm build
```

```
━━━ Warming build caches ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ✓ go (10 services)    41.2s
  ✓ node (3 services)   12.8s
```

- **Once per command, not per service.** Services with the same `warm`, `warm_dir` and build `env_vars` share one run. The env is part of the match because a cache filled for one `GOOS`/`GOARCH` misses for another.
- **Every language before any build.** Distinct warm commands run concurrently, and the build step starts when all have finished.
- **Runs from the project root.** Set `warm_dir` (relative to the root) when a service's module or lockfile lives elsewhere.
- **Only when building.** Skipped when no build step runs (`--only-tags deploy`), and with `--no-warm`.
- **Best effort.** A failed warm is reported and the builds still run, cold, so any real error surfaces from the build itself.
- **Skipped when the cache already has it (Go).** Before warming Go services automatically, Pilum asks `go list -deps` which packages are stale, and skips the warm when every package two or more services share is already cached: `go (18 services, auto) (462ms, skipped: shared packages already cached)`. Packages only one service uses are that build's work either way.

Use the same `env_vars` and build tags the build uses, so the entries the warm run creates are the ones the build looks up. With a warm cache, the warm run takes about a second.

### Docker layer cache

`build.docker_cache: true` keeps an image's layer cache in its registry (`<image>:buildcache`), so a fresh CI runner reuses unchanged layers. It needs a buildx builder that can export cache, such as the `docker-container` driver from `docker/setup-buildx-action`; with the default builder Pilum warns and builds without it. To turn it on for every service, set it once in `.pilum.yml`:

```yaml
# .pilum.yml
build:
  docker_cache: true
```

A service's own `build.docker_cache` in `pilum.yaml`, `true` or `false`, wins over the project default.

## Editor Support

Pilum provides a JSON Schema for `pilum.yaml` that enables autocompletion, validation, and inline documentation in supported editors. `pilum check` validates against the same schema, built into the binary.

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
| `pilum rollback <services...>` | | Roll back to the previous version (Cloud Run) |

### Configuration Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `pilum init` | | Generate a new `pilum.yaml` interactively |
| `pilum check [services...]` | `validate` | Validate configs against recipe requirements and the schema |
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
| `--timeout` | `-T` | `60` | Step timeout in seconds. A recipe step's `timeout:` replaces the default; passing `-T` replaces both |
| `--retries` | `-r` | `3` | Number of retries on failure |
| `--max-workers` | | `0` (auto) | Parallel workers. Auto: build steps use CPUs ÷ `resources.cpu` (default 2), capped by memory and sized from the Docker VM for docker steps; push/deploy steps use up to 16 |
| `--only-tags` | | | Only run steps with these tags |
| `--exclude-tags` | | | Exclude steps with these tags |
| `--env` | `-e` | | Environment to apply (merges overrides) |
| `--provider` | | | Filter services by provider |
| `--only-changed` | | `false` | Only deploy services with changes since base branch |
| `--since` | | | Git ref to compare against (default: main or master) |
| `--no-deps` | | `false` | Disable dependency-based deployment ordering |
| `--no-warm` | | `false` | Skip `build.warm`, the shared cache fill before builds |
| `--force` | `-f` | `false` | Override deployment lock |
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
│   └── deploy.lock            # Deployment lock (temporary)
├── _templates/                # Dockerfile templates
├── recipes/                  # Recipe definitions (YAML)
│   ├── gcp-cloud-run-recipe.yaml
│   ├── aws-lambda-recipe.yaml
│   └── ...
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

1. **Discovery** - Pilum finds all `pilum.yaml` files in your project (respects `.gitignore` files at any level, each scoped to its directory, and `.pilumignore`)
2. **Validation** - Each service is validated against its recipe's required fields
3. **Matching** - Services are matched to recipes based on `provider` field
4. **Locking** - A deployment lock prevents concurrent runs
5. **Orchestration** - Steps execute in order; services run in parallel within steps
6. **Recording** - Results are saved to `.pilum/history.jsonl`

```
Step 1: build
  ├── api-gateway     ✓ (1.3s)
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
| `lib/recipe/` | Recipe loading and validation |
| `lib/registry/` | Step handler registration |
| `lib/orchestrator/` | Parallel execution engine |
| `lib/service_info/` | Service discovery and filtering |
| `lib/lock/` | Deployment lock management |
| `lib/ci/` | CI integrations (GitHub commit status) |
| `lib/history/` | Deployment history recording |
| `lib/git/` | Git-aware change detection |
| `ingredients/` | Cloud-specific command generators |
| `recipes/` | Deployment workflow definitions |
| `schemas/` | JSON Schema for editor support |

## Troubleshooting

### `failed to compute cache key: "…/dist": not found` on Docker build step

**Cause:** the Docker build daemon can't find a path the Dockerfile is trying to `COPY` — even though it exists on disk. The usual culprit is `.dockerignore` filtering the path out of the build context before it reaches the daemon.

Pilum's Go pipeline compiles binaries into `services/<name>/dist/` in the build step, then the golang-api Dockerfile does `COPY ./services/<name>/dist .` in the image step. If your repo's `.dockerignore` contains `**/dist`, `dist/`, or any pattern that matches those directories, they're stripped from the context and the `COPY` fails with:

```
ERROR: failed to compute cache key: failed to calculate checksum of ref …::…: "/services/<name>/dist": not found
```

Docker also emits (easy to miss) a warning line:

```
1 warning found:
 - CopyIgnoredFile: Attempting to Copy file "services/<name>/dist" that is excluded by .dockerignore (line N)
```

**Fix:** remove the `dist` pattern from `.dockerignore`, or narrow it to only frontend build outputs (e.g., `apps/web/dist`, `apps/desktop/dist`) rather than the repo-wide `**/dist`.

**Mental model.** `.gitignore` controls what git tracks. `.dockerignore` controls what Docker ships to the daemon at build time — files can exist on disk but be invisible to the build. If a Dockerfile `COPY`s a path, that path MUST NOT match anything in `.dockerignore`, even if it's a build artifact. Two-strategy repos (one flow compiles inside the image, another compiles outside) need to be careful — the `dist` you ignore for one strategy is the `dist` the other needs.

## Documentation

Full documentation available at **[pilum.dev/docs](https://pilum.dev/docs/getting-started/introduction/)**

- [Getting Started](https://pilum.dev/docs/getting-started/introduction/) - Introduction and quick start
- [When to Use Pilum (and When Not To)](https://pilum.dev/docs/guides/when-to-use/) - An honest fit guide
- [Pilum vs. GCP Native Tooling](https://pilum.dev/docs/comparisons/pilum-vs-gcp-native/) / [Pilum vs. CI/CD Pipelines](https://pilum.dev/docs/comparisons/pilum-vs-ci-cd/) - Comparisons
- [FAQ](https://pilum.dev/docs/reference/faq/) - Security, maintenance, lock-in, Terraform/CI relationships
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

## Security

Found a vulnerability? See [SECURITY.md](SECURITY.md) for how to report it privately.

## License

[Apache License 2.0](LICENSE)
