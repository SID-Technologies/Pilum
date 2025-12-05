# Pilum Architecture

This document explains the internal architecture of Pilum for contributors and advanced users.

## Design Philosophy

Pilum follows these core principles:

1. **Recipe-driven** - Deployment workflows defined in YAML, not Go code
2. **Cloud-agnostic** - Same service config, different providers
3. **Parallel execution** - Services deploy concurrently within steps
4. **Step barriers** - All services complete step N before step N+1 begins

## High-Level Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                         pilum deploy                                 │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Service Discovery                               │
│  Find all service.yaml files in project                             │
│  lib/service_info/get_services.go                                   │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       Recipe Loading                                 │
│  Load recipes from recepies/ directory                              │
│  lib/recepie/loader.go                                              │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Recipe Matching                                   │
│  Match services to recipes by provider field                        │
│  lib/recepie/recepie.go                                             │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       Validation                                     │
│  Validate services have required recipe fields                      │
│  lib/recepie/recepie.go:ValidateService()                           │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Orchestration                                   │
│  Execute steps in order with parallel service execution             │
│  lib/orchestrator/runner.go                                         │
└─────────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Service Discovery (`lib/service_info/`)

Finds and parses `service.yaml` files throughout the project.

**Key files:**
- `get_services.go` - Walks directory tree finding services
- `service_info.go` - ServiceInfo struct and parsing logic

**ServiceInfo struct:**
```go
type ServiceInfo struct {
    Name           string         // Service name
    Description    string         // Service description
    Provider       string         // Cloud provider (gcp, aws, azure, homebrew)
    Region         string         // Deployment region
    Project        string         // Project/account identifier
    Template       string         // Dockerfile template name
    Path           string         // Path to service directory
    Config         map[string]any // Raw YAML config for custom fields
    BuildConfig    BuildConfig    // Build configuration
    HomebrewConfig HomebrewConfig // Homebrew-specific config
    // ...
}
```

### 2. Recipe System (`lib/recepie/`)

Defines deployment workflows as YAML files.

**Key files:**
- `loader.go` - Loads recipes from `recepies/` directory
- `recepie.go` - Recipe struct and validation logic

**Recipe struct:**
```go
type Recipe struct {
    Name           string          // Recipe name
    Description    string          // Human-readable description
    Provider       string          // Provider identifier for matching
    Service        string          // Service type identifier
    RequiredFields []RequiredField // Fields services must provide
    Steps          []RecipeStep    // Ordered deployment steps
}

type RecipeStep struct {
    Name          string            // Step name (for handler matching)
    Command       any               // Explicit command (string or []string)
    ExecutionMode string            // "root" or "service_dir"
    EnvVars       map[string]string // Step-specific environment
    Timeout       int               // Max execution time (seconds)
    Retries       int               // Retry count on failure
    Tags          []string          // Labels for filtering
}
```

**Recipe-driven validation:**

When `pilum check` runs, each service is validated against its recipe:

```go
func (r *Recipe) ValidateService(svc *ServiceInfo) error {
    for _, field := range r.RequiredFields {
        value := getServiceField(svc, field.Name)
        if value == "" && field.Default == "" {
            return errors.New("recipe '%s' requires field '%s': %s",
                r.Name, field.Name, field.Description)
        }
    }
    return nil
}
```

### 3. Command Registry (`lib/registry/`)

Maps step names to command generators (handlers).

**Key files:**
- `command_registry.go` - Registry implementation
- `commands.go` - Default handler registration

**How it works:**

```go
// Register a handler
registry.Register("deploy", "gcp", func(ctx StepContext) any {
    return gcp.GenerateGCPDeployCommand(ctx.Service, ctx.ImageName)
})

// Handler lookup (provider-specific takes precedence)
handler, found := registry.GetHandler("deploy to cloud run", "gcp")
```

**Handler matching rules:**

1. Step name is compared case-insensitively
2. Pattern matches if it's a substring of step name
3. Provider-specific handlers (`deploy:gcp`) take precedence over generic (`deploy`)

**StepContext:**
```go
type StepContext struct {
    Service      serviceinfo.ServiceInfo // Service configuration
    ImageName    string                   // Docker image name
    Tag          string                   // Version tag
    Registry     string                   // Container registry
    TemplatePath string                   // Path to templates
}
```

