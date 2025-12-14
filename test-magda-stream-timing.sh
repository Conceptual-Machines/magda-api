#!/bin/bash

# Test script for MAGDA streaming endpoint with detailed timing
# Tests a command that generates 2 actions and shows intermediate timing
# Usage: ./test-magda-stream-timing.sh [JWT_TOKEN]

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

echo "🧪 Testing MAGDA streaming endpoint with timing"
echo "📡 URL: ${API_URL}/api/v1/magda/chat/stream"
echo "📝 Question: create a track named Drums with Serum instrument and add a 4 bar clip starting at bar 1"
echo ""

# Track timing (use milliseconds for cross-platform compatibility)
if [[ "$OSTYPE" == "darwin"* ]]; then
  # macOS - use Python for millisecond precision
  REQUEST_START=$(python3 -c "import time; print(int(time.time() * 1000))")
  GET_TIME_MS() { python3 -c "import time; print(int(time.time() * 1000))"; }
else
  # Linux - use date with nanoseconds, convert to ms
  REQUEST_START=$(($(date +%s%N) / 1000000))
  GET_TIME_MS() { echo $(($(date +%s%N) / 1000000)); }
fi
FIRST_ACTION_TIME=""
LAST_ACTION_TIME=""
ACTION_COUNT=0
declare -a ACTION_TIMES=()
declare -a ACTION_NAMES=()

# Test request - stream with no buffering
curl -X POST "${API_URL}/api/v1/magda/chat/stream" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -d '{
    "question": "create a track named Drums with Serum instrument and add a 4 bar clip starting at bar 1"
  }' \
  --no-buffer \
  -N \
  -s | while IFS= read -r line; do
    if [[ "$line" =~ ^data: ]]; then
      # Extract JSON from SSE line
      JSON=$(echo "$line" | sed 's/^data: //')

      # Check if it's an action event
      if echo "$JSON" | jq -e '.type == "action"' > /dev/null 2>&1; then
        ACTION_COUNT=$((ACTION_COUNT + 1))
        if [[ "$OSTYPE" == "darwin"* ]]; then
          NOW=$(python3 -c "import time; print(int(time.time() * 1000))")
        else
          NOW=$(($(date +%s%N) / 1000000))
        fi

        # Extract action name
        ACTION_NAME=$(echo "$JSON" | jq -r '.action.action // empty' 2>/dev/null)
        ACTION_NAMES+=("$ACTION_NAME")

        if [ -z "$FIRST_ACTION_TIME" ]; then
          FIRST_ACTION_TIME=$NOW
          TIME_TO_FIRST=$((NOW - REQUEST_START))
          echo "⏱️  Action #1 (${ACTION_NAME}) received: ${TIME_TO_FIRST}ms"
        else
          TIME_TO_ACTION=$((NOW - REQUEST_START))
          TIME_SINCE_PREV=$((NOW - LAST_ACTION_TIME))
          echo "⏱️  Action #${ACTION_COUNT} (${ACTION_NAME}) received: ${TIME_TO_ACTION}ms (${TIME_SINCE_PREV}ms since previous)"
        fi

        LAST_ACTION_TIME=$NOW
        ACTION_TIMES+=($NOW)

        # Show the action
        echo "$JSON" | jq -c '.action' 2>/dev/null || echo "$JSON"
      elif echo "$JSON" | jq -e '.type == "done"' > /dev/null 2>&1; then
        if [[ "$OSTYPE" == "darwin"* ]]; then
          REQUEST_END=$(python3 -c "import time; print(int(time.time() * 1000))")
        else
          REQUEST_END=$(($(date +%s%N) / 1000000))
        fi
        TOTAL_DURATION=$((REQUEST_END - REQUEST_START))

        if [ -n "$FIRST_ACTION_TIME" ]; then
          TIME_TO_FIRST_MS=$((FIRST_ACTION_TIME - REQUEST_START))
          TIME_TO_LAST_MS=$((LAST_ACTION_TIME - REQUEST_START))

          echo ""
          echo "📊 Timing Summary:"
          echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
          echo "  Total duration:        ${TOTAL_DURATION}ms"
          echo "  Time to first action:  ${TIME_TO_FIRST_MS}ms"
          echo "  Time to last action:   ${TIME_TO_LAST_MS}ms"
          echo "  Actions received:      ${ACTION_COUNT}"

          # Show per-action timing if we have multiple actions
          if [ ${#ACTION_TIMES[@]} -gt 1 ]; then
            echo ""
            echo "  Per-action timing:"
            for i in "${!ACTION_TIMES[@]}"; do
              ACTION_INDEX=$((i + 1))
              ACTION_TIME=${ACTION_TIMES[$i]}
              TIME_TO_ACTION=$((ACTION_TIME - REQUEST_START))
              if [ $i -gt 0 ]; then
                PREV_TIME=${ACTION_TIMES[$((i - 1))]}
                TIME_SINCE_PREV=$((ACTION_TIME - PREV_TIME))
                echo "    Action #${ACTION_INDEX} (${ACTION_NAMES[$i]}): ${TIME_TO_ACTION}ms (+${TIME_SINCE_PREV}ms)"
              else
                echo "    Action #${ACTION_INDEX} (${ACTION_NAMES[$i]}): ${TIME_TO_ACTION}ms"
              fi
            done
          fi
          echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        else
          echo ""
          echo "📊 Timing Summary:"
          echo "  Total duration: ${TOTAL_DURATION}ms"
          echo "  No actions received"
        fi
        echo ""
        echo "$JSON" | jq -c '.actions' 2>/dev/null || echo "$JSON"
      else
        echo "$line"
      fi
    else
      echo "$line"
    fi
  done

echo ""
echo "✅ Test complete"
