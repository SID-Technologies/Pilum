#!/bin/bash
# E2E Test Runner for Pilum
# This script runs visual tests to verify the CLI output and spinner behavior

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PILUM="$PROJECT_ROOT/dist/pilum"
FIXTURES="$SCRIPT_DIR/fixtures"
RECIPES="$SCRIPT_DIR/recipes"

# Colors for test output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

print_header() {
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
    echo -e "${BOLD}$1${RESET}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
    echo ""
}

print_test() {
    echo -e "${YELLOW}▶ $1${RESET}"
}

print_success() {
    echo -e "${GREEN}✓ $1${RESET}"
}

print_separator() {
    echo ""
    echo -e "${CYAN}──────────────────────────────────────────────────${RESET}"
    echo ""
}

print_fail() {
    echo -e "${RED}✗ $1${RESET}"
}

# assert_contains fails the test if $1 (haystack) does NOT contain $2 (needle).
# Used by Test 8's dry-run output assertions to verify specific flags / image
# refs appear in emitted gcloud commands. $3 is a label for the failure message.
assert_contains() {
    local haystack="$1"
    local needle="$2"
    local label="$3"

    if ! echo "$haystack" | grep -qF -- "$needle"; then
        print_fail "expected to find '$needle' ($label)"
        echo "--- output ---"
        echo "$haystack"
        echo "---"
        exit 1
    fi
}

# Build pilum with coverage instrumentation
print_header "Building Pilum (with coverage)"
cd "$PROJECT_ROOT"
mkdir -p dist
mkdir -p "$PROJECT_ROOT/coverage/e2e"
go build -cover -o dist/pilum .
print_success "Build complete (coverage-instrumented)"

# Set coverage directory for all test runs
export GOCOVERDIR="$PROJECT_ROOT/coverage/e2e"

# Test 1: GCP Cloud Run Recipe (4 steps)
print_header "Test 1: GCP Cloud Run Recipe (4 steps)"
print_test "Running deploy for GCP service..."
print_separator

cd "$FIXTURES/gcp-service"
CI=true "$PILUM" deploy --recipe-path "$RECIPES" 2>/dev/null

print_separator
print_success "GCP Cloud Run test complete"

# Test 2: AWS Lambda Recipe (3 steps)
print_header "Test 2: AWS Lambda Recipe (3 steps)"
print_test "Running deploy for AWS service..."
print_separator

cd "$FIXTURES/aws-service"
CI=true "$PILUM" deploy --recipe-path "$RECIPES" 2>/dev/null

print_separator
print_success "AWS Lambda test complete"

# Test 3: Homebrew Recipe (3 steps)
print_header "Test 3: Homebrew Recipe (3 steps)"
print_test "Running deploy for Homebrew service..."
print_separator

cd "$FIXTURES/homebrew-service"
CI=true "$PILUM" deploy --recipe-path "$RECIPES" 2>/dev/null

print_separator
print_success "Homebrew test complete"

# Test 4: Multi-service with different recipes (heterogeneous steps)
print_header "Test 4: Multi-Service Deployment (Mixed Recipes)"
print_test "Running deploy for all services simultaneously..."
print_test "This tests the step barrier behavior with different step counts:"
print_test "  - GCP: 4 steps"
print_test "  - AWS: 3 steps"
print_test "  - Homebrew: 3 steps"
print_separator

cd "$FIXTURES"
CI=true "$PILUM" deploy --recipe-path "$RECIPES" 2>/dev/null

print_separator
print_success "Multi-service test complete"

# Test 5: Dry-run mode
print_header "Test 5: Dry-Run Mode"
print_test "Running dry-run to show commands without execution..."
print_separator

cd "$FIXTURES"
"$PILUM" deploy --recipe-path "$RECIPES" --dry-run 2>/dev/null

print_separator
print_success "Dry-run test complete"

# Test 6: Publish (excludes deploy step)
print_header "Test 6: Publish Mode (Excludes Deploy Steps)"
print_test "Running publish which should skip deploy-related steps..."
print_separator

cd "$FIXTURES/gcp-service"
CI=true "$PILUM" publish --recipe-path "$RECIPES" 2>/dev/null

print_separator
print_success "Publish test complete"

# Test 7: GitHub commit status — env detection
print_header "Test 7: GitHub Commit Status — Env Detection"
print_test "Running deploy with --github-status and mock env vars..."
print_separator

