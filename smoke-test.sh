#!/bin/bash

# AIDEAS API Smoke Test
# Quick test to verify the API is deployed and working correctly
# Usage: ./smoke-test.sh [base-url]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

BASE_URL=${1:-"https://api.musicalaideas.com"}
FAILED=0

# Load environment variables for authentication
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$SCRIPT_DIR"

# Always load from .env file first
if [ -f "$PROJECT_ROOT/.env" ]; then
    set -a
    while IFS= read -r line || [ -n "$line" ]; do
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ -z "${line// }" ]] && continue
        export "$line" 2>/dev/null || true
    done < "$PROJECT_ROOT/.env"
    set +a
fi

# Get auth token
TOKEN=""
if [ -n "$AIDEAS_EMAIL" ] && [ -n "$AIDEAS_PASSWORD" ]; then
    echo "🔐 Authenticating..."
    # Build JSON payload using jq to handle escaping properly
    AUTH_PAYLOAD=$(jq -n --arg email "$AIDEAS_EMAIL" --arg password "$AIDEAS_PASSWORD" '{email: $email, password: $password}')

    # Debug: Show what we're sending (but don't print the password)
    if [ "${DEBUG:-0}" = "1" ]; then
        echo "DEBUG: Sending to $BASE_URL/api/auth/register/beta"
        echo "DEBUG: Payload email: $AIDEAS_EMAIL"
    fi

    # Try register first, then login
    AUTH_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/register/beta" \
        -H "Content-Type: application/json" \
        -d "$AUTH_PAYLOAD")

    if echo "$AUTH_RESPONSE" | grep -q "access_token"; then
        TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.access_token' 2>/dev/null)
        echo "✅ Registered and authenticated"
    else
        # Try login
        AUTH_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
            -H "Content-Type: application/json" \
            -d "$AUTH_PAYLOAD")
        if echo "$AUTH_RESPONSE" | grep -q "access_token"; then
            TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.access_token' 2>/dev/null)
            if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
                echo "✅ Logged in and authenticated"
            else
                echo -e "${RED}❌ Failed to extract token from response${NC}"
                echo "   Response: $AUTH_RESPONSE"
            fi
        else
            echo -e "${RED}❌ Login failed${NC}"
            echo "   Response: $AUTH_RESPONSE"
        fi
    fi
    echo ""
elif [ -z "$AIDEAS_EMAIL" ] || [ -z "$AIDEAS_PASSWORD" ]; then
    echo -e "${YELLOW}⚠️  Warning: AIDEAS_EMAIL or AIDEAS_PASSWORD not set${NC}"
    echo "   Looking for .env in: $PROJECT_ROOT"
    if [ ! -f "$PROJECT_ROOT/.env" ]; then
        echo "   No .env file found"
    fi
    echo ""
fi

if [ -z "$TOKEN" ]; then
    echo -e "${YELLOW}⚠️  Warning: No authentication token. Generation test will be skipped.${NC}"
    echo "   To test MCP tools, ensure AIDEAS_EMAIL and AIDEAS_PASSWORD are set in .env"
    echo ""
    SKIP_GENERATION=true
else
    SKIP_GENERATION=false
fi

echo "🔍 Running smoke tests for AIDEAS API"
echo "Base URL: $BASE_URL"
echo ""

# Test 1: Health Check
echo -n "✓ Testing health endpoint... "
HEALTH_RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/health")
HTTP_CODE=$(echo "$HEALTH_RESPONSE" | tail -1)
BODY=$(echo "$HEALTH_RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ] && echo "$BODY" | grep -q "healthy"; then
    echo -e "${GREEN}PASS${NC}"
    echo "  Response: $BODY"
else
    echo -e "${RED}FAIL${NC}"
    echo "  Expected: 200 with {\"status\":\"healthy\"}"
    echo "  Got: HTTP $HTTP_CODE - $BODY"
    FAILED=$((FAILED + 1))
fi
echo ""

# Test 2: Generation endpoint (requires auth)
if [ "$SKIP_GENERATION" = "true" ]; then
    echo -n "✓ Testing generation endpoint... "
    echo -e "${YELLOW}SKIPPED (no auth token)${NC}"
    echo ""
else
    echo -n "✓ Testing generation endpoint... "
GENERATION_PAYLOAD='{
  "model": "gpt-5-mini",
  "input_array": [
    {
      "role": "user",
      "content": "{\"user_prompt\": \"Continue this chord progression with harmonically appropriate next chords.\"}"
    },
    {
      "role": "user",
      "content": "{\"notes\": [{\"midiNoteNumber\": 60, \"velocity\": 100, \"startBeats\": 0.0, \"durationBeats\": 2.0}, {\"midiNoteNumber\": 64, \"velocity\": 100, \"startBeats\": 0.0, \"durationBeats\": 2.0}, {\"midiNoteNumber\": 67, \"velocity\": 100, \"startBeats\": 0.0, \"durationBeats\": 2.0}]}"
    }
  ],
  "stream": false,
  "reasoning_mode": "minimal"
}'

