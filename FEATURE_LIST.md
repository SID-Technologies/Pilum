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

---

## Phase 1: Foundation (Pre-Launch)

### CLI Polish
- [ ] `pilum init` - Interactive scaffolding for new services
- [ ] `pilum validate` - Validate service.yaml and recipe syntax
- [ ] `pilum list` - List all discovered services and their recipes
- [ ] `--verbose / -v` - Stream command stdout/stderr in real-time
- [ ] `--quiet / -q` - Minimal output (CI-friendly)
- [ ] `--json` - JSON output for scripting/automation
- [ ] Better error messages with suggestions ("did you mean X?")

### Service Filtering
- [ ] `--filter=name:api-*` - Glob patterns for service names
- [ ] `--filter=tag:backend` - Filter by tags defined in service.yaml
- [ ] `--filter=recipe:gcp-cloudrun` - Filter by recipe type
- [ ] `--exclude=name:legacy-*` - Exclude patterns

### Configuration
- [ ] Global config file (`~/.pilum/config.yaml`)
- [ ] Project-level defaults (`.pilum.yaml`)
- [ ] Environment variable substitution in recipes (`${VAR}`)
- [ ] Config inheritance (base recipe + overrides)

---

## Phase 2: More Recipes & Languages

### Cloud Platforms
- [ ] Kubernetes (generic manifests)
- [ ] Kubernetes + Helm
- [ ] Kubernetes + Kustomize
- [ ] Azure Container Apps
- [ ] Azure Functions
- [ ] Fly.io
- [ ] Railway
- [ ] Render
- [ ] DigitalOcean App Platform
- [ ] Cloudflare Workers

### Build Systems
- [ ] Go (current)
- [ ] Node.js / npm / pnpm
- [ ] Python / pip / poetry
- [ ] Rust / Cargo
- [ ] Java / Maven / Gradle
- [ ] .NET / dotnet

### Release Targets
- [ ] GitHub Releases (with assets)
- [ ] GitLab Releases
- [ ] Docker Hub
- [ ] AWS ECR
- [ ] GCP Artifact Registry
- [ ] Azure Container Registry
- [ ] npm registry
- [ ] PyPI
- [ ] crates.io
- [ ] Homebrew (current)
- [ ] APT/DEB packages
- [ ] RPM packages
- [ ] Scoop (Windows)
- [ ] Chocolatey (Windows)

---

## Phase 3: Professional Features

### Deployment Safety
- [ ] `pilum diff` - Show what would change before deploying
- [ ] Deployment locks (prevent concurrent deploys)
- [ ] Required approvals for production
- [ ] Canary deployments (% traffic routing)
- [ ] Blue/green deployments
- [ ] Automatic rollback on health check failure
- [ ] `pilum rollback [service]` - Manual rollback to previous version

### Status & Observability
- [ ] `pilum status` - Check deployed service health
- [ ] `pilum logs [service]` - Tail logs from deployed services
- [ ] `pilum history` - View deployment history (local cache)
- [ ] Deployment duration tracking and trends

### Notifications
- [ ] Slack webhook on deploy start/complete/fail
- [ ] Discord webhook
- [ ] Microsoft Teams webhook
- [ ] Generic webhook (POST JSON)
- [ ] GitHub commit status updates
- [ ] GitHub deployment environments

### Secrets Management
- [ ] `pilum secrets list`
- [ ] `pilum secrets set KEY=value`
- [ ] HashiCorp Vault integration
- [ ] AWS Secrets Manager integration
- [ ] GCP Secret Manager integration
- [ ] Azure Key Vault integration
- [ ] 1Password CLI integration
- [ ] SOPS encrypted files

---

## Phase 4: Team & Enterprise

### Collaboration
- [ ] `pilum login` - Authenticate to Pilum Cloud
- [ ] Team workspaces
- [ ] Deployment audit log (who deployed what, when)
- [ ] Role-based access control
- [ ] Deploy approvals workflow

### Pilum Cloud (Hosted Dashboard)
- [ ] Deployment history visualization
- [ ] Service dependency graph
- [ ] One-click rollbacks
- [ ] Environment management (dev/staging/prod)
- [ ] Secrets management UI
- [ ] Team member management
- [ ] SSO (SAML, OIDC)
- [ ] Compliance reports

### Enterprise
- [ ] Self-hosted Pilum Cloud
- [ ] LDAP/Active Directory integration
- [ ] Audit log export
- [ ] Custom recipe repository (private)
- [ ] Priority support SLA

---

## Phase 5: Advanced Automation

### CI/CD Integration
- [ ] GitHub Actions (official action)
- [ ] GitLab CI templates
- [ ] CircleCI orb
- [ ] Bitbucket Pipelines
- [ ] Jenkins plugin
- [ ] `pilum ci detect` - Auto-detect CI environment

### Monorepo Support
- [ ] Detect changed services (git diff)
- [ ] `--only-changed` flag
- [ ] Dependency graph between services
- [ ] Parallel builds with dependency ordering
- [ ] Nx/Turborepo-style caching

### Infrastructure as Code
- [ ] Terraform integration (run terraform before deploy)
- [ ] Pulumi integration
- [ ] CloudFormation integration
- [ ] Generate IaC from service.yaml

### Testing Integration
- [ ] Pre-deploy test hooks
- [ ] Post-deploy smoke tests
- [ ] Integration test orchestration
- [ ] Test environment provisioning

---

## Competitive Positioning

### vs GoReleaser
- GoReleaser: Focused on Go binaries and release artifacts
- Pilum: Multi-service deployment orchestration for any language/platform

### vs Waypoint
- Waypoint: HashiCorp's opinionated build/deploy/release workflow
- Pilum: Lightweight, recipe-driven, no server required

### vs Helm/Kustomize
- Helm/Kustomize: Kubernetes-specific templating
- Pilum: Cloud-agnostic, supports K8s as one target among many

### vs Terraform/Pulumi
- Terraform/Pulumi: Infrastructure provisioning
- Pilum: Application deployment (complements IaC tools)

### vs Platform-specific CLIs (gcloud, aws, az)
- Platform CLIs: Single-cloud, imperative commands
- Pilum: Multi-cloud, declarative recipes, unified workflow
