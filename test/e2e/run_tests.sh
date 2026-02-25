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

# Test 7: Build cache — first run populates cache
print_header "Test 7: Build Cache — First Run Populates Cache"
print_test "Running deploy to populate build cache..."
print_separator

cd "$FIXTURES/gcp-service"
rm -rf .pilum
CI=true "$PILUM" deploy --recipe-path "$RECIPES" --tag=v1.0.0 2>/dev/null

if [ -f .pilum/build-cache.json ]; then
    print_success "Cache file created at .pilum/build-cache.json"
else
    echo -e "${RED}✗ Cache file not found${RESET}"
    exit 1
fi

print_separator
print_success "Build cache population test complete"

# Test 8: Build cache — second run hits cache (same tag)
print_header "Test 8: Build Cache — Cache Hit (Same Tag)"
print_test "Running deploy again with same tag — expecting cache hit..."
print_separator

cd "$FIXTURES/gcp-service"
CACHE_OUTPUT=$(CI=true "$PILUM" deploy --recipe-path "$RECIPES" --tag=v1.0.0 --debug 2>&1)

if echo "$CACHE_OUTPUT" | grep -q "Cache hit"; then
    print_success "Cache hit detected in debug output"
else
    echo -e "${RED}✗ Expected 'Cache hit' in debug output${RESET}"
    echo "$CACHE_OUTPUT"
    exit 1
fi

print_separator
print_success "Build cache hit test complete"

# Test 9: Build cache — different tag triggers rebuild
print_header "Test 9: Build Cache — Different Tag Triggers Rebuild"
print_test "Running deploy with different tag — first build step should rebuild..."
print_separator

cd "$FIXTURES/gcp-service"
CACHE_OUTPUT=$(CI=true "$PILUM" deploy --recipe-path "$RECIPES" --tag=v2.0.0 --debug 2>&1)

# The first build step should execute (not be cached) because the tag changed.
# Subsequent build steps may hit cache after the first updates the cache entry.
if echo "$CACHE_OUTPUT" | grep -q "Compiling Go binary"; then
    print_success "First build step rebuilt (tag change detected)"
else
    echo -e "${RED}✗ Expected first build step to rebuild after tag change${RESET}"
    echo "$CACHE_OUTPUT"
    exit 1
fi

print_separator
print_success "Build cache tag change test complete"

# Test 10: Build cache — --no-cache forces rebuild
print_header "Test 10: Build Cache — --no-cache Forces Rebuild"
print_test "Running deploy with --no-cache — expecting forced rebuild..."
print_separator

cd "$FIXTURES/gcp-service"
CACHE_OUTPUT=$(CI=true "$PILUM" deploy --recipe-path "$RECIPES" --tag=v2.0.0 --no-cache --debug 2>&1)

if echo "$CACHE_OUTPUT" | grep -q "Cache hit"; then
    echo -e "${RED}✗ Unexpected cache hit (--no-cache was set)${RESET}"
    echo "$CACHE_OUTPUT"
    exit 1
else
    print_success "No cache hit (--no-cache forced rebuild)"
fi

print_separator
print_success "Build cache --no-cache test complete"

# Test 11: Build cache — file change triggers rebuild
print_header "Test 11: Build Cache — File Change Triggers Rebuild"
print_test "Modifying source file and re-running — first build step should rebuild..."
print_separator

cd "$FIXTURES/gcp-service"
echo "// changed" >> main.go
CACHE_OUTPUT=$(CI=true "$PILUM" deploy --recipe-path "$RECIPES" --tag=v2.0.0 --debug 2>&1)

# The first build step should execute (not be cached) because the file hash changed.
if echo "$CACHE_OUTPUT" | grep -q "Compiling Go binary"; then
    print_success "First build step rebuilt (file change detected)"
else
    echo -e "${RED}✗ Expected first build step to rebuild after file change${RESET}"
    echo "$CACHE_OUTPUT"
    git checkout main.go 2>/dev/null || true
    exit 1
fi

# Restore the file
git checkout main.go 2>/dev/null || true

print_separator
print_success "Build cache file change test complete"

# Test 12: GitHub commit status — env detection
print_header "Test 12: GitHub Commit Status — Env Detection"
print_test "Running deploy with --github-status and mock env vars..."
print_separator

cd "$FIXTURES/gcp-service"
rm -rf .pilum
GITHUB_ACTIONS=true GITHUB_TOKEN=test-token GITHUB_SHA=abc123 GITHUB_REPOSITORY=test-org/test-repo \
    CI=true "$PILUM" deploy --recipe-path "$RECIPES" --github-status 2>/dev/null

print_success "GitHub status test passed (no crash with mock env)"

print_separator
print_success "GitHub commit status test complete"

# Cleanup
rm -rf "$FIXTURES/gcp-service/.pilum"

# Summary
print_header "All E2E Tests Complete"
echo -e "${GREEN}✓${RESET} Test 1: GCP Cloud Run (4 steps)"
echo -e "${GREEN}✓${RESET} Test 2: AWS Lambda (3 steps)"
echo -e "${GREEN}✓${RESET} Test 3: Homebrew (3 steps)"
echo -e "${GREEN}✓${RESET} Test 4: Multi-service mixed recipes"
echo -e "${GREEN}✓${RESET} Test 5: Dry-run mode"
echo -e "${GREEN}✓${RESET} Test 6: Publish mode"
echo -e "${GREEN}✓${RESET} Test 7: Build cache — first run populates cache"
echo -e "${GREEN}✓${RESET} Test 8: Build cache — cache hit (same tag)"
echo -e "${GREEN}✓${RESET} Test 9: Build cache — different tag triggers rebuild"
echo -e "${GREEN}✓${RESET} Test 10: Build cache — --no-cache forces rebuild"
echo -e "${GREEN}✓${RESET} Test 11: Build cache — file change triggers rebuild"
echo -e "${GREEN}✓${RESET} Test 12: GitHub commit status — env detection"
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
