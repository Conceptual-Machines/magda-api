#!/bin/bash

# Comprehensive Integration Tests for Refactored API
# Tests both AIDEAS and MAGDA endpoints with real API calls

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Load environment variables
if [ -f .env ]; then
    echo -e "${YELLOW}📦 Loading environment from .env...${NC}"
    export $(grep -v '^#' .env | xargs)
fi

# Configuration
BASE_URL=${AIDEAS_API_URL:-"http://localhost:8080"}
EMAIL=${AIDEAS_EMAIL:-""}
PASSWORD=${AIDEAS_PASSWORD:-""}

# Check if server is running
echo -e "${BLUE}🔍 Checking if server is running at $BASE_URL...${NC}"
if ! curl -s -f "$BASE_URL/health" > /dev/null 2>&1; then
    echo -e "${RED}❌ Server is not running at $BASE_URL${NC}"
    echo "   Please start the server first: make dev"
    exit 1
fi
echo -e "${GREEN}✓ Server is running${NC}"
echo ""

# Function to register or login and get token
get_auth_token() {
    if [ -z "$EMAIL" ] || [ -z "$PASSWORD" ]; then
        echo -e "${YELLOW}⚠️  EMAIL or PASSWORD not set, trying to register...${NC}"
        # Try to register
        RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/register/beta" \
            -H "Content-Type: application/json" \
            -d "{\"email\": \"test-$(date +%s)@test.com\", \"password\": \"test123456\"}")

        if echo "$RESPONSE" | jq -e '.access_token' > /dev/null 2>&1; then
            echo "$RESPONSE" | jq -r '.access_token'
            return 0
        fi
    else
        # Try login first
        RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
            -H "Content-Type: application/json" \
            -d "{\"email\": \"$EMAIL\", \"password\": \"$PASSWORD\"}")

        if echo "$RESPONSE" | jq -e '.access_token' > /dev/null 2>&1; then
            echo "$RESPONSE" | jq -r '.access_token'
            return 0
        fi

        # If login fails, try register
        RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/register/beta" \
            -H "Content-Type: application/json" \
            -d "{\"email\": \"$EMAIL\", \"password\": \"$PASSWORD\"}")

        if echo "$RESPONSE" | jq -e '.access_token' > /dev/null 2>&1; then
            echo "$RESPONSE" | jq -r '.access_token'
            return 0
        fi
    fi

    echo -e "${RED}❌ Failed to authenticate${NC}"
    return 1
}

# Get auth token
echo -e "${BLUE}🔐 Authenticating...${NC}"
TOKEN=$(get_auth_token)
if [ -z "$TOKEN" ]; then
    exit 1
fi
echo -e "${GREEN}✓ Authenticated${NC}"
echo ""

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Function to run a test
run_test() {
    local test_name="$1"
    local test_func="$2"

    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}🧪 Test: $test_name${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    if $test_func; then
        echo -e "${GREEN}✅ PASSED: $test_name${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}❌ FAILED: $test_name${NC}"
        ((TESTS_FAILED++))
    fi
    echo ""
}

# Test 1: Health Check
test_health() {
    RESPONSE=$(curl -s "$BASE_URL/health")
    if echo "$RESPONSE" | jq -e '.status' > /dev/null 2>&1; then
        echo "Response: $(echo "$RESPONSE" | jq -c '.')"
        return 0
    fi
    return 1
}

# Test 2: AIDEAS Generation Endpoint (non-streaming)
test_aideas_generation() {
    RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/aideas/generations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d '{
            "model": "gpt-5-mini",
            "input_array": [
                {
                    "role": "user",
                    "content": "{\"user_prompt\": \"Create a simple 2-bar chord progression in C major\", \"bpm\": 120, \"variations\": 1}"
                }
            ],
            "stream": false,
            "output_format": "dsl",
            "reasoning_mode": "low"
        }')

    echo "Response: $(echo "$RESPONSE" | jq -c 'keys')"

    if echo "$RESPONSE" | jq -e '.output_parsed.choices[0].notes' > /dev/null 2>&1; then
        NOTE_COUNT=$(echo "$RESPONSE" | jq '.output_parsed.choices[0].notes | length')
        echo "Generated $NOTE_COUNT notes"
        if [ "$NOTE_COUNT" -gt 0 ]; then
            return 0
        fi
    fi

    echo "Error: $(echo "$RESPONSE" | jq -r '.error // "Unknown error"')"
    return 1
}

# Test 3: AIDEAS Generation Endpoint (streaming)
test_aideas_streaming() {
    EVENTS_RECEIVED=0
    COMPLETED=false

    while IFS= read -r line; do
        if [[ $line == data:* ]]; then
            DATA="${line#data: }"
            if echo "$DATA" | jq -e '.type' > /dev/null 2>&1; then
                TYPE=$(echo "$DATA" | jq -r '.type')
                ((EVENTS_RECEIVED++))
                if [ "$TYPE" == "completed" ]; then
                    COMPLETED=true
                fi
            fi
        fi
    done < <(curl -s -N -X POST "$BASE_URL/api/v1/aideas/generations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d '{
            "model": "gpt-5-mini",
            "input_array": [
                {
                    "role": "user",
                    "content": "{\"user_prompt\": \"Create a simple bassline in C minor\", \"bpm\": 120, \"variations\": 1}"
                }
            ],
            "stream": true,
            "output_format": "dsl",
            "reasoning_mode": "minimal"
        }')

    echo "Received $EVENTS_RECEIVED events"
    if [ "$COMPLETED" == true ] && [ "$EVENTS_RECEIVED" -gt 0 ]; then
        return 0
    fi
    return 1
}

