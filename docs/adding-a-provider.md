# Adding a New Provider to Pilum

This guide walks through adding cloud provider support to Pilum. We'll cover three examples:
1. **AWS Lambda** (SAM-based) - Using explicit commands only
2. **AWS ECR + Fargate** - Using handlers with custom ingredients
3. **Azure Container Apps** - Hybrid approach

## Understanding the Architecture

Pilum uses a **recipe-driven** approach:

```
service.yaml → Recipe (YAML) → Steps → Handlers/Commands → Execution
```

**Three ways to define step commands:**

| Approach | When to Use | Example |
|----------|-------------|---------|
| **Explicit commands** | Simple, static commands | `command: ["aws", "s3", "sync"]` |
| **Generic handlers** | Reusable across providers | `build`, `docker`, `push` |
| **Provider handlers** | Provider-specific logic | `deploy` for `gcp` |

## Example 1: AWS Lambda (Explicit Commands)

The simplest approach - no Go code required. Just define explicit commands in the recipe.

### Step 1: Create the Recipe

**File: `recepies/aws-lambda-recepie.yaml`**

```yaml
name: aws-lambda
description: Deploy to AWS Lambda using SAM
provider: aws
service: lambda

required_fields:
  - name: region
    description: AWS region (e.g., us-east-1)
    type: string

  - name: project
    description: Project name for S3 bucket naming
    type: string

  - name: stack_name
    description: CloudFormation stack name
    type: string
    default: ${name}-stack  # Uses service name if not specified

steps:
  - name: build
    command: ["sam", "build"]
    execution_mode: service_dir
    timeout: 120

  - name: package
    command: ["sam", "package",
              "--s3-bucket", "${project}-deployments",
              "--region", "${region}"]
    execution_mode: service_dir
    timeout: 180

  - name: deploy
    command: ["sam", "deploy",
              "--stack-name", "${name}",
              "--region", "${region}",
              "--no-confirm-changeset",
              "--capabilities", "CAPABILITY_IAM"]
    execution_mode: service_dir
    timeout: 240
    retries: 1
    tags:
      - deploy
```

### Step 2: Create a Service Configuration

**File: `services/my-lambda/service.yaml`**

```yaml
name: my-lambda-function
provider: aws
region: us-east-1
project: my-company
stack_name: my-lambda-stack

build:
  language: go
  version: "1.23"
```

### Step 3: Test

```bash
# Validate configuration
pilum check

# Preview commands
pilum dry-run

# Deploy
pilum deploy --tag=v1.0.0
```

**That's it!** No Go code needed for explicit command recipes.

---

## Example 2: AWS ECR + Fargate (With Handlers)

For more complex providers, create handlers that generate commands dynamically.

### Step 1: Create the Recipe

**File: `recepies/aws-fargate-recepie.yaml`**

```yaml
name: aws-fargate
description: Deploy to AWS ECS Fargate via ECR
provider: aws-fargate
service: container

required_fields:
  - name: region
    description: AWS region
    type: string

  - name: cluster
    description: ECS cluster name
    type: string

  - name: ecr_repository
    description: ECR repository name
    type: string

  - name: task_definition
    description: ECS task definition name
    type: string

  - name: service_name
    description: ECS service name
    type: string
    default: ${name}

steps:
  # Step 1: Build the Go binary
  - name: build binary
    execution_mode: service_dir
    timeout: 300

  # Step 2: Build Docker image
  - name: build docker image
    execution_mode: service_dir
    timeout: 300

  # Step 3: Authenticate with ECR
  - name: ecr login
    execution_mode: root
    timeout: 60

  # Step 4: Push to ECR
  - name: push to ecr
    execution_mode: root
    timeout: 180

  # Step 5: Update ECS service
  - name: deploy to fargate
    execution_mode: root
    timeout: 300
    retries: 2
    tags:
      - deploy
```

### Step 2: Create the Ingredient Package

**File: `ingredients/aws/ecr.go`**

```go
package aws

import (
    "fmt"
    service "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateECRLoginCommand generates the ECR authentication command
func GenerateECRLoginCommand(svc service.ServiceInfo) []string {
    return []string{
        "aws", "ecr", "get-login-password",
        "--region", svc.Region,
        "|",
        "docker", "login",
        "--username", "AWS",
        "--password-stdin",
        fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", svc.Project, svc.Region),
    }
}

// GenerateECRPushCommand generates the ECR push command
func GenerateECRPushCommand(svc service.ServiceInfo, imageName string) []string {
    ecrRepo := getConfigString(svc, "ecr_repository", svc.Name)
    return []string{
        "docker", "push",
        fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s",
            svc.Project, svc.Region, ecrRepo, imageName),
    }
}

// getConfigString safely gets a string from service config
func getConfigString(svc service.ServiceInfo, key, defaultVal string) string {
    if val, ok := svc.Config[key].(string); ok && val != "" {
        return val
    }
    return defaultVal
}
```

