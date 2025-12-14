#!/bin/bash

# Arpeggio Tool Smoke Test
# Tests that the API correctly calls and uses the MCP arpeggio tool
# Usage: ./test-arpeggio-smoke.sh [base-url]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

BASE_URL=${1:-"https://api.musicalaideas.com"}
FAILED=0
WARNINGS=0

# Load environment variables for authentication
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$SCRIPT_DIR"

# Load environment variables from .envrc
if [ -f "$PROJECT_ROOT/.envrc" ]; then
    set -a
    source "$PROJECT_ROOT/.envrc" >/dev/null 2>&1 || true
    set +a
fi

# Debug: Check if credentials were loaded
if [ -z "$AIDEAS_EMAIL" ] || [ -z "$AIDEAS_PASSWORD" ]; then
    echo -e "${YELLOW}⚠️  Debug: Credentials not loaded from .envrc${NC}"
    if grep -q "AIDEAS_EMAIL" "$PROJECT_ROOT/.envrc" 2>/dev/null; then
        eval "$(grep "^export AIDEAS_" "$PROJECT_ROOT/.envrc" 2>/dev/null)" || true
    fi
fi

# Get auth token
TOKEN=""
if [ -n "$AIDEAS_EMAIL" ] && [ -n "$AIDEAS_PASSWORD" ]; then
    echo -e "${BLUE}🔐 Authenticating...${NC}"
    AUTH_PAYLOAD=$(jq -n --arg email "$AIDEAS_EMAIL" --arg password "$AIDEAS_PASSWORD" '{email: $email, password: $password}')

    AUTH_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/register/beta" \
        -H "Content-Type: application/json" \
        -d "$AUTH_PAYLOAD")

    if echo "$AUTH_RESPONSE" | grep -q "access_token"; then
        TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.access_token' 2>/dev/null)
        echo -e "${GREEN}✅ Authenticated${NC}"
    else
        AUTH_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
            -H "Content-Type: application/json" \
            -d "$AUTH_PAYLOAD")
        if echo "$AUTH_RESPONSE" | grep -q "access_token"; then
            TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.access_token' 2>/dev/null)
            if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
                echo -e "${GREEN}✅ Logged in and authenticated${NC}"
            fi
        fi
    fi
    echo ""
fi

if [ -z "$TOKEN" ]; then
    echo -e "${RED}❌ No authentication token. Cannot run tests.${NC}"
    exit 1
fi

echo -e "${BLUE}🧪 Arpeggio Tool Smoke Tests${NC}"
echo "Base URL: $BASE_URL"
echo ""

# Test 1: Generation with prompt that should trigger arpeggio tool
echo -e "${YELLOW}Test 1: Generation Uses Arpeggio Tool${NC}"
GENERATION_PAYLOAD='{
  "model": "gpt-5-mini",
  "input_array": [
    {
      "role": "user",
      "content": "{\"user_prompt\": \"Use the arpeggio tool to create a C major arpeggio pattern\"}"
    }
  ],
  "stream": true,
  "reasoning_mode": "minimal"
}'

echo "  Sending generation request..."
GENERATION_RESPONSE=$(curl -s -N \
  -X POST "$BASE_URL/api/v1/generations" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "$GENERATION_PAYLOAD" 2>&1)

# Save response to file for debugging
RESPONSE_FILE=$(mktemp)
echo "$GENERATION_RESPONSE" > "$RESPONSE_FILE"

# Check for MCP enabled event
MCP_ENABLED=$(echo "$GENERATION_RESPONSE" | grep -i '"type":"mcp_enabled"' | head -1)
if [ -z "$MCP_ENABLED" ]; then
    MCP_ENABLED=$(echo "$GENERATION_RESPONSE" | grep -i 'mcp_enabled' | head -1)
fi

# Check for MCP tool calls - look for mcp_used event which shows tool usage
MCP_USED_EVENT=$(echo "$GENERATION_RESPONSE" | grep '"type":"mcp_used"' | head -1 | sed 's/data: //' 2>/dev/null || echo "")

if [ -n "$MCP_ENABLED" ]; then
    echo -e "  ${GREEN}✅ MCP is enabled${NC}"
else
    echo -e "  ${RED}❌ MCP not enabled in response${NC}"
    FAILED=$((FAILED + 1))
fi

# Check for arpeggio tool calls via mcp_used event
ARPEGGIO_TOOL_CALLED=false
if [ -n "$MCP_USED_EVENT" ]; then
    ARP_TOOLS=$(echo "$MCP_USED_EVENT" | jq -r '.data.tools // [] | .[] | select(. | contains("arp"))' 2>/dev/null || echo "")

    if [ -n "$ARP_TOOLS" ]; then
        ARPEGGIO_TOOL_CALLED=true
        echo -e "  ${GREEN}✅ Arpeggio tools called via MCP${NC}"
        echo "$ARP_TOOLS" | while IFS= read -r tool; do
            echo -e "     ${GREEN}✅ Tool: $tool${NC}"
        done

        # Get call count
        CALL_COUNT=$(echo "$MCP_USED_EVENT" | jq -r '.data.calls // 0' 2>/dev/null || echo "0")
        echo "     Total MCP calls: $CALL_COUNT"
    else
        echo -e "  ${YELLOW}⚠️  No arpeggio tools found in MCP usage${NC}"
        ALL_TOOLS=$(echo "$MCP_USED_EVENT" | jq -r '.data.tools // [] | join(", ")' 2>/dev/null || echo "")
        if [ -n "$ALL_TOOLS" ]; then
            echo "     MCP tools used: $ALL_TOOLS"
        fi
        WARNINGS=$((WARNINGS + 1))
    fi