cd "$FIXTURES/gcp-service"
rm -rf .pilum
GITHUB_ACTIONS=true GITHUB_TOKEN=test-token GITHUB_SHA=abc123 GITHUB_REPOSITORY=test-org/test-repo \
    CI=true "$PILUM" deploy --recipe-path "$RECIPES" --github-status 2>/dev/null

print_success "GitHub status test passed (no crash with mock env)"

print_separator
print_success "GitHub commit status test complete"

# Test 8: New recipe types (gcp-artifact-registry-image, gcp-cloud-run-from-image,
# gcp-cloud-run sidecars) + env-vars merge bug fix.
#
# Test 8 splits into two layers:
#
#   A) Orchestration — run the binary against MOCKED test recipes and confirm
#      the step graph executes cleanly. Same shape as Tests 1-7. Catches:
#      "the recipe loads, the step handler dispatches, the pipeline finishes."
#
#   B) Command emission — run the binary against the REAL embedded recipes
#      with --dry-run, capture the would-be gcloud commands, and grep for the
#      specific flags. Catches: "we changed the command builder and broke the
#      output shape" — the regression layer that Tests 1-7 don't have.
print_header "Test 8: New Recipe Types + Sidecars + env-vars Merge"

# 8A1: gcp-artifact-registry-image orchestration.
print_test "8A1: gcp-artifact-registry-image (3 steps, mocked)"
cd "$FIXTURES/gcp-image-publish"
rm -rf .pilum
CI=true "$PILUM" deploy --recipe-path "$RECIPES"
print_separator
print_success "8A1 complete"

# 8A2: gcp-cloud-run-from-image orchestration.
print_test "8A2: gcp-cloud-run-from-image (1 step, mocked)"
cd "$FIXTURES/gcp-from-image"
rm -rf .pilum
CI=true "$PILUM" deploy --recipe-path "$RECIPES"
print_separator
print_success "8A2 complete"

# 8A3: gcp-cloud-run with sidecars orchestration.
print_test "8A3: gcp-cloud-run with sidecars (4 steps, mocked)"
cd "$FIXTURES/gcp-sidecars"
rm -rf .pilum
CI=true "$PILUM" deploy --recipe-path "$RECIPES"
print_separator
print_success "8A3 complete"

# 8B1: from-image deploy must reference the pre-built image verbatim.
# Without --recipe-path, the binary loads its EMBEDDED real recipes — so
# --dry-run shows the actual gcloud command, not a mock echo.
print_test "8B1: from-image deploy emits explicit image reference"
cd "$FIXTURES/gcp-from-image"
rm -rf .pilum
FROM_IMAGE_OUTPUT=$(CI=true "$PILUM" deploy --dry-run 2>&1 || true)
assert_contains "$FROM_IMAGE_OUTPUT" \
    "us-central1-docker.pkg.dev/shared/platform/platform-base:v0.5.0" \
    "deploy command must reference the exact image from pilum.yaml, not construct one"
assert_contains "$FROM_IMAGE_OUTPUT" "gcloud" "command must use gcloud"
assert_contains "$FROM_IMAGE_OUTPUT" "run" "command must be a Cloud Run deploy"
print_separator
print_success "8B1 complete"

# 8B2: multi-container deploys emit one --container=NAME group per container
# in the right order. This is the core of the sidecar feature; if the output
# shape is wrong, every prod deploy with sidecars fails at the gcloud layer.
print_test "8B2: sidecars emit multi-container gcloud flags"
cd "$FIXTURES/gcp-sidecars"
rm -rf .pilum
SIDECARS_OUTPUT=$(CI=true "$PILUM" deploy --dry-run 2>&1 || true)

# Ingress + sidecar = at least two --container declarations.
SIDECAR_CONTAINER_COUNT=$(echo "$SIDECARS_OUTPUT" | grep -o -- "--container" | wc -l | tr -d ' ')
if [ "$SIDECAR_CONTAINER_COUNT" -lt "2" ]; then
    print_fail "expected at least 2 --container flags (ingress + 1 sidecar); got $SIDECAR_CONTAINER_COUNT"
    echo "--- output ---"
    echo "$SIDECARS_OUTPUT"
    echo "---"
    exit 1
fi

assert_contains "$SIDECARS_OUTPUT" "multi-container-svc" "ingress container named after the service"
assert_contains "$SIDECARS_OUTPUT" "telemetry-sidecar" "sidecar container name appears"
assert_contains "$SIDECARS_OUTPUT" "us-docker.pkg.dev/shared/obs/otel-collector:v0.1.0" \
    "sidecar image reference appears"
assert_contains "$SIDECARS_OUTPUT" "--port=8080" \
    "explicit --port required in multi-container mode (no implicit default)"
