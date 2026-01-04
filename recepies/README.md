# Adding a New Recipe

This guide explains how to add a new deployment recipe to Pilum.

## Overview

Recipes define deployment workflows as ordered steps. Each step can either:
1. Use a **registered handler** (auto-generated command based on step name)
2. Use an **explicit command** (shell command defined in the recipe)

Recipes also define **required fields** that services must provide - this is how validation works without writing Go code for each provider.

## Step 1: Create the Recipe YAML

Create a new file in `recepies/` following this structure:

```yaml
name: my-provider
description: Deploy to My Provider
provider: my-provider    # Used for handler lookup and validation
service: my-service      # Required - service type identifier

# Required fields - validated when running `pilum check`
required_fields:
  - name: project
    description: Project identifier
    type: string

  - name: region
    description: Deployment region
    type: string
    default: us-east-1    # Optional default value

# Optional fields - shown during `pilum init`, have defaults
optional_fields:
  - name: min_instances
    description: Minimum number of instances
    type: int
    default: "0"

  - name: max_instances
    description: Maximum number of instances
    type: int
    default: "10"

steps:
  - name: build
    execution_mode: service_dir
    timeout: 300

  - name: deploy
    execution_mode: root
    timeout: 180
    retries: 2
```

## Required Fields

The `required_fields` section defines what a `pilum.yaml` must contain to use this recipe. This is **recipe-driven validation** - no Go code needed per provider.

```yaml
required_fields:
  - name: project           # Field name in pilum.yaml
    description: GCP project ID   # Shown in error messages
    type: string            # string, int, bool, list
    default: ""             # If set, field is optional
```

When you run `pilum check`, it:
1. Finds all `pilum.yaml` files
2. Looks up the recipe for each service's provider
3. Validates that all required fields (without defaults) are present

## Optional Fields

The `optional_fields` section defines additional configuration that users can provide. These are shown during `pilum init` and have sensible defaults.

```yaml
optional_fields:
  - name: min_instances     # Field name in pilum.yaml
    description: Minimum number of instances
    type: int
    default: "0"            # Used if not specified

  - name: memory
    description: Memory allocation (e.g., 512Mi, 2Gi)
    type: string
    default: "512Mi"
```

Optional fields are **not validated** by `pilum check` - they simply provide suggestions during service initialization.

### Field Types

| Type | Description |
|------|-------------|
| `string` | Text value |
| `int` | Integer value |
| `bool` | true/false |
| `list` | Array of values |

### Validation Example

Recipe:
```yaml
required_fields:
  - name: project
    description: GitHub org for release URLs
    type: string
```

Service (valid):
```yaml
name: my-app
provider: homebrew
project: my-org
```

Service (invalid - missing project):
```yaml
name: my-app
provider: homebrew
# Error: recipe 'homebrew' requires field 'project': GitHub org for release URLs
```

## Step Options

| Field | Description |
|-------|-------------|
| `name` | Step name (used for handler lookup) |
| `command` | Explicit shell command (optional - overrides handler) |
| `execution_mode` | `root` (project root) or `service_dir` (service directory) |
| `timeout` | Max execution time in seconds |
| `retries` | Number of retry attempts on failure |
| `env_vars` | Environment variables for this step |
| `tags` | Labels for filtering steps |

## Using Explicit Commands

For custom logic, define the command directly:

```yaml
steps:
  - name: custom step
    command: |
      echo "Building ${name} version ${tag}"
      ./custom-build.sh
    execution_mode: root
    timeout: 60
```

Available variables: `${name}`, `${tag}`, `${provider}`, `${region}`, `${project}`

## Step 2: Register Handlers (Optional)

If your recipe uses step names that need auto-generated commands, register handlers in `lib/registry/commands.go`:

```go
func registerMyProviderHandlers(reg *CommandRegistry) {
    reg.Register("deploy", "my-provider", func(ctx StepContext) any {
        return myprovider.GenerateDeployCommand(ctx.Service, ctx.ImageName)
    })
}
```

Then add the call in `RegisterDefaultHandlers()`:
```go
func RegisterDefaultHandlers(reg *CommandRegistry) {
    registerDockerHandlers(reg)
    registerBuildHandlers(reg)
    registerDeployHandlers(reg)
    registerHomebrewHandlers(reg)
    registerMyProviderHandlers(reg)  // Add your handlers
}
```

### Handler Matching

Handlers are matched by:
1. **Pattern** - Substring match against step name (case-insensitive)
2. **Provider** - Optional, for provider-specific handlers

```go
// Matches any step containing "deploy" for "gcp" provider
registry.Register("deploy", "gcp", handler)

// Matches any step containing "build" for any provider
registry.Register("build", "", handler)
```

Provider-specific handlers take precedence over generic ones.

## Step 3: Create the Ingredient (Optional)

If you need custom command generation, create a new ingredient in `ingredients/`:

```
ingredients/
└── myprovider/
    └── deploy.go
```

```go
package myprovider

import serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

func GenerateDeployCommand(svc serviceinfo.ServiceInfo, imageName string) []string {
    return []string{
        "my-cli",
        "deploy",
        "--name", svc.Name,
        "--image", imageName,
        "--region", svc.Region,
    }
}
```

## Step 4: Test Your Recipe

```bash
# Validate services against recipe requirements
pilum check

# Preview what would execute
pilum dry-run --provider my-provider

# Run the deployment
pilum deploy
```

## Examples

### Simple Recipe (Explicit Commands Only)

No handler registration needed - just shell commands and validation:

```yaml
name: static-site
description: Deploy static site to S3
provider: aws-s3
service: static    # Required - service type identifier

required_fields:
  - name: bucket
    description: S3 bucket name
    type: string

optional_fields:
  - name: cache_control
    description: Cache-Control header value
    type: string
    default: "max-age=3600"

steps:
  - name: build
    command: npm run build
    execution_mode: service_dir
    timeout: 120

  - name: sync to s3
    command: aws s3 sync ./dist s3://${bucket} --delete
    execution_mode: service_dir
    timeout: 60
```

### Full Recipe (With Handlers)

Using registered handlers for standard steps:

```yaml
name: kubernetes
description: Deploy to Kubernetes
provider: k8s
service: container    # Required - service type identifier

required_fields:
  - name: cluster
    description: Kubernetes cluster name
    type: string
  - name: namespace
    description: Kubernetes namespace
    type: string
    default: default

steps:
  - name: build binary
    execution_mode: service_dir
    timeout: 300

  - name: docker build
    execution_mode: service_dir
    timeout: 300

  - name: push to registry
    execution_mode: root
    timeout: 120

  - name: deploy to k8s
    execution_mode: root
    timeout: 180
```

## Handler vs Explicit Command

| Use Handler When | Use Explicit Command When |
|------------------|---------------------------|
| Reusable across recipes | One-off custom logic |
| Complex command generation | Simple shell commands |
| Needs service context | Static commands |
| Provider-specific logic | Provider-agnostic steps |

## Scaling to 50+ Providers

The recipe-driven approach scales well:

| Approach | 50 providers means... |
|----------|----------------------|
| Go validators per provider | 50 Go files to maintain |
| Recipe-driven validation | 50 YAML files (you need these anyway) |

Benefits:
- **Self-documenting** - Recipe declares its own requirements (required + optional fields)
- **No code changes** - Adding a provider = adding a YAML file
- **User-editable** - Users can customize without recompiling
- **Discoverable** - `pilum check` shows exactly what's missing
- **Interactive setup** - `pilum init` walks users through all fields
