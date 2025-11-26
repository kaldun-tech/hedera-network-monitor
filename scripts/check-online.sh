#!/bin/bash
# Pre-push checks that require the monitor service running
# Run this before pushing to verify the system works end-to-end
#
# To start the monitor service for online checks:
#   1. Build: make build
#   2. Ensure config/config.yaml exists (copy from config.example.yaml if needed)
#   3. Start service: ./monitor
#   4. In another terminal, run this script: ./scripts/check-online.sh
#
# Optional: Set API_URL and TEST_ACCOUNT environment variables to override defaults:
#   API_URL=http://localhost:9000 TEST_ACCOUNT=0.0.1000 ./scripts/check-online.sh

set -e

echo "=== Running online pre-push checks ==="
echo "Note: This requires the monitor service to be running"
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

fail() {
    echo -e "${RED}✗ FAILED: $1${NC}"
    exit 1
}

pass() {
    echo -e "${GREEN}✓ PASSED: $1${NC}"
}

warn() {
    echo -e "${YELLOW}⚠ WARNING: $1${NC}"
}

# Configuration
API_URL="${API_URL:-http://localhost:8080}"
TEST_ACCOUNT="${TEST_ACCOUNT:-0.0.5000}"
TIMEOUT=5

# Check 1: Monitor is running (API is accessible)
echo "1. Checking if monitor service is running..."
if ! curl -s -m $TIMEOUT "$API_URL/health" > /dev/null 2>&1; then
    fail "Monitor service is not running at $API_URL"
fi
pass "Monitor service is running"
echo ""

# Check 2: Health endpoint
echo "2. Checking health endpoint..."
HEALTH=$(curl -s -m $TIMEOUT "$API_URL/health")
if echo "$HEALTH" | grep -q '"status":"healthy"'; then
    pass "Health check"
else
    warn "Health endpoint returned unexpected response"
fi
echo ""

# Check 3: Metrics endpoint
echo "3. Checking metrics API endpoint..."
if curl -s -m $TIMEOUT "$API_URL/api/v1/metrics?limit=1" > /dev/null 2>&1; then
    pass "Metrics endpoint"
else
    fail "Metrics endpoint is not responding"
fi
echo ""

# Check 4: Alert rules are loaded
echo "4. Checking alert rules..."
ALERTS=$(curl -s -m $TIMEOUT "$API_URL/api/v1/alerts")
ALERT_COUNT=$(echo "$ALERTS" | grep -o '"id"' | wc -l)
if [ "$ALERT_COUNT" -gt 0 ]; then
    pass "Alert rules loaded ($ALERT_COUNT rules)"
else
    warn "No alert rules found (check config/config.yaml)"
fi
echo ""

# Check 5: CLI can query balance
echo "5. Testing CLI account balance query..."
if ./hmon account balance "$TEST_ACCOUNT" > /dev/null 2>&1; then
    pass "CLI balance query works"
else
    warn "CLI balance query failed (check credentials and network)"
fi
echo ""

# Check 6: CLI network status
echo "6. Testing CLI network status..."
NETWORK_STATUS=$(./hmon network status 2>&1)
if echo "$NETWORK_STATUS" | grep -q "Network Status"; then
    pass "CLI network status works"
else
    warn "CLI network status returned unexpected format"
fi
echo ""

# Check 7: Metrics are being collected
echo "7. Checking if metrics are being collected..."
METRICS_JSON=$(curl -s -m $TIMEOUT "$API_URL/api/v1/metrics?limit=100")
# Count "Name" fields (capitalized) in the metrics array
METRIC_COUNT=$(echo "$METRICS_JSON" | grep -o '"Name"' | wc -l)

if [ "$METRIC_COUNT" -gt 0 ]; then
    pass "Metrics are being collected ($METRIC_COUNT metrics)"
else
    warn "No metrics collected yet (wait 30+ seconds for collectors to run)"
fi
echo ""

echo "=== Online checks complete ==="
echo "Ready to push!"