assert_contains "$SIDECARS_OUTPUT" "--depends-on=multi-container-svc" \
    "sidecar depends_on must render as --depends-on= flag"
print_separator
print_success "8B2 complete"

# 8B3: image-publish honors image_name + version overrides. Caught a real
# bug during initial test development — the --tag CLI flag defaulted to
# "latest" and was short-circuiting the YAML `version:` field, breaking
# the determinism guarantee for pinned platform images.
print_test "8B3: image-publish service uses image_name + version overrides"
cd "$FIXTURES/gcp-image-publish"
rm -rf .pilum
IMAGE_PUBLISH_OUTPUT=$(CI=true "$PILUM" deploy --dry-run 2>&1 || true)

assert_contains "$IMAGE_PUBLISH_OUTPUT" "platform-base:v0.5.0" \
    "image_name (platform-base) + version (v0.5.0) overrides must apply to the docker tag"

if echo "$IMAGE_PUBLISH_OUTPUT" | grep -q "deploy to cloud run"; then
    print_fail "gcp-artifact-registry-image must NOT emit a deploy step"
    echo "--- output ---"
    echo "$IMAGE_PUBLISH_OUTPUT"
    echo "---"
    exit 1
fi
print_separator
print_success "8B3 complete"

# 8B4: regression test for the env-vars merge fix. Before the fix,
# cloud_run.env_vars: (nested) was read by the compose generator but silently
# dropped from the real Cloud Run deploy command. This test verifies the
# nested form now reaches --set-env-vars end-to-end.
print_test "8B4: cloud_run.env_vars now reaches real deploys"

TMP_FIXTURE=$(mktemp -d)
trap 'rm -rf "$TMP_FIXTURE"' EXIT

cat >"$TMP_FIXTURE/pilum.yaml" <<'YAML'
name: env-merge-test
type: gcp-cloud-run
provider: gcp
project: test-project
region: us-central1
registry_name: test-project
template: golang-api.v1.dockerfile

build:
  language: go
  cmd: "echo 'build...'"

cloud_run:
  memory: 512Mi
  # The headline of the env-vars merge fix: this used to silently disappear
  # from the deploy command (only the compose generator read it). After
  # the fix, NewServiceInfo merges it into svc.EnvVars so the deploy path
  # sees it too.
  env_vars:
    NESTED_BUG_FIX_CANARY: "made-it"
YAML

cat >"$TMP_FIXTURE/main.go" <<'GO'
package main

func main() {}
GO

cd "$TMP_FIXTURE"
ENV_MERGE_OUTPUT=$(CI=true "$PILUM" deploy --dry-run 2>&1 || true)
assert_contains "$ENV_MERGE_OUTPUT" \
    "NESTED_BUG_FIX_CANARY=made-it" \
    "cloud_run.env_vars must reach the real gcloud --set-env-vars output (regression test for the merge fix)"
print_separator
print_success "8B4 complete"

print_success "Test 8 complete (all 7 sub-tests passed)"

# Summary
print_header "All E2E Tests Complete"
echo -e "${GREEN}✓${RESET} Test 1: GCP Cloud Run (4 steps)"
echo -e "${GREEN}✓${RESET} Test 2: AWS Lambda (3 steps)"
echo -e "${GREEN}✓${RESET} Test 3: Homebrew (3 steps)"
echo -e "${GREEN}✓${RESET} Test 4: Multi-service mixed recipes"
echo -e "${GREEN}✓${RESET} Test 5: Dry-run mode"
echo -e "${GREEN}✓${RESET} Test 6: Publish mode"
echo -e "${GREEN}✓${RESET} Test 7: GitHub commit status — env detection"
echo -e "${GREEN}✓${RESET} Test 8: New recipe types + sidecars + env-vars merge"
echo ""

# Generate coverage report from E2E tests
print_header "Generating E2E Coverage Report"
cd "$PROJECT_ROOT"
go tool covdata textfmt -i=coverage/e2e -o=coverage-e2e.out 2>/dev/null || true
if [ -f coverage-e2e.out ]; then
    echo -e "${GREEN}✓${RESET} E2E coverage saved to coverage-e2e.out"
    echo ""
    echo "To merge with unit test coverage:"
    echo "  go test ./... -coverprofile=coverage-unit.out"
    echo "  go tool covdata merge -i=coverage/e2e -o=coverage/merged"
    echo "  go tool covdata textfmt -i=coverage/merged -o=coverage-merged.out"
fi
echo ""
