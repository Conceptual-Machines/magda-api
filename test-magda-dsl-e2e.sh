#!/bin/bash

# E2E test for MAGDA DSL functionality
# Tests: LLM generates DSL → Go translates → Returns actions

set -e

API_URL="${API_URL:-http://localhost:8080}"

# Always load from .env file first
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
if [ -f "$SCRIPT_DIR/.env" ]; then
    set -a
    while IFS= read -r line || [ -n "$line" ]; do
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ -z "${line// }" ]] && continue
        export "$line" 2>/dev/null || true
    done < "$SCRIPT_DIR/.env"
    set +a
fi

# Get JWT token
JWT_TOKEN="${MAGDA_JWT_TOKEN:-${AIDEAS_JWT_TOKEN:-}}"

if [ -z "$JWT_TOKEN" ] && [ -n "${MAGDA_EMAIL:-${AIDEAS_EMAIL:-}}" ] && [ -n "${MAGDA_PASSWORD:-${AIDEAS_PASSWORD:-}}" ]; then
    EMAIL="${MAGDA_EMAIL:-${AIDEAS_EMAIL:-}}"
    PASSWORD="${MAGDA_PASSWORD:-${AIDEAS_PASSWORD:-}}"
    echo "🔐 Logging in to get JWT token..."
    AUTH_PAYLOAD=$(jq -n --arg email "$EMAIL" --arg password "$PASSWORD" '{email: $email, password: $password}')

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
echo "📡 URL: ${API_URL}/api/v1/magda/chat"
echo "📝 Question: create a track with Serum"
echo ""

# Test request (non-streaming)
RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/magda/chat" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -d '{
    "question": "create a track with Serum"
  }')

echo "📥 Response received:"
echo ""

# Parse non-streaming JSON response
if echo "$RESPONSE" | jq -e '.actions' > /dev/null 2>&1; then
    ACTION_COUNT=$(echo "$RESPONSE" | jq '.actions | length')
    echo "✅ Received $ACTION_COUNT actions"
    echo ""

    if [ "$ACTION_COUNT" -gt 0 ]; then
        echo "Actions:"
        echo "$RESPONSE" | jq -c '.actions[]' | head -5
        echo ""
        exit 0
    else
        echo "❌ No actions in response"
        if echo "$RESPONSE" | jq -e '.error' > /dev/null 2>&1; then
            ERROR=$(echo "$RESPONSE" | jq -r '.error')
            echo "Error: $ERROR"
        fi
        exit 1
    fi
else
    echo "❌ Invalid response format"
    if echo "$RESPONSE" | jq -e '.error' > /dev/null 2>&1; then
        ERROR=$(echo "$RESPONSE" | jq -r '.error')
        echo "Error: $ERROR"
    else
        echo "Response: $(echo "$RESPONSE" | head -200)"
    fi
    exit 1
fi
