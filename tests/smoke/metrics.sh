#!/bin/bash

# Smoke test: Metrics endpoint
# Usage: ./metrics.sh [base-url]

set -e

BASE_URL=${1:-"http://localhost:8080"}

echo "🔍 Testing metrics endpoint at $BASE_URL"

RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/metrics")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ] && echo "$BODY" | grep -q "healthy"; then
    echo "✅ Metrics check passed"

    if command -v jq &> /dev/null; then
        VERSION=$(echo "$BODY" | jq -r '.version // "unknown"')
        UPTIME=$(echo "$BODY" | jq -r '.uptime // "unknown"')
        echo "   Version: $VERSION"
        echo "   Uptime: $UPTIME"
    fi
    exit 0
else
    echo "❌ Metrics check failed"
    echo "   HTTP: $HTTP_CODE"
    echo "   Response: $BODY"
    exit 1
fi
