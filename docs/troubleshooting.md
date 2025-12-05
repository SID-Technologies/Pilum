# Troubleshooting Guide

Common issues and solutions when using Pilum.

## Service Discovery Issues

### "No services found"

**Symptom:**
```
$ pilum deploy
No services to deploy
```

**Causes:**
1. No `service.yaml` files in the project
2. Running from wrong directory
3. Service files named incorrectly

**Solutions:**

Check you're in the project root:
```bash
# Should show your project with recepies/ and services
ls -la

# Find service files
find . -name "service.yaml"
```

Ensure files are named `service.yaml` (not `service.yml` or `services.yaml`).

### "No recipe found for provider"

**Symptom:**
```
Error: no recipe found for provider 'my-provider'
```

**Causes:**
1. Recipe file missing
2. Provider name mismatch
3. Recipe YAML syntax error

**Solutions:**

Check recipe exists:
```bash
ls recepies/
# Should show: my-provider-recepie.yaml (note spelling)
```

Verify provider matches:
```yaml
# service.yaml
provider: gcp  # Must match recipe's provider field

# gcp-cloud-run-recepie.yaml
provider: gcp  # This value
```

Validate recipe syntax:
```bash
# Check for YAML errors
cat recepies/my-provider-recepie.yaml | python3 -c "import yaml,sys; yaml.safe_load(sys.stdin)"
```

## Validation Errors

### "recipe requires field 'X'"

**Symptom:**
```
$ pilum check
Error: recipe 'gcp-cloud-run' requires field 'region': GCP region to deploy to
```

**Solution:**

Add the missing field to your `service.yaml`:
```yaml
name: my-service
provider: gcp
region: us-central1  # Add missing field
project: my-project
```

Check recipe requirements:
```bash
cat recepies/gcp-cloud-run-recepie.yaml
# Look at required_fields section
```

### Nested field errors

**Symptom:**
```
Error: recipe 'homebrew' requires field 'homebrew.tap_url'
```

**Solution:**

Use nested YAML structure:
```yaml
name: my-tool
provider: homebrew

homebrew:
  tap_url: https://github.com/org/homebrew-tap
  project_url: https://github.com/org/project
  token_env: GH_TOKEN
```

## Build Failures

### "command not found"

**Symptom:**
```
Step 1: build binary
  my-service  ✗ (0.1s) - exec: "go": executable file not found in $PATH
```

**Solution:**

Ensure required tools are installed and in PATH:
```bash
# For Go services
which go
go version

# For SAM Lambda
which sam
sam --version

# For Docker
which docker
docker --version
```

### Build timeout

**Symptom:**
```
  my-service  ✗ (300.0s) - context deadline exceeded
```

**Solutions:**

1. Increase timeout in recipe:
```yaml
steps:
  - name: build binary
    timeout: 600  # 10 minutes
```

2. Or use CLI flag:
```bash
pilum deploy --timeout=600
```

3. Check if build is stuck (run manually):
```bash
cd services/my-service
go build -o dist/app .
```

### Docker build fails

**Symptom:**
```
Step 2: build docker image
  my-service  ✗ - Dockerfile not found
```

**Solutions:**

Check Dockerfile exists:
```bash
ls services/my-service/Dockerfile
```

Or specify template in service.yaml:
```yaml
name: my-service
template: go-alpine  # Uses templates/go-alpine/Dockerfile
```

## Deployment Failures

### Authentication errors

**GCP:**
```
ERROR: (gcloud.run.deploy) PERMISSION_DENIED
```

**Solution:**
```bash
gcloud auth login
gcloud auth configure-docker
gcloud config set project YOUR_PROJECT_ID
```

**AWS:**
```
Unable to locate credentials
```

**Solution:**
```bash
aws configure
# Or use environment variables
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
export AWS_REGION=us-east-1
```

**Azure:**
```
az login required
```

**Solution:**
```bash
az login
az acr login --name YOUR_ACR_NAME
```

### Push to registry fails

**Symptom:**
```
Step 3: push to registry
  my-service  ✗ - denied: access forbidden
```