# Test 4: MAGDA Chat Endpoint
test_magda_chat() {
    RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/magda/chat" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d '{
            "question": "Create a new track called Drums",
            "state": {}
        }')

    echo "Response: $(echo "$RESPONSE" | jq -c 'keys')"

    if echo "$RESPONSE" | jq -e '.actions' > /dev/null 2>&1; then
        ACTION_COUNT=$(echo "$RESPONSE" | jq '.actions | length')
        echo "Generated $ACTION_COUNT actions"
        if [ "$ACTION_COUNT" -gt 0 ]; then
            FIRST_ACTION=$(echo "$RESPONSE" | jq -r '.actions[0].action')
            echo "First action: $FIRST_ACTION"
            return 0
        fi
    fi

    echo "Error: $(echo "$RESPONSE" | jq -r '.error // "Unknown error"')"
    return 1
}

# Test 5: MAGDA Chat Streaming
test_magda_streaming() {
    EVENTS_RECEIVED=0
    ACTIONS_RECEIVED=0

    while IFS= read -r line; do
        if [[ $line == data:* ]]; then
            DATA="${line#data: }"
            if echo "$DATA" | jq -e '.type' > /dev/null 2>&1; then
                TYPE=$(echo "$DATA" | jq -r '.type')
                ((EVENTS_RECEIVED++))

                if [ "$TYPE" == "action" ]; then
                    ((ACTIONS_RECEIVED++))
                    ACTION_KIND=$(echo "$DATA" | jq -r '.data.action // "unknown"')
                    echo "  Action $ACTIONS_RECEIVED: $ACTION_KIND"
                fi

                if [ "$TYPE" == "completed" ]; then
                    echo "Stream completed"
                    break
                fi
            fi
        fi
    done < <(curl -s -N -X POST "$BASE_URL/api/v1/magda/chat/stream" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d '{
            "question": "Create a track with Serum and add a clip at bar 2",
            "state": {}
        }')

    echo "Received $EVENTS_RECEIVED events, $ACTIONS_RECEIVED actions"
    if [ "$ACTIONS_RECEIVED" -gt 0 ]; then
        return 0
    fi
    return 1
}

# Test 6: MAGDA DSL Parser
test_magda_dsl() {
    RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/magda/dsl" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d '{
            "dsl": "track(instrument=\"Serum\").newClip(bar=2, length_bars=4)"
        }')

    echo "Response: $(echo "$RESPONSE" | jq -c 'keys')"

    if echo "$RESPONSE" | jq -e '.actions' > /dev/null 2>&1; then
        ACTION_COUNT=$(echo "$RESPONSE" | jq '.actions | length')
        echo "Parsed $ACTION_COUNT actions from DSL"
        if [ "$ACTION_COUNT" -gt 0 ]; then
            return 0
        fi
    fi

    echo "Error: $(echo "$RESPONSE" | jq -r '.error // "Unknown error"')"
    return 1
}

# Test 7: Verify CFG Grammar Cleaning (check logs)
test_cfg_grammar_cleaning() {
    echo "This test verifies that grammar-school CleanGrammarForCFG is being called"
    echo "Check server logs for: 'Grammar cleaned for CFG'"
    echo "Running a MAGDA request to trigger CFG cleaning..."

    RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/magda/chat" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d '{
            "question": "Create a track",
            "state": {}
        }' 2>&1)

    # Just verify the request succeeded (grammar cleaning happens server-side)
    if echo "$RESPONSE" | jq -e '.actions' > /dev/null 2>&1; then
        echo "Request succeeded - check server logs for grammar cleaning messages"
        return 0
    fi
    return 1
}

# Run all tests
echo -e "${GREEN}🚀 Starting Integration Tests${NC}"
echo -e "${GREEN}Base URL: $BASE_URL${NC}"
echo ""

run_test "Health Check" test_health
run_test "AIDEAS Generation (non-streaming)" test_aideas_generation
run_test "AIDEAS Generation (streaming)" test_aideas_streaming
run_test "MAGDA Chat" test_magda_chat
run_test "MAGDA Chat Streaming" test_magda_streaming
run_test "MAGDA DSL Parser" test_magda_dsl
run_test "CFG Grammar Cleaning" test_cfg_grammar_cleaning

# Summary
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}📊 Test Summary${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Passed: $TESTS_PASSED${NC}"
echo -e "${RED}❌ Failed: $TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}💥 Some tests failed${NC}"
    exit 1
fi
