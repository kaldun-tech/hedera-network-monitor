#!/bin/bash
# Pre-commit checks that don't require the monitor service running
# Run this before committing code

set -e

echo "=== Running offline pre-commit checks ==="
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

fail() {
    echo -e "${RED}✗ FAILED: $1${NC}"
    exit 1
}

pass() {
    echo -e "${GREEN}✓ PASSED: $1${NC}"
}

# Check 1: Format code
echo "1. Checking code formatting..."
if ! make fmt > /dev/null 2>&1; then
    fail "Code formatting"
fi
pass "Code formatting"
echo ""

# Check 2: Run linter
echo "2. Running linter..."
if ! make lint > /dev/null 2>&1; then
    fail "Linter checks"
fi
pass "Linter checks"
echo ""

# Check 3: Run unit tests
echo "3. Running unit tests..."
if ! make test > /dev/null 2>&1; then
    fail "Unit tests"
fi
pass "Unit tests"
echo ""

# Check 4: Build all binaries
echo "4. Building binaries..."
if ! make build > /dev/null 2>&1; then
    fail "Build"
fi
pass "Build"
echo ""

# Check 5: Verify dependencies
echo "5. Checking go.mod..."
if ! go mod tidy > /dev/null 2>&1; then
    fail "go mod tidy"
fi
if [ -n "$(git diff go.mod go.sum)" ]; then
    fail "go.mod/go.sum out of sync (run 'go mod tidy')"
fi
pass "Dependencies"
echo ""

echo "=== All offline checks passed! ==="
echo "You're ready to commit."