**Solutions:**

1. Authenticate with registry:
```bash
# GCP
gcloud auth configure-docker us-central1-docker.pkg.dev

# AWS ECR
aws ecr get-login-password | docker login --username AWS --password-stdin 123456789.dkr.ecr.us-east-1.amazonaws.com

# Docker Hub
docker login
```

2. Check registry URL in service.yaml:
```yaml
registry_name: us-central1-docker.pkg.dev/my-project/my-repo
```

### "service does not exist"

**Symptom:**
```
ERROR: (gcloud.run.deploy) The service 'my-service' does not exist
```

**Solution:**

First deployment requires `--allow-unauthenticated` or proper IAM. The GCP handler includes this, but check your project permissions:

```bash
gcloud projects get-iam-policy YOUR_PROJECT_ID
```

## Parallel Execution Issues

### Race conditions

**Symptom:**
Services interfering with each other during parallel execution.

**Solutions:**

1. Reduce worker count:
```bash
pilum deploy --max-workers=1
```

2. Use separate working directories:
```yaml
# Recipe
steps:
  - name: build
    execution_mode: service_dir  # Each service runs in its own directory
```

### Out of memory

**Symptom:**
System slows or crashes with many parallel builds.

**Solution:**
```bash
# Limit concurrent workers
pilum deploy --max-workers=2
```

## Dry Run Issues

### Commands look wrong

**Symptom:**
```bash
$ pilum dry-run
my-service: build binary
  [go build -o dist/app .]  # Missing flags
```

**Solutions:**

1. Check build configuration in service.yaml:
```yaml
build:
  language: go
  version: "1.23"
  env_vars:
    CGO_ENABLED: "0"
  flags:
    ldflags: ["-s", "-w"]
```

2. Check handler in registry (`lib/registry/commands.go`)

### Variables not substituted

**Symptom:**
```
my-service: deploy
  [aws s3 sync ./dist s3://${project}-bucket]  # ${project} not replaced
```

**Solution:**

Ensure field exists in service.yaml:
```yaml
name: my-service
project: my-project  # This enables ${project} substitution
```

## Tag Filtering Issues

### Steps not running

**Symptom:**
```bash
$ pilum deploy --only-tags=deploy
Step 1: build binary  # Skipped
Step 2: docker build  # Skipped
Step 3: deploy        # Only this runs
```

**Explanation:**

This is expected! `--only-tags=deploy` skips steps without the `deploy` tag.

**Solution:**

If you want build + deploy:
```bash
# Run all steps (default)
pilum deploy

# Or run build separately, then deploy
pilum build --tag=v1.0.0
pilum deploy --only-tags=deploy --tag=v1.0.0
```

### Wrong tags

Check recipe has correct tags:
```yaml
steps:
  - name: deploy to cloud run
    tags:
      - deploy  # Case-sensitive
```

## Homebrew-Specific Issues

### Formula generation fails

**Symptom:**
```
Step 4: update formula
  pilum  ✗ - formula template error
```

**Solutions:**

1. Check required homebrew fields:
```yaml
homebrew:
  project_url: https://github.com/org/project
  tap_url: https://github.com/org/homebrew-tap
  token_env: GH_TOKEN
```

2. Ensure checksums exist:
```bash
ls dist/*.sha256
```

### Tap push fails

**Symptom:**
```
Step 5: push to tap
  pilum  ✗ - authentication required
```

**Solution:**

Set the token environment variable:
```bash
export GH_TOKEN="your-github-token"
# Or whatever token_env is set to in service.yaml
```

Token needs write access to the tap repository.

## Debug Mode

For detailed output, use debug mode:

```bash
pilum deploy --debug
```

This shows:
- Command arguments
- Working directories
- Environment variables
- Full error traces

## Getting Help

If issues persist:

1. Run with `--debug` for verbose output
2. Check [GitHub Issues](https://github.com/sid-technologies/pilum/issues)
3. Open a new issue with:
   - Pilum version (`pilum --version`)
   - Full error message
   - service.yaml (sanitized)
   - Recipe being used
   - Debug output
