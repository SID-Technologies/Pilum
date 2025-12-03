#!/bin/bash
# Test multi-service deployment with mixed recipes
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$PROJECT_ROOT" && go build -o pilum .
cd "$SCRIPT_DIR/fixtures"
CI=true "$PROJECT_ROOT/pilum" deploy --recipe-path "$SCRIPT_DIR/recipes"
