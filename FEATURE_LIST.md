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

---

## Phase 2: Visibility & Safety

### Status & Observability
- [ ] `pilum status` - Show deployed service versions, health, and last deploy time
- [ ] `pilum logs [service]` - Tail logs from deployed services (wraps `gcloud logs`)
- [ ] `pilum history` - View deployment history (local cache)

### Monorepo Support
- [x] `--only-changed` flag - Detect git changes, deploy only affected services
- [x] `--since` flag - Specify git ref to compare against (default: main/master)
- [x] Dependency graph between services (`depends_on` in pilum.yaml)

### Deployment Safety
- [ ] `pilum rollback [service]` - Rollback to previous revision
- [ ] Deployment locks (prevent concurrent deploys to same service)

### Multi-Target Deployments
- [ ] Deploy same service to multiple targets (e.g., Cloud Run + GKE) from single config
- [ ] Options: multiple `pilum.yaml` files or `targets:` array in config

### Environment Management
- [ ] Environment configs (`--env prod` / `--env staging`)
- [ ] Per-environment overrides in pilum.yaml

---

## Phase 3: CI/CD & Automation

### CI/CD Integration
- [ ] GitHub Actions (official action: `uses: pilum/deploy@v1`)
- [ ] `pilum ci detect` - Auto-detect CI environment and set defaults
- [ ] GitHub commit status updates
- [ ] GitHub deployment environments

### Advanced Monorepo
- [ ] Parallel builds with dependency ordering
- [ ] Build caching (hash-based skip)
- [ ] Pattern matching for service selection (`pilum deploy "api-*"`)
- [ ] Filter services by provider (`--provider=gcp`)
- [ ] Environment variable substitution in pilum.yaml (`${GCP_PROJECT}`)

---

## Phase 4: Expanded Providers

### Cloud Platforms (Priority Order)
- [ ] AWS ECS (Fargate)
- [ ] Kubernetes (generic manifests)
- [ ] Azure Container Apps
- [ ] Fly.io
- [ ] GCP Cloud Run Jobs (batch/migration workloads)

### Release Targets
- [ ] GitHub Releases (with assets)
- [ ] Docker Hub
- [ ] AWS ECR
- [ ] GCP Artifact Registry (current: gcr.io)
- [ ] Azure Container Registry

### Notifications
- [ ] Generic webhook (POST JSON on deploy start/complete/fail)

---

## Phase 5: Package Managers & Registries

### Language-Specific Registries
- [ ] npm registry
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
| Self-hosted Pilum Cloud | Too early to consider |
