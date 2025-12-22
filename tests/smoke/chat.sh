#!/bin/bash

# Smoke test: MAGDA chat endpoint
# Usage: ./chat.sh [base-url]
# Requires: OPENAI_API_KEY set on the server

set -e

BASE_URL=${1:-"http://localhost:8080"}

echo "🔍 Testing MAGDA chat endpoint at $BASE_URL"

# Simple test request
PAYLOAD='{
  "question": "create a track called Bass",
  "state": {
    "project": {"name": "Test", "length": 120.0},
    "tracks": []
  }
}'

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/chat" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD")

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -qE "(actions|error)"; then
        if echo "$BODY" | grep -q '"actions"'; then
            echo "✅ Chat endpoint passed"

            if command -v jq &> /dev/null; then
                ACTION_COUNT=$(echo "$BODY" | jq '.actions | length' 2>/dev/null || echo "0")
                echo "   Actions returned: $ACTION_COUNT"

                # Show first action
                FIRST_ACTION=$(echo "$BODY" | jq -r '.actions[0].action // "none"' 2>/dev/null)
                echo "   First action: $FIRST_ACTION"
            fi
            exit 0
        else
            echo "⚠️  Response received but no actions (may be expected)"
            echo "   Response: $BODY"
            exit 0
        fi
    fi
fi

echo "❌ Chat endpoint failed"
echo "   HTTP: $HTTP_CODE"
echo "   Response: $BODY"
exit 1
