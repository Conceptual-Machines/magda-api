#!/bin/bash

# Test script for MAGDA streaming endpoint
# Usage: ./test-magda-stream.sh [JWT_TOKEN]

set -e

API_URL="${API_URL:-http://localhost:8080}"

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

echo "🧪 Testing MAGDA streaming endpoint..."
echo "📡 URL: ${API_URL}/api/v1/magda/chat/stream"
echo "📝 Question: create a track named Test Track"
echo ""

# Track timing
REQUEST_START=$(date +%s%N)
FIRST_ACTION_TIME=""
LAST_ACTION_TIME=""
ACTION_COUNT=0

# Test request - stream with no buffering, process line by line
curl -X POST "${API_URL}/api/v1/magda/chat/stream" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -d '{
    "question": "create a track named Test Track"
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
        NOW=$(date +%s%N)

        if [ -z "$FIRST_ACTION_TIME" ]; then
          FIRST_ACTION_TIME=$NOW
          TIME_TO_FIRST=$(( (NOW - REQUEST_START) / 1000000 ))
          echo "⏱️  First action received: ${TIME_TO_FIRST}ms"
        fi

        LAST_ACTION_TIME=$NOW
        TIME_TO_ACTION=$(( (NOW - REQUEST_START) / 1000000 ))
        ACTION_NAME=$(echo "$JSON" | jq -r '.action.action // empty' 2>/dev/null)
        echo "  Action #${ACTION_COUNT} (${ACTION_NAME}): ${TIME_TO_ACTION}ms"

        # Show the action
        echo "$JSON" | jq -c '.action' 2>/dev/null || echo "$JSON"
      elif echo "$JSON" | jq -e '.type == "done"' > /dev/null 2>&1; then
        REQUEST_END=$(date +%s%N)
        TOTAL_DURATION=$(( (REQUEST_END - REQUEST_START) / 1000000 ))

        if [ -n "$FIRST_ACTION_TIME" ]; then
          TIME_TO_FIRST_MS=$(( (FIRST_ACTION_TIME - REQUEST_START) / 1000000 ))
          TIME_TO_LAST_MS=$(( (LAST_ACTION_TIME - REQUEST_START) / 1000000 ))

          echo ""
          echo "📊 Timing Summary:"
          echo "  Total duration: ${TOTAL_DURATION}ms"
          echo "  Time to first action: ${TIME_TO_FIRST_MS}ms"
          echo "  Time to last action: ${TIME_TO_LAST_MS}ms"
          echo "  Actions received: ${ACTION_COUNT}"
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
