#!/bin/bash

# Streaming Endpoint Smoke Tests
# Tests the /api/generate endpoint with stream=true parameter

set -e

BASE_URL=${1:-"https://api.musicalaideas.com"}
VERBOSE=${VERBOSE:-false}

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧪 Streaming Endpoint Smoke Tests${NC}"
echo "URL: $BASE_URL/api/generate"
echo ""

# Test 1: Basic streaming test
echo -e "${YELLOW}Test 1: Basic Streaming${NC}"
RESPONSE=$(curl -s -X POST "$BASE_URL/api/generate" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5-mini",
    "stream": true,
    "input_array": [{
      "role": "user",
      "content": "{\"user_prompt\": \"Generate a simple C major chord progression\"}"
    }]
  }')

# Check for required event types
if echo "$RESPONSE" | grep -q '"type":"start"'; then
  echo -e "  ${GREEN}✅ Start event received${NC}"
else
  echo -e "  ${RED}❌ No start event${NC}"
  exit 1
fi

if echo "$RESPONSE" | grep -q '"type":"mcp_enabled"'; then
  echo -e "  ${GREEN}✅ MCP enabled event received${NC}"
else
  echo -e "  ${YELLOW}⚠️  No MCP enabled event (might be disabled)${NC}"
fi

if echo "$RESPONSE" | grep -q '"type":"heartbeat"'; then
  echo -e "  ${GREEN}✅ Heartbeat events received${NC}"
else
  echo -e "  ${RED}❌ No heartbeat events${NC}"
  exit 1
fi

if echo "$RESPONSE" | grep -q '"type":"complete"'; then
  echo -e "  ${GREEN}✅ Complete event received${NC}"
else
  echo -e "  ${RED}❌ No complete event${NC}"
  exit 1
fi

if echo "$RESPONSE" | grep -q '"type":"done"'; then
  echo -e "  ${GREEN}✅ Done event received${NC}"
else
  echo -e "  ${RED}❌ No done event${NC}"
  exit 1
fi

# Test 2: MCP tool usage
echo -e "\n${YELLOW}Test 2: MCP Tool Usage (Lydian mode)${NC}"
RESPONSE=$(curl -s -X POST "$BASE_URL/api/generate" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5-mini",
    "stream": true,
    "input_array": [{
      "role": "user",
      "content": "{\"user_prompt\": \"Generate a progression using Lydian mode\"}"
    }]
  }')

if echo "$RESPONSE" | grep -q '"type":"mcp_used"'; then
  echo -e "  ${GREEN}✅ MCP tools were called${NC}"
  # Extract MCP tools from the response
  MCP_TOOLS=$(echo "$RESPONSE" | grep '"type":"mcp_used"' | head -1 | sed 's/data: //' | jq -r '.data.tools // []' 2>/dev/null || echo "[]")
  echo "     Tools: $MCP_TOOLS"
else
  echo -e "  ${YELLOW}⚠️  MCP tools not used (unexpected for Lydian query)${NC}"
fi

# Test 3: Check for errors
echo -e "\n${YELLOW}Test 3: Error Handling${NC}"
if echo "$RESPONSE" | grep -q '"type":"error"'; then
  echo -e "  ${RED}❌ Error event found in response${NC}"
  ERROR_MSG=$(echo "$RESPONSE" | grep '"type":"error"' | head -1 | sed 's/data: //' | jq -r '.message' 2>/dev/null || echo "Unknown")
  echo "     Error: $ERROR_MSG"
  exit 1
else
  echo -e "  ${GREEN}✅ No error events${NC}"
fi

# Test 4: Non-streaming mode still works
echo -e "\n${YELLOW}Test 4: Non-Streaming Mode (stream=false)${NC}"
RESPONSE=$(curl -s -X POST "$BASE_URL/api/generate" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5-mini",
    "stream": false,
    "input_array": [{
      "role": "user",
      "content": "{\"user_prompt\": \"Generate a simple Am chord progression\"}"
    }]
  }')

if echo "$RESPONSE" | jq -e '.output_parsed.choices' > /dev/null 2>&1; then
  NOTE_COUNT=$(echo "$RESPONSE" | jq -r '.output_parsed.choices[0].notes | length')
  MCP_USED=$(echo "$RESPONSE" | jq -r '.mcpUsed // false')
  echo -e "  ${GREEN}✅ Non-streaming mode works${NC}"
  echo "     Notes: $NOTE_COUNT, MCP Used: $MCP_USED"
else
  echo -e "  ${RED}❌ Non-streaming mode failed${NC}"
  exit 1
fi

# Test 5: Performance check
echo -e "\n${YELLOW}Test 5: Streaming Performance${NC}"
START_TIME=$(date +%s)
RESPONSE=$(curl -s -X POST "$BASE_URL/api/generate" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5-mini",
    "stream": true,
    "input_array": [{
      "role": "user",
      "content": "{\"user_prompt\": \"Create a jazz progression\"}"
    }]
  }')
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

EVENT_COUNT=$(echo "$RESPONSE" | grep -c "^data:" || echo 0)
echo -e "  ${GREEN}✅ Streaming completed in ${DURATION}s${NC}"
echo "     Events received: $EVENT_COUNT"

if [ $DURATION -lt 120 ]; then
  echo -e "  ${GREEN}✅ Within timeout limit (120s)${NC}"
else
  echo -e "  ${YELLOW}⚠️  Took ${DURATION}s (close to timeout)${NC}"
fi

# Final summary
echo -e "\n${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ All streaming smoke tests passed!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
