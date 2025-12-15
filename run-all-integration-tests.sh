#!/bin/bash

# Run all integration tests for the refactored API
# This script runs comprehensive tests to verify the refactoring is working correctly

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}🧪 Running All Integration Tests${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Check if server is running
BASE_URL=${MAGDA_API_URL:-"http://localhost:8080"}
if ! curl -s -f "$BASE_URL/health" > /dev/null 2>&1; then
    echo -e "${RED}❌ Server is not running at $BASE_URL${NC}"
    echo "   Please start the server first: make dev"
    exit 1
fi

# Test scripts to run
TESTS=(
    "test-magda.sh:MAGDA Integration Test"
    "test-magda-dsl-e2e.sh:MAGDA DSL E2E Test"
)

PASSED=0
FAILED=0
SKIPPED=0

for test_info in "${TESTS[@]}"; do
    IFS=':' read -r test_script test_name <<< "$test_info"

    if [ ! -f "$test_script" ]; then
        echo -e "${YELLOW}⚠️  Skipping $test_name: $test_script not found${NC}"
        ((SKIPPED++))
        continue
    fi

    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}▶️  Running: $test_name${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    # Run test and capture output (let it run fully, don't stop on errors)
    set +e
    TEST_OUTPUT=$(bash "$test_script" 2>&1)
    TEST_EXIT=$?
    set -e

    # Show test output
    echo "$TEST_OUTPUT"
    echo ""

    # Determine result from exit code and output
    if [ $TEST_EXIT -eq 0 ]; then
        echo -e "${GREEN}✅ $test_name: PASSED${NC}"
        ((PASSED++))
    elif [ $TEST_EXIT -eq 1 ]; then
        echo -e "${RED}❌ $test_name: FAILED${NC}"
        ((FAILED++))
    else
        echo -e "${YELLOW}⚠️  $test_name: SKIPPED${NC}"
        ((SKIPPED++))
    fi
    echo ""
done

# Summary
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}📊 Test Summary${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Passed: $PASSED${NC}"
echo -e "${RED}❌ Failed: $FAILED${NC}"
echo -e "${YELLOW}⚠️  Skipped: $SKIPPED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}💥 Some tests failed${NC}"
    exit 1
fi