### 4. Ingredients (`ingredients/`)

Cloud-specific command generators.

**Structure:**
```
ingredients/
├── build/        # Generic build commands
│   └── build.go
├── docker/       # Docker build/push
│   ├── build.go
│   ├── push.go
│   └── helpers.go
├── gcp/          # Google Cloud Run
│   └── google_cloud_run.go
├── homebrew/     # Homebrew releases
│   ├── build.go
│   ├── archive.go
│   ├── formula.go
│   └── tap.go
└── aws/          # AWS (placeholder)
```

**Example ingredient:**
```go
// ingredients/gcp/google_cloud_run.go
func GenerateGCPDeployCommand(svc service.ServiceInfo, imageName string) []string {
    cmd := []string{
        "gcloud", "run", "deploy", svc.Name,
        "--image", imageName,
        "--region", svc.Region,
        "--platform", "managed",
        "--allow-unauthenticated",
    }
    // Add secrets if configured
    if len(svc.Secrets) > 0 {
        secretsStr := formatSecrets(svc.Secrets)
        cmd = append(cmd, "--set-secrets", secretsStr)
    }
    return cmd
}
```

### 5. Orchestrator (`lib/orchestrator/`)

Executes deployment pipelines with parallel service execution.

**Key files:**
- `runner.go` - Main execution engine
- `spinner.go` - Progress indicators
- `output.go` - Result formatting

**Execution model:**

```
Step 1: build binary          ← barrier
  ├── service-a  ───┐
  ├── service-b  ───┼── parallel execution
  └── service-c  ───┘
                     │
                     ▼ all complete
Step 2: docker build          ← barrier
  ├── service-a  ───┐
  ├── service-b  ───┼── parallel execution
  └── service-c  ───┘
                     │
                     ▼ all complete
Step 3: deploy                ← barrier
  └── ...
```

**Runner options:**
```go
type RunnerOptions struct {
    Tag          string   // Version tag
    Registry     string   // Container registry URL
    DryRun       bool     // Preview without executing
    MaxWorkers   int      // Parallel worker count
    MaxSteps     int      // Limit steps (0 = all)
    ExcludeSteps []string // Skip steps by name
    ExcludeTags  []string // Skip steps with these tags
    OnlyTags     []string // Only run steps with these tags
    Timeout      int      // Default timeout
    Retries      int      // Default retry count
}
```

### 6. Worker Queue (`lib/worker_queue/`)

Manages parallel task execution with a semaphore-based worker pool.

**Key files:**
- `worker_queue.go` - Queue implementation
- `command_worker.go` - Task execution
- `task_info.go` - Task configuration

**Worker count:**
- Default: min(service count, 4)
- Override with `--max-workers`
- Configurable per-step via semaphore

### 7. Output (`lib/output/`)

Consistent CLI output formatting.

**Usage:**
```go
output.Info("Starting deployment...")
output.Success("Deployed %s", serviceName)
output.Error("Failed: %v", err)
output.Debug("Verbose: %s", details)  // Only with --debug
```

**Features:**
- Colored output (semantic theming)
- Spinners for in-progress tasks
- Aligned service names
- Duration tracking

### 8. Errors (`lib/errors/`)

Contextual error handling.

**Usage:**
```go
// Wrap errors with context
if err != nil {
    return errors.Wrap(err, "failed to load service %s", name)
}

// Create new errors
return errors.New("invalid provider: %s", provider)
```

## Command Flow

### `pilum deploy`

```
cmd/deploy.go
    │
    ▼
lib/service_info/get_services.go
    │ Find all service.yaml files
    ▼
lib/recepie/loader.go
    │ Load all recipes
    ▼
lib/orchestrator/runner.go:NewRunner()
    │ Initialize registry and index recipes
    ▼
lib/orchestrator/runner.go:Run()
    │
    ├── For each step index:
    │   │
    │   ▼
    │   lib/orchestrator/runner.go:executeStep()
    │       │
    │       ├── Collect tasks (service + step pairs)
    │       ├── Filter by tags (--only-tags, --exclude-tags)
    │       │
    │       ▼
    │       lib/orchestrator/runner.go:executeTasksParallel()
    │           │
    │           ├── Start spinner manager
    │           ├── Launch workers (semaphore-limited)
    │           │
    │           ▼
    │           lib/orchestrator/runner.go:executeTask()
    │               │
    │               ▼
    │               lib/orchestrator/runner.go:generateCommand()
    │                   │
    │                   ├── If step.Command set → substituteVars()
    │                   └── Else → registry.GetHandler()
    │                              └── handler(StepContext) → command
    │               │
    │               ▼
    │               lib/worker_queue/command_worker.go
    │                   │ Execute command with timeout/retries
    │                   ▼
    │               Return TaskResult
    │
    ▼
Output results
```