# Build headers with auth if available
HEADERS=("-H" "Content-Type: application/json")
if [ -n "$TOKEN" ]; then
    HEADERS+=("-H" "Authorization: Bearer $TOKEN")
fi

GENERATION_RESPONSE=$(curl -s -w "\n%{http_code}" \
  -X POST "$BASE_URL/api/v1/generations" \
  "${HEADERS[@]}" \
  -d "$GENERATION_PAYLOAD")

HTTP_CODE=$(echo "$GENERATION_RESPONSE" | tail -1)
BODY=$(echo "$GENERATION_RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    # Accept either genResults or output_parsed format
    if echo "$BODY" | grep -qE "(genResults|output_parsed|choices)"; then
        echo -e "${GREEN}PASS${NC}"
        echo "  Successfully generated musical output"
        # Pretty print the response (if jq is available)
        if command -v jq &> /dev/null; then
            # Try both formats
            NOTE_COUNT=$(echo "$BODY" | jq -r '(.genResults[0].notes // .output_parsed.choices[0].notes) | length' 2>/dev/null || echo "unknown")
            DESCRIPTION=$(echo "$BODY" | jq -r '(.genResults[0].description // .output_parsed.choices[0].description)' 2>/dev/null || echo "")
            REQUEST_ID=$(echo "$BODY" | jq -r '.request_id // "unknown"' 2>/dev/null)

            # Extract usage information
            TOTAL_TOKENS=$(echo "$BODY" | jq -r '.usage.total_tokens // "unknown"' 2>/dev/null)
            INPUT_TOKENS=$(echo "$BODY" | jq -r '.usage.input_tokens // "unknown"' 2>/dev/null)
            OUTPUT_TOKENS=$(echo "$BODY" | jq -r '.usage.output_tokens // "unknown"' 2>/dev/null)
            REASONING_TOKENS=$(echo "$BODY" | jq -r '.usage.output_tokens_details.reasoning_tokens // 0' 2>/dev/null)

            # MCP usage
            MCP_USED=$(echo "$BODY" | jq -r '.mcpUsed // false' 2>/dev/null)
            MCP_CALLS=$(echo "$BODY" | jq -r '.mcpCalls // 0' 2>/dev/null)
            MCP_TOOLS=$(echo "$BODY" | jq -r '.mcpTools // []' 2>/dev/null)

            echo "  Request ID: $REQUEST_ID"
            echo "  Notes: $NOTE_COUNT"
            echo "  Description: $DESCRIPTION"
            echo ""
            echo "  📊 Token Usage:"
            echo "    Total: $TOTAL_TOKENS tokens"
            echo "    Input: $INPUT_TOKENS tokens"
            echo "    Output: $OUTPUT_TOKENS tokens"
            if [ "$REASONING_TOKENS" != "0" ]; then
                echo "    Reasoning: $REASONING_TOKENS tokens"
            fi
            echo ""
            echo "  🔧 MCP:"
            echo "    Used: $MCP_USED"
            if [ "$MCP_CALLS" != "0" ]; then
                echo "    Calls: $MCP_CALLS"
                if [ "$MCP_TOOLS" != "[]" ] && [ "$MCP_TOOLS" != "null" ]; then
                    echo "    Tools: $MCP_TOOLS"
                fi
            fi
        fi
    else
        echo -e "${YELLOW}PARTIAL${NC}"
        echo "  Response: $BODY"
    fi
else
    echo -e "${RED}FAIL${NC}"
    echo "  Expected: 200 with musical output"
    echo "  Got: HTTP $HTTP_CODE"
    echo "  Response: $BODY"
    FAILED=$((FAILED + 1))
fi
fi
echo ""

# Test 3: CORS headers
echo -n "✓ Testing CORS headers... "
CORS_RESPONSE=$(curl -s -I -X OPTIONS "$BASE_URL/api/v1/generations" \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: POST")

if echo "$CORS_RESPONSE" | grep -q "Access-Control-Allow-Origin"; then
    echo -e "${GREEN}PASS${NC}"
    echo "  CORS headers present"
else
    echo -e "${YELLOW}WARNING${NC}"
    echo "  CORS headers not found (may not be critical)"
fi
echo ""

# Test 4: SSL/TLS certificate (only if HTTPS)
if [[ "$BASE_URL" == https://* ]]; then
    echo -n "✓ Testing SSL certificate... "
    CERT_INFO=$(curl -vI "$BASE_URL/health" 2>&1 | grep -E "(SSL connection|Server certificate)" || true)
    if [ -n "$CERT_INFO" ]; then
        echo -e "${GREEN}PASS${NC}"
        echo "  SSL/TLS connection established"
    else
        echo -e "${YELLOW}WARNING${NC}"
        echo "  Could not verify SSL certificate"
    fi
    echo ""
fi

# Summary
echo "======================================"
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All critical tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ $FAILED critical test(s) failed${NC}"
    exit 1
fi
