#!/bin/bash

# Smoke test: JSFX generation endpoint
# Usage: ./jsfx.sh [base-url]
# Requires: OPENAI_API_KEY set on the server

set -e

BASE_URL=${1:-"http://localhost:8080"}

echo "🔍 Testing JSFX generation endpoint at $BASE_URL"

# Simple test request
PAYLOAD='{
  "message": "Create a simple gain plugin with a slider",
  "code": "",
  "filename": "test.jsfx"
}'

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/jsfx/generate" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD")

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q "jsfx_code"; then
        echo "✅ JSFX generation passed"

        if command -v jq &> /dev/null; then
            CODE_LENGTH=$(echo "$BODY" | jq -r '.jsfx_code | length' 2>/dev/null || echo "0")
            MESSAGE=$(echo "$BODY" | jq -r '.message // "none"' 2>/dev/null)
            echo "   Code length: $CODE_LENGTH chars"
            echo "   Message: $MESSAGE"
        fi
        exit 0
    fi
fi

echo "❌ JSFX generation failed"
echo "   HTTP: $HTTP_CODE"
echo "   Response: $BODY"
exit 1
