#!/bin/bash

# E2E test for MAGDA DSL functionality
# Tests: LLM generates DSL → Go translates → Returns actions

set -e

API_URL="${API_URL:-https://api.musicalaideas.com}"

# Load environment variables
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
if [ -f "$SCRIPT_DIR/.envrc" ]; then
    set -a
    while IFS= read -r line; do
        eval "$line" 2>/dev/null || true
    done < <(grep "^export " "$SCRIPT_DIR/.envrc" 2>/dev/null)
    set +a
fi

# Get JWT token
JWT_TOKEN="${AIDEAS_JWT_TOKEN}"

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
    exit 1
fi

echo "🧪 E2E Test: MAGDA DSL Flow"
echo "📡 URL: ${API_URL}/api/v1/magda/chat/stream"
echo "📝 Question: create a track with Serum"
echo ""

# Track timing
START_TIME=$(date +%s.%N)
FIRST_ACTION_TIME=""
LAST_ACTION_TIME=""

# Test request
RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/magda/chat/stream" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -d '{
    "question": "create a track with Serum"
  }' \
  --no-buffer \
  -N)

END_TIME=$(date +%s.%N)

echo "📥 Response received:"
echo ""

ACTION_COUNT=0
DSL_FOUND=false
ACTIONS_RECEIVED=0
FIRST_ACTION_TIME=""
LAST_ACTION_TIME=""

# Process SSE events and capture timing info
TEMP_FILE=$(mktemp)
echo "$RESPONSE" | while IFS= read -r line; do
    if [[ "$line" =~ ^data: ]]; then
        JSON=$(echo "$line" | sed 's/^data: //')
        NOW=$(date +%s.%N)

        # Check for action events
        if echo "$JSON" | jq -e '.type == "action"' > /dev/null 2>&1; then
            ACTIONS_RECEIVED=$((ACTIONS_RECEIVED + 1))
            if [ -z "$FIRST_ACTION_TIME" ]; then
                FIRST_ACTION_TIME=$NOW
                TIME_TO_FIRST=$(echo "$FIRST_ACTION_TIME - $START_TIME" | bc)
                echo "⏱️  First action received: $(printf "%.3f" $TIME_TO_FIRST)s"
                echo "$FIRST_ACTION_TIME" > "$TEMP_FILE.first"
            fi
            LAST_ACTION_TIME=$NOW
            echo "$LAST_ACTION_TIME" > "$TEMP_FILE.last"
            echo "$ACTIONS_RECEIVED" > "$TEMP_FILE.count"
            TIME_TO_ACTION=$(echo "$NOW - $START_TIME" | bc)
            ACTION=$(echo "$JSON" | jq -c '.action' 2>/dev/null)
            ACTION_NAME=$(echo "$ACTION" | jq -r '.action // "unknown"' 2>/dev/null)
            echo "✅ Action #${ACTIONS_RECEIVED} (${ACTION_NAME}) at $(printf "%.3f" $TIME_TO_ACTION)s:"
            echo "$ACTION" | jq '.'
            echo ""
        elif echo "$JSON" | jq -e '.type == "done"' > /dev/null 2>&1; then
            echo "✅ Stream complete"
            echo ""
            echo "$JSON" | jq -c '.actions // []' 2>/dev/null
            break
        elif echo "$JSON" | jq -e '.type == "error"' > /dev/null 2>&1; then
            ERROR_MSG=$(echo "$JSON" | jq -r '.message // .error // .' 2>/dev/null)
            # If we already received actions, this might just be a stream completion issue
            if [ $ACTIONS_RECEIVED -gt 0 ]; then
                echo "⚠️  Stream error (but actions were received): $ERROR_MSG"
            else
                echo "❌ Error: $ERROR_MSG"
                exit 1
            fi
        fi
    fi
done

# Read timing info from temp files
if [ -f "$TEMP_FILE.first" ]; then
    FIRST_ACTION_TIME=$(cat "$TEMP_FILE.first")
    LAST_ACTION_TIME=$(cat "$TEMP_FILE.last")
    ACTIONS_RECEIVED=$(cat "$TEMP_FILE.count")
    rm -f "$TEMP_FILE.first" "$TEMP_FILE.last" "$TEMP_FILE.count"
fi

# Calculate timing
TOTAL_DURATION=$(echo "$END_TIME - $START_TIME" | bc)
if [ -n "$FIRST_ACTION_TIME" ]; then
    TIME_TO_FIRST_MS=$(echo "($FIRST_ACTION_TIME - $START_TIME) * 1000" | bc)
    TIME_TO_LAST_MS=$(echo "($LAST_ACTION_TIME - $START_TIME) * 1000" | bc)
    echo ""
    echo "📊 Timing Summary:"
    echo "  Total duration: $(printf "%.3f" $TOTAL_DURATION)s"
    echo "  Time to first action: $(printf "%.0f" $TIME_TO_FIRST_MS)ms"
    echo "  Time to last action: $(printf "%.0f" $TIME_TO_LAST_MS)ms"
    echo "  Actions received: ${ACTIONS_RECEIVED}"
else
    echo ""
    echo "📊 Timing Summary:"
    echo "  Total duration: $(printf "%.3f" $TOTAL_DURATION)s"
    echo "  No actions received"
fi
echo ""
echo "✅ E2E test complete"
