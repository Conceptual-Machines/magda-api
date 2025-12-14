#!/bin/bash

# Compare streaming vs non-streaming MAGDA endpoints
# Tests both endpoints with the same question and compares timing
# Usage: ./test-magda-compare.sh [JWT_TOKEN]

set -e

API_URL="${API_URL:-https://api.musicalaideas.com}"

# Load environment variables from .envrc if available
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
if [ -f "$SCRIPT_DIR/.envrc" ]; then
    set -a
    while IFS= read -r line; do
        eval "$line" 2>/dev/null || true
    done < <(grep "^export " "$SCRIPT_DIR/.envrc" 2>/dev/null)
    set +a
fi

# Get JWT token - try from arg, env var, or login
JWT_TOKEN="${1:-${AIDEAS_JWT_TOKEN}}"

if [ -z "$JWT_TOKEN" ] && [ -n "$AIDEAS_EMAIL" ] && [ -n "$AIDEAS_PASSWORD" ]; then
    echo "🔐 Logging in to get JWT token..."
    AUTH_PAYLOAD=$(jq -n --arg email "$AIDEAS_EMAIL" --arg password "$AIDEAS_PASSWORD" '{email: $email, password: $password}')

    AUTH_RESPONSE=$(curl -s -X POST "${API_URL}/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "$AUTH_PAYLOAD")

    if echo "$AUTH_RESPONSE" | grep -q "access_token"; then
        JWT_TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.access_token' 2>/dev/null)
        if [ -n "$JWT_TOKEN" ] && [ "$JWT_TOKEN" != "null" ]; then
            echo "✅ Logged in successfully"
        fi
    fi
fi

if [ -z "$JWT_TOKEN" ] || [ "$JWT_TOKEN" = "null" ]; then
    echo "❌ Error: JWT token required"
    echo "Usage: $0 [JWT_TOKEN]"
    echo "Or set AIDEAS_JWT_TOKEN, or AIDEAS_EMAIL and AIDEAS_PASSWORD in .envrc"
    exit 1
fi

QUESTION="create a track named Drums with Serum instrument and add a 4 bar clip starting at bar 1"

echo "🧪 Comparing MAGDA Streaming vs Non-Streaming Endpoints"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📝 Question: ${QUESTION}"
echo ""

# ============================================================================
# TEST 1: NON-STREAMING ENDPOINT
# ============================================================================
echo "📊 TEST 1: Non-Streaming Endpoint (/api/v1/magda/chat)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [[ "$OSTYPE" == "darwin"* ]]; then
  NON_STREAM_START=$(python3 -c "import time; print(int(time.time() * 1000))")
else
  NON_STREAM_START=$(($(date +%s%N) / 1000000))
fi

NON_STREAM_RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/magda/chat" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -d "$(jq -n --arg question "$QUESTION" '{question: $question}')")

if [[ "$OSTYPE" == "darwin"* ]]; then
  NON_STREAM_END=$(python3 -c "import time; print(int(time.time() * 1000))")
else
  NON_STREAM_END=$(($(date +%s%N) / 1000000))
fi

NON_STREAM_DURATION=$((NON_STREAM_END - NON_STREAM_START))
NON_STREAM_ACTIONS_COUNT=$(echo "$NON_STREAM_RESPONSE" | jq -r '.actions | length' 2>/dev/null || echo "0")
NON_STREAM_ERROR=$(echo "$NON_STREAM_RESPONSE" | jq -r '.error // empty' 2>/dev/null)

echo "  Total duration: ${NON_STREAM_DURATION}ms"
echo "  Actions received: ${NON_STREAM_ACTIONS_COUNT}"
if [ "$NON_STREAM_ACTIONS_COUNT" -gt 0 ]; then
  echo "  Time to first action: ${NON_STREAM_DURATION}ms (wait for all)"
  echo "  ⏱️  All actions arrive together at the end"
elif [ -n "$NON_STREAM_ERROR" ]; then
  echo "  ❌ Error: ${NON_STREAM_ERROR}"
fi
echo ""

# ============================================================================
# TEST 2: STREAMING ENDPOINT (using timing script approach)
# ============================================================================
echo "📊 TEST 2: Streaming Endpoint (/api/v1/magda/chat/stream)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Use the same approach as test-magda-stream-timing.sh
if [[ "$OSTYPE" == "darwin"* ]]; then
  REQUEST_START=$(python3 -c "import time; print(int(time.time() * 1000))")
else
  REQUEST_START=$(($(date +%s%N) / 1000000))
fi

FIRST_ACTION_TIME=""
LAST_ACTION_TIME=""
ACTION_COUNT=0
PREVIOUS_ACTION_TIME=""

curl -s -X POST "${API_URL}/api/v1/magda/chat/stream" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -d "$(jq -n --arg question "$QUESTION" '{question: $question}')" \
  --no-buffer -N | while IFS= read -r line; do
    if [[ "$line" =~ ^data: ]]; then
      JSON=$(echo "$line" | sed 's/^data: //')

      if echo "$JSON" | jq -e '.type == "action"' > /dev/null 2>&1; then
        ACTION_COUNT=$((ACTION_COUNT + 1))
        if [[ "$OSTYPE" == "darwin"* ]]; then
          NOW=$(python3 -c "import time; print(int(time.time() * 1000))")
        else
          NOW=$(($(date +%s%N) / 1000000))
        fi

        if [ -z "$FIRST_ACTION_TIME" ]; then
          FIRST_ACTION_TIME=$NOW
        fi
        LAST_ACTION_TIME=$NOW

        TIME_TO_ACTION_MS=$((NOW - REQUEST_START))
        if [ -z "$PREVIOUS_ACTION_TIME" ]; then
          PREVIOUS_ACTION_TIME=$REQUEST_START
        fi
        TIME_SINCE_PREVIOUS_MS=$((NOW - PREVIOUS_ACTION_TIME))
        PREVIOUS_ACTION_TIME=$NOW

        ACTION_NAME=$(echo "$JSON" | jq -r '.action.action // empty' 2>/dev/null)
        echo "  ⏱️  Action #${ACTION_COUNT} (${ACTION_NAME}) received: ${TIME_TO_ACTION_MS}ms (${TIME_SINCE_PREVIOUS_MS}ms since previous)"
      elif echo "$JSON" | jq -e '.type == "done"' > /dev/null 2>&1; then
        if [[ "$OSTYPE" == "darwin"* ]]; then
          REQUEST_END=$(python3 -c "import time; print(int(time.time() * 1000))")
        else
          REQUEST_END=$(($(date +%s%N) / 1000000))
        fi
        TOTAL_DURATION_MS=$((REQUEST_END - REQUEST_START))

        echo ""
        echo "  📊 Timing Summary:"
        echo "    Total duration: ${TOTAL_DURATION_MS}ms"
        if [ -n "$FIRST_ACTION_TIME" ]; then
          TIME_TO_FIRST_MS=$((FIRST_ACTION_TIME - REQUEST_START))
          TIME_TO_LAST_MS=$((LAST_ACTION_TIME - REQUEST_START))
          echo "    Time to first action: ${TIME_TO_FIRST_MS}ms"
          echo "    Time to last action: ${TIME_TO_LAST_MS}ms"
          echo "    Actions received: ${ACTION_COUNT}"
        else
          echo "    No actions received"
        fi
      elif echo "$JSON" | jq -e '.type == "error"' > /dev/null 2>&1; then
        ERROR_MSG=$(echo "$JSON" | jq -r '.message // "Unknown error"' 2>/dev/null)
        echo "  ❌ Error: ${ERROR_MSG}"
      fi
    fi
  done

echo ""

# ============================================================================
# COMPARISON SUMMARY
# ============================================================================
echo "📊 COMPARISON SUMMARY"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "  Non-Streaming:"
echo "    • Total time: ${NON_STREAM_DURATION}ms"
echo "    • Actions: ${NON_STREAM_ACTIONS_COUNT}"
echo "    • First action: Arrives at ${NON_STREAM_DURATION}ms (after all complete)"
echo ""
echo "  Streaming:"
echo "    • See detailed timing above"
echo "    • First action: Arrives incrementally as generated"
echo ""
echo "  ⚡ Key Benefit:"
if [ "$NON_STREAM_ACTIONS_COUNT" -gt 0 ]; then
  echo "    Streaming allows actions to execute as soon as they're ready,"
  echo "    rather than waiting for all actions to complete."
  echo "    This provides a more responsive user experience."
else
  echo "    ⚠️  Non-streaming endpoint returned no actions"
  echo "    Response: $(echo "$NON_STREAM_RESPONSE" | jq -c '.' 2>/dev/null | head -c 200)"
fi

echo ""
echo "✅ Comparison complete"
