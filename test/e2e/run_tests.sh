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

# Summary
print_header "All E2E Tests Complete"
echo -e "${GREEN}✓${RESET} Test 1: GCP Cloud Run (4 steps)"
echo -e "${GREEN}✓${RESET} Test 2: AWS Lambda (3 steps)"
echo -e "${GREEN}✓${RESET} Test 3: Homebrew (3 steps)"
echo -e "${GREEN}✓${RESET} Test 4: Multi-service mixed recipes"
echo -e "${GREEN}✓${RESET} Test 5: Dry-run mode"
echo -e "${GREEN}✓${RESET} Test 6: Publish mode"
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