**File: `ingredients/aws/fargate.go`**

```go
package aws

import (
    "fmt"
    service "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateFargateDeployCommand generates the ECS service update command
func GenerateFargateDeployCommand(svc service.ServiceInfo, imageName string) []string {
    cluster := getConfigString(svc, "cluster", "default")
    serviceName := getConfigString(svc, "service_name", svc.Name)
    taskDef := getConfigString(svc, "task_definition", svc.Name)

    return []string{
        "aws", "ecs", "update-service",
        "--cluster", cluster,
        "--service", serviceName,
        "--task-definition", taskDef,
        "--force-new-deployment",
        "--region", svc.Region,
    }
}

// GenerateECRImageName generates the full ECR image URI
func GenerateECRImageName(svc service.ServiceInfo, tag string) string {
    ecrRepo := getConfigString(svc, "ecr_repository", svc.Name)
    return fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s",
        svc.Project, svc.Region, ecrRepo, tag)
}
```

### Step 3: Register Handlers

**File: `lib/registry/commands.go`**

Add to the existing file:

```go
import (
    // ... existing imports
    "github.com/sid-technologies/pilum/ingredients/aws"
)

func RegisterDefaultHandlers(reg *CommandRegistry) {
    registerDockerHandlers(reg)
    registerBuildHandlers(reg)
    registerDeployHandlers(reg)
    registerHomebrewHandlers(reg)
    registerAWSHandlers(reg)  // Add this line
}

func registerAWSHandlers(reg *CommandRegistry) {
    // ECR login
    reg.Register("ecr", "aws-fargate", func(ctx StepContext) any {
        return aws.GenerateECRLoginCommand(ctx.Service)
    })

    // Push to ECR
    reg.Register("push", "aws-fargate", func(ctx StepContext) any {
        return aws.GenerateECRPushCommand(ctx.Service, ctx.ImageName)
    })

    // Deploy to Fargate
    reg.Register("deploy", "aws-fargate", func(ctx StepContext) any {
        return aws.GenerateFargateDeployCommand(ctx.Service, ctx.ImageName)
    })
}
```

### Step 4: Service Configuration

**File: `services/my-api/service.yaml`**

```yaml
name: my-api
provider: aws-fargate
region: us-east-1
project: "123456789012"  # AWS Account ID

cluster: production-cluster
ecr_repository: my-api
task_definition: my-api-task
service_name: my-api-service

build:
  language: go
  version: "1.23"
  env_vars:
    CGO_ENABLED: "0"
    GOOS: linux
    GOARCH: amd64
```

---

## Example 3: Azure Container Apps (Hybrid)

Mix explicit commands with handlers for flexibility.

### Step 1: Create the Recipe

**File: `recepies/azure-container-apps-recepie.yaml`**

```yaml
name: azure-container-apps
description: Deploy to Azure Container Apps
provider: azure
service: container

required_fields:
  - name: resource_group
    description: Azure resource group
    type: string

  - name: acr_name
    description: Azure Container Registry name
    type: string

  - name: container_app_env
    description: Container Apps environment name
    type: string

  - name: location
    description: Azure region (e.g., eastus)
    type: string
    default: eastus

steps:
  - name: build binary
    execution_mode: service_dir
    timeout: 300

  - name: build docker image
    execution_mode: service_dir
    timeout: 300

  # Explicit command for ACR login
  - name: acr login
    command: ["az", "acr", "login", "--name", "${acr_name}"]
    execution_mode: root
    timeout: 60

  - name: push to acr
    execution_mode: root
    timeout: 180

  - name: deploy to container apps
    execution_mode: root
    timeout: 300
    retries: 2
    tags:
      - deploy
```

### Step 2: Create Ingredient

**File: `ingredients/azure/container_apps.go`**

```go
package azure

import (
    "fmt"
    service "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateACRPushCommand generates Azure Container Registry push command
func GenerateACRPushCommand(svc service.ServiceInfo, imageName string) []string {
    acrName := getConfigString(svc, "acr_name", "")
    return []string{
        "docker", "push",
        fmt.Sprintf("%s.azurecr.io/%s", acrName, imageName),
    }
}

// GenerateContainerAppsDeployCommand generates the az containerapp update command
func GenerateContainerAppsDeployCommand(svc service.ServiceInfo, imageName string) []string {
    resourceGroup := getConfigString(svc, "resource_group", "")
    acrName := getConfigString(svc, "acr_name", "")

    return []string{
        "az", "containerapp", "update",
        "--name", svc.Name,
        "--resource-group", resourceGroup,
        "--image", fmt.Sprintf("%s.azurecr.io/%s", acrName, imageName),
    }
}

func getConfigString(svc service.ServiceInfo, key, defaultVal string) string {
    if val, ok := svc.Config[key].(string); ok && val != "" {
        return val
    }
    return defaultVal
}
```

