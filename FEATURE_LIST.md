# Pilum Feature Roadmap

## Current State (v0.1)

- [x] Multi-service orchestration with step barriers
- [x] Parallel execution within steps
- [x] Recipe-driven YAML configuration
- [x] GCP Cloud Run recipe
- [x] AWS Lambda recipe
- [x] Homebrew release recipe
- [x] Dry-run mode
- [x] Publish mode (build + push, no deploy)
- [x] Retry with exponential backoff
- [x] Animated spinners and colored output
- [x] Semantic color theming
- [x] 83% test coverage with unit + E2E tests
- [x] Codecov integration for CI coverage tracking
- [x] Multi-region deployments (`regions: [us-central1, europe-west1]`)

---

## Phase 1: Foundation (Pre-Launch) ✅

### CLI Polish
- [x] `pilum init` - Interactive scaffolding for new services
- [x] `pilum validate` - Validate pilum.yaml and recipe syntax (alias: `check`)
- [x] `pilum list` - List all discovered services and their recipes
- [x] `--verbose / -v` - Stream command stdout/stderr in real-time
- [x] `--quiet / -q` - Minimal output (CI-friendly)
- [x] `--json` - JSON output for scripting/automation
- [x] Environment variable substitution in recipes (`${VAR}`)
- [x] Better error messages with suggestions ("did you mean X?")
- [x] **Wave-based deployment ordering** - Deploy/execute steps respect `depends_on`, build/push remain flat parallel
- [x] **npm recipe** - Build and publish npm packages to any registry
- [x] **GCP Cloud Run Jobs** - Recipe + handlers for batch/migration workloads
- [x] **Cloudflare Pages recipe** - Build and deploy static sites via wrangler

---

## Phase 2: Visibility & Safety ✅

### Status & Observability
- [x] `pilum status` - Show deployed service versions, health, and last deploy time
- [x] `pilum logs [service]` - Tail logs from deployed services (wraps provider-specific log commands)
- [x] `pilum history` - View deployment history (JSONL, project-local)

### Monorepo Support
- [x] `--only-changed` flag - Detect git changes, deploy only affected services
- [x] `--since` flag - Specify git ref to compare against (default: main/master)
- [x] Dependency graph between services (`depends_on` in pilum.yaml)

### Deployment Safety
- [x] Deployment locks (prevent concurrent deploys to same service)

### Environment Management
- [x] Environment configs (`--env prod` / `--env staging`)
- [x] Per-environment overrides in pilum.yaml

---

## Phase 3: AI & Automation Friendliness ✅

### Structured Output
- [x] **Complete JSON output for all commands** - All commands (`list`, `check`, `init`, `delete-builds`, `dry-run`) return structured JSON via `--json`. Centralized `withJSON` wrapper pattern.
- [x] **Distinct exit codes** - Config=2, NoServices=3, Deploy=4, IO=5, InvalidArgs=6, Lock=7. Defined in `lib/exitcodes/`.

### Machine-Readable Config
- [x] **JSON Schema for `pilum.yaml`** - Published schema for editor autocompletion and validation (`schemas/pilum.yaml.schema.json`).
- [x] **Non-interactive `pilum init`** - `pilum init --provider=cloudflare --service=pages --name=my-site --language=node` generates a complete `pilum.yaml` without prompting.

### CI/CD Integration
- [x] GitHub Actions (official action: `pilum-action`)
- [x] GitHub commit status updates (`--github-status` flag, auto-detects in GitHub Actions)

### Advanced Monorepo
- [x] Parallel builds with dependency ordering (wave-based execution)
- [x] Pattern matching for service selection (`pilum deploy "api-*"`)
- [x] Filter services by provider (`--provider=gcp`)
- [x] Environment variable substitution in pilum.yaml (`${GCP_PROJECT}`)

---

## Pre-v1: Polish & Reliability

### Signal Handling
- [x] SIGINT/SIGTERM handler for clean lock release — signal handler in `runPipeline()` releases the lock immediately on Ctrl+C or `kill`, instead of waiting for stale detection.

### Webhook Notifications
- [x] Generic webhook support - POST JSON on deploy start/success/fail. Configured in `.pilum.yml` with per-webhook event filtering. Payload includes service names, tag, duration, success/failure, error details, and a human-readable `text` field (works with Slack/Discord/Teams incoming webhooks out of the box).

---

## Phase 4: Expanded Providers

### Cloud Platforms
- [ ] AWS ECS (Fargate)
- [x] Azure Container Apps
- [ ] Fly.io

### Release Targets
- [ ] GitHub Releases (with assets)
- [x] Docker Hub (`dockerhub` provider)
- [x] AWS ECR (`aws` provider → `*.dkr.ecr.*.amazonaws.com`)
- [x] GCP Artifact Registry (`gcp` provider → `*-docker.pkg.dev`)
- [x] Azure Container Registry (`azure` provider → `*.azurecr.io`)
- [x] GitHub Container Registry (`github` provider → `ghcr.io`)

---

## Phase 5: Package Managers & Registries

### Language-Specific Registries
- [ ] PyPI
- [ ] crates.io
- [ ] NuGet

### System Packages
- [ ] APT/DEB packages
- [ ] RPM packages
- [ ] Scoop (Windows)
- [ ] Chocolatey (Windows)

---

## Phase 6: Pilum Cloud (Future)

> Dedicated build runners and deployment orchestration - competing with CircleCI/Jenkins for deploy-focused workflows.

### Core Platform
- [ ] `pilum login` - Authenticate to Pilum Cloud
- [ ] Hosted build runners (no local Docker required)
- [ ] Deployment queue and scheduling
- [ ] Deployment history visualization

### Team Features
- [ ] Team workspaces
- [ ] Deployment audit log (who deployed what, when)
- [ ] Role-based access control
- [ ] Deploy approvals workflow

### Integrations
- [ ] Slack notifications
- [ ] Discord notifications
- [ ] Microsoft Teams notifications
- [ ] Service dependency graph visualization

---

## Removed / Deferred

These were considered but intentionally not prioritized:

| Feature | Reason |
|---------|--------|
| Config inheritance | Internal DRY improvement, not user-facing value |
| Secrets management | Cloud providers handle this (Secret Manager, SSM) |
| Canary / Blue-green deploys | Cloud Run has traffic splitting built-in |
| Terraform/Pulumi integration | Out of scope - Pilum deploys code, not infra |
| `pilum rollback` | Dangerous as CLI command; `pilum deploy --tag=<old>` already covers the use case |
| Self-hosted Pilum Cloud | Too early to consider |
| Multi-target deployments | Two `pilum.yaml` files in separate dirs already handles this cleanly. A `targets:` array would fight the "one config = one deployment" simplicity. |
| Kubernetes (generic) | Enormous surface area, ecosystem already has Helm/Kustomize/ArgoCD/Flux. Pilum would either be too simple to be useful or replicate those tools. |
| `pilum ci detect` | YAGNI. Per-CI flags (`--github-status`) with env var auto-detection is simpler and more explicit than a generic CI detection command. |
| GitHub deployment environments | GitHub-specific, requires environment protection rule API knowledge, small audience vs commit statuses. |
| MCP server | Separate project, not a CLI feature. Better suited for Pugio (the MCP governance platform) or a standalone repo. |