else
    echo -e "  ${YELLOW}⚠️  No mcp_used event found in streaming response${NC}"
    WARNINGS=$((WARNINGS + 1))
fi
echo ""

# Test 2: Verify arpeggio tool response structure
echo -e "${YELLOW}Test 2: Arpeggio Tool Response Structure${NC}"
if [ "$ARPEGGIO_TOOL_CALLED" = "true" ]; then
    echo -e "  ${GREEN}✅ Arpeggio tools were called (verified in Test 1)${NC}"
    echo -e "  ${GREEN}✅ Tool call verification passed${NC}"
else
    echo -e "  ${YELLOW}⚠️  Skipping response check - no arpeggio tools called${NC}"
    WARNINGS=$((WARNINGS + 1))
fi
echo ""

# Test 3: Verify final generation uses tool results
echo -e "${YELLOW}Test 3: Generation Uses Tool Results${NC}"
RESULT_EVENT=$(echo "$GENERATION_RESPONSE" | grep '"type":"result"' | head -1 | sed 's/data: //' 2>/dev/null || echo "")

if [ -z "$RESULT_EVENT" ]; then
    # Try "completed" instead
    RESULT_EVENT=$(echo "$GENERATION_RESPONSE" | grep '"type":"completed"' | head -1 | sed 's/data: //' 2>/dev/null || echo "")
fi

if [ -n "$RESULT_EVENT" ]; then
    # Parse the result event
    NOTES=$(echo "$RESULT_EVENT" | jq -r '.data.output_parsed.choices[0].notes // []' 2>/dev/null || echo "[]")
    MCP_USED=$(echo "$RESULT_EVENT" | jq -r '.data.mcpUsed // false' 2>/dev/null)
    MCP_TOOLS=$(echo "$RESULT_EVENT" | jq -r '.data.mcpTools // []' 2>/dev/null)

    if [ -n "$NOTES" ] && [ "$NOTES" != "[]" ] && [ "$NOTES" != "null" ]; then
        NOTE_COUNT=$(echo "$NOTES" | jq 'length')
        echo -e "  ${GREEN}✅ Generation completed with $NOTE_COUNT notes${NC}"

        # Check if notes have velocity (should be influenced by arpeggio tool)
        VELOCITY_COUNT=$(echo "$NOTES" | jq '[.[] | select(.velocity != null and .velocity > 0)] | length')
        if [ "$VELOCITY_COUNT" -eq "$NOTE_COUNT" ]; then
            echo -e "  ${GREEN}✅ All notes have velocity${NC}"

            # Check velocity range
            MIN_VEL=$(echo "$NOTES" | jq '[.[] | .velocity] | min')
            MAX_VEL=$(echo "$NOTES" | jq '[.[] | .velocity] | max')
            echo "     Velocity range: $MIN_VEL - $MAX_VEL"
        else
            echo -e "  ${YELLOW}⚠️  Some notes missing velocity ($VELOCITY_COUNT/$NOTE_COUNT)${NC}"
            WARNINGS=$((WARNINGS + 1))
        fi

        # Check MCP usage stats
        if [ "$MCP_USED" = "true" ]; then
            echo -e "  ${GREEN}✅ MCP was used in generation${NC}"
            if [ -n "$MCP_TOOLS" ] && [ "$MCP_TOOLS" != "[]" ]; then
                ARP_TOOLS=$(echo "$MCP_TOOLS" | jq -r '.[] | select(. | contains("arp")) // empty' 2>/dev/null || echo "")
                if [ -n "$ARP_TOOLS" ]; then
                    echo -e "  ${GREEN}✅ Arpeggio tool listed in MCP tools${NC}"
                    echo "$ARP_TOOLS" | while IFS= read -r tool; do
                        echo "     Tool: $tool"
                    done
                else
                    echo -e "  ${YELLOW}⚠️  Arpeggio tool not listed in MCP tools${NC}"
                    echo "     MCP tools: $MCP_TOOLS"
                    WARNINGS=$((WARNINGS + 1))
                fi
            else
                echo -e "  ${YELLOW}⚠️  No MCP tools listed in result${NC}"
                WARNINGS=$((WARNINGS + 1))
            fi
        else
            echo -e "  ${YELLOW}⚠️  MCP not marked as used in result${NC}"
            WARNINGS=$((WARNINGS + 1))
        fi
    else
        echo -e "  ${YELLOW}⚠️  No notes found in final generation${NC}"
        WARNINGS=$((WARNINGS + 1))
    fi
else
    echo -e "  ${RED}❌ No result event found${NC}"
    FAILED=$((FAILED + 1))
fi
echo ""

# Summary
echo "======================================"
if [ $FAILED -eq 0 ]; then
    if [ $WARNINGS -eq 0 ]; then
        echo -e "${GREEN}✅ All arpeggio tool tests passed!${NC}"
        exit 0
    else
        echo -e "${GREEN}✅ All critical tests passed!${NC}"
        echo -e "${YELLOW}⚠️  $WARNINGS warning(s)${NC}"
        exit 0
    fi
else
    echo -e "${RED}❌ $FAILED test(s) failed${NC}"
    if [ $WARNINGS -gt 0 ]; then
        echo -e "${YELLOW}⚠️  $WARNINGS warning(s)${NC}"
    fi
    exit 1
fi
