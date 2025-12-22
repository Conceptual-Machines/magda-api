#!/bin/bash

# Smoke test: Health endpoint
# Usage: ./health.sh [base-url]

set -e

BASE_URL=${1:-"http://localhost:8080"}

echo "🔍 Testing health endpoint at $BASE_URL"

RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/health")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ] && echo "$BODY" | grep -q "healthy"; then
    echo "✅ Health check passed"
    echo "   Response: $BODY"
    exit 0
else
    echo "❌ Health check failed"
    echo "   HTTP: $HTTP_CODE"
    echo "   Response: $BODY"
    exit 1
fi