### Step 3: Register Handlers

```go
func registerAzureHandlers(reg *CommandRegistry) {
    reg.Register("push", "azure", func(ctx StepContext) any {
        return azure.GenerateACRPushCommand(ctx.Service, ctx.ImageName)
    })

    reg.Register("deploy", "azure", func(ctx StepContext) any {
        return azure.GenerateContainerAppsDeployCommand(ctx.Service, ctx.ImageName)
    })
}
```

---

## Handler Matching Rules

Handlers are matched by **pattern** (substring) and optionally **provider**:

```go
// Matches step names containing "deploy" for "gcp" provider only
registry.Register("deploy", "gcp", handler)

// Matches step names containing "build" for any provider
registry.Register("build", "", handler)

// Matches "docker" in step name (e.g., "build docker image")
registry.Register("docker", "", handler)
```

**Priority:** Provider-specific handlers take precedence over generic ones.

**Step name matching examples:**

| Step Name | Pattern | Provider | Matches? |
|-----------|---------|----------|----------|
| `build binary` | `build` | `""` | Yes |
| `build docker image` | `docker` | `""` | Yes |
| `deploy to cloud run` | `deploy` | `gcp` | Yes (if service is gcp) |
| `deploy to cloud run` | `deploy` | `aws` | No (wrong provider) |

---

## Available Variables in Explicit Commands

These variables are available in explicit command templates:

| Variable | Source | Example |
|----------|--------|---------|
| `${name}` | `service.yaml` name field | `my-api` |
| `${tag}` | `--tag` CLI flag | `v1.0.0` |
| `${provider}` | Service provider | `aws` |
| `${region}` | Service region | `us-east-1` |
| `${project}` | Service project | `my-project` |

Example usage:

```yaml
command: ["aws", "s3", "sync", "./dist", "s3://${project}-${name}-${tag}"]
```

---

## Step Configuration Options

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Step name (used for handler matching) |
| `command` | string[] | Explicit command (overrides handler) |
| `execution_mode` | string | `root` or `service_dir` |
| `timeout` | int | Max seconds to wait |
| `retries` | int | Retry count on failure |
| `env_vars` | map | Environment variables |
| `tags` | string[] | Labels for filtering (`--only-tags`, `--exclude-tags`) |

---

## Testing Your Provider

```bash
# 1. Validate service configurations
pilum check

# 2. Preview generated commands
pilum dry-run

# 3. Run build steps only
pilum build --tag=test

# 4. Run specific step by tag
pilum deploy --only-tags=deploy --tag=v1.0.0

# 5. Full deployment
pilum deploy --tag=v1.0.0
```

---

## Checklist for New Providers

- [ ] Recipe YAML in `recepies/<provider>-recepie.yaml`
- [ ] Required fields defined with descriptions
- [ ] Steps ordered correctly (build → push → deploy)
- [ ] Timeouts set appropriately
- [ ] Deploy steps tagged with `deploy`
- [ ] Handlers registered (if not using explicit commands)
- [ ] Ingredient package created (if using handlers)
- [ ] Example service.yaml documented
- [ ] Tested with `pilum dry-run`

---

## Common Patterns

### Multi-Platform Builds

```yaml
steps:
  - name: build linux amd64
    command: ["go", "build", "-o", "dist/app-linux-amd64"]
    env_vars:
      GOOS: linux
      GOARCH: amd64
    execution_mode: service_dir

  - name: build linux arm64
    command: ["go", "build", "-o", "dist/app-linux-arm64"]
    env_vars:
      GOOS: linux
      GOARCH: arm64
    execution_mode: service_dir
```

### Conditional Authentication

```yaml
steps:
  - name: authenticate
    command: ["gcloud", "auth", "configure-docker", "${region}-docker.pkg.dev"]
    execution_mode: root
    timeout: 30
```

### Health Check After Deploy

```yaml
steps:
  - name: deploy
    command: ["kubectl", "apply", "-f", "k8s/"]
    execution_mode: service_dir

  - name: verify
    command: ["kubectl", "rollout", "status", "deployment/${name}"]
    execution_mode: service_dir
    timeout: 120
```
