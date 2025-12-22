#!/bin/bash

# Run all smoke tests
# Usage: ./run-all.sh [base-url]

set -e

BASE_URL=${1:-"http://localhost:8080"}
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🧪 MAGDA API Smoke Tests"
echo "   Base URL: $BASE_URL"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

PASSED=0
FAILED=0

run_test() {
    local name=$1
    local script=$2

    echo "▶️  $name"
    if bash "$SCRIPT_DIR/$script" "$BASE_URL" > /tmp/test_output.txt 2>&1; then
        echo -e "${GREEN}   ✅ PASSED${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}   ❌ FAILED${NC}"
        cat /tmp/test_output.txt | sed 's/^/   /'
        FAILED=$((FAILED + 1))
    fi
    echo ""
}

# Health check (always run first)
run_test "Health Check" "health.sh"

# Metrics
run_test "Metrics" "metrics.sh"

# API endpoints (require OPENAI_API_KEY)
echo -e "${YELLOW}Note: API tests require OPENAI_API_KEY on the server${NC}"
echo ""

run_test "MAGDA Chat" "chat.sh"
run_test "JSFX Generation" "jsfx.sh"

# Summary
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "   ${GREEN}Passed: $PASSED${NC}"
echo -e "   ${RED}Failed: $FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All smoke tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed${NC}"
    exit 1
fi