### `pilum check`

```
cmd/check.go
    │
    ▼
lib/service_info/get_services.go
    │ Find all service.yaml files
    ▼
lib/recepie/loader.go
    │ Load all recipes
    ▼
For each service:
    │
    ├── Find recipe by provider
    │
    ▼
lib/recepie/recepie.go:ValidateService()
    │ Check all required fields
    ▼
Report missing fields
```

## Variable Substitution

Recipe commands support variable substitution:

| Variable | Source |
|----------|--------|
| `${name}` | service.yaml `name` field |
| `${provider}` | service.yaml `provider` field |
| `${region}` | service.yaml `region` field |
| `${project}` | service.yaml `project` field |
| `${tag}` | `--tag` CLI flag |

**Implementation:**
```go
func (r *Runner) substituteVars(cmd any, svc ServiceInfo) any {
    replacer := strings.NewReplacer(
        "${name}", svc.Name,
        "${provider}", svc.Provider,
        "${region}", svc.Region,
        "${project}", svc.Project,
        "${tag}", r.options.Tag,
    )
    // Apply to string or []string commands
}
```

## Step Filtering

Steps can be filtered using tags and names:

**Recipe definition:**
```yaml
steps:
  - name: deploy to cloud run
    tags:
      - deploy
```

**CLI usage:**
```bash
# Only run deploy-tagged steps
pilum deploy --only-tags=deploy

# Skip deploy-tagged steps
pilum deploy --exclude-tags=deploy

# Combine with build command (auto-excludes deploy)
pilum build --tag=v1.0.0
```

**Implementation (`lib/orchestrator/runner.go`):**
```go
func (r *Runner) shouldSkipStep(step *RecipeStep) bool {
    // If OnlyTags is set, step must have matching tag
    if len(r.options.OnlyTags) > 0 {
        if !r.stepHasAnyTag(step, r.options.OnlyTags) {
            return true
        }
    }
    // If ExcludeTags is set, skip steps with excluded tags
    if len(r.options.ExcludeTags) > 0 {
        if r.stepHasAnyTag(step, r.options.ExcludeTags) {
            return true
        }
    }
    return false
}
```

## Extension Points

### Adding a Provider

1. **Recipe only** (explicit commands):
   - Add `recepies/<provider>-recepie.yaml`
   - Define steps with `command:` field

2. **Recipe + handlers** (dynamic commands):
   - Add recipe YAML
   - Create `ingredients/<provider>/*.go`
   - Register handlers in `lib/registry/commands.go`

See [docs/adding-a-provider.md](adding-a-provider.md) for detailed examples.

### Custom Step Types

Register new step patterns:

```go
// lib/registry/commands.go
func registerMyHandlers(reg *CommandRegistry) {
    reg.Register("test", "", func(ctx StepContext) any {
        return []string{"go", "test", "./..."}
    })
}
```

### Custom Output

Use the output package for consistent formatting:

```go
import "github.com/sid-technologies/pilum/lib/output"

output.Header("My Section")
output.Info("Processing %s", name)
output.Success("Done!")
```

## File Locations

| Component | Location | Purpose |
|-----------|----------|---------|
| CLI commands | `cmd/*.go` | Cobra command definitions |
| Service parsing | `lib/service_info/` | service.yaml parsing |
| Recipe system | `lib/recepie/` | Recipe loading and validation |
| Handler registry | `lib/registry/` | Step → command mapping |
| Execution engine | `lib/orchestrator/` | Parallel step execution |
| Worker pool | `lib/worker_queue/` | Task parallelization |
| Ingredients | `ingredients/*/` | Cloud-specific commands |
| Recipes | `recepies/*.yaml` | Deployment workflows |
| Output | `lib/output/` | CLI formatting |
| Errors | `lib/errors/` | Error handling |
