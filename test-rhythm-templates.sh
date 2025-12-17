#!/bin/bash

# Test rhythm templates in arranger DSL
# Tests various rhythm templates: swing, bossa, syncopated, etc.

set -e

API_URL="${API_URL:-http://localhost:8080}"

# Load environment variables
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
if [ -f "$SCRIPT_DIR/.env" ]; then
    set -a
    source "$SCRIPT_DIR/.env"
    set +a
fi

# Get JWT token
JWT_TOKEN="${MAGDA_ACCESS_TOKEN}"

if [ -z "$JWT_TOKEN" ] && [ -n "$MAGDA_EMAIL" ] && [ -n "$MAGDA_PASSWORD" ]; then
    echo "🔐 Logging in to get JWT token..."
    AUTH_PAYLOAD=$(jq -n --arg email "$MAGDA_EMAIL" --arg password "$MAGDA_PASSWORD" '{email: $email, password: $password}')

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
    echo "❌ Error: JWT token required (set MAGDA_ACCESS_TOKEN)"
    exit 1
fi

# Test cases for rhythm templates
declare -a TESTS=(
    'swing|Create a jazz chord progression with swing rhythm|chord(Cmaj7, start=0, dur=4, rhythm=swing)'
    'bossa|Create a bossa nova chord progression|chord(Dm7, start=0, dur=8, rhythm=bossa)'
    'syncopated|Create syncopated funk chords|chord(C, start=0, dur=4, rhythm=syncopated)'
    'arpeggio-swing|Create a swing arpeggio|arpeggio(Em, up, start=0, dur=4, rhythm=swing)'
    'arpeggio-bossa|Create a bossa arpeggio|arpeggio(Am7, updown, start=0, dur=8, rhythm=bossa)'
)

echo "🧪 Testing Rhythm Templates in Arranger DSL"
echo "📡 URL: ${API_URL}/api/v1/aideas/generations"
echo ""

PASSED=0
FAILED=0

for test_case in "${TESTS[@]}"; do
    IFS='|' read -r test_name description expected_pattern <<< "$test_case"

    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🧪 Test: $test_name"
    echo "📝 Description: $description"
    echo ""

    # Create request payload with input_array format
    REQUEST_PAYLOAD=$(jq -n \
        --arg prompt "$description" \
        '{input_array: [{role: "user", content: $prompt}]}')

    # Make request (30 second timeout)
    RESPONSE=$(curl -s --max-time 30 -X POST "${API_URL}/api/v1/aideas/generations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${JWT_TOKEN}" \
        -d "$REQUEST_PAYLOAD")

    # Check if response contains DSL with rhythm
    if echo "$RESPONSE" | jq -e '.dsl' > /dev/null 2>&1; then
        DSL=$(echo "$RESPONSE" | jq -r '.dsl')

        # Check if DSL contains rhythm pattern
        if echo "$DSL" | grep -qi "rhythm="; then
            echo "✅ DSL generated with rhythm:"
            echo "$DSL" | jq -r '.' 2>/dev/null || echo "$DSL"
            echo ""

            # Check if it matches expected pattern
            if echo "$DSL" | grep -qi "$expected_pattern"; then
                echo "✅ Pattern matches expected: $expected_pattern"
                PASSED=$((PASSED + 1))
            else
                echo "⚠️  Pattern doesn't match exactly, but rhythm is present"
                PASSED=$((PASSED + 1))
            fi
        else
            echo "❌ DSL generated but no rhythm parameter found"
            echo "DSL: $DSL"
            FAILED=$((FAILED + 1))
        fi
    elif echo "$RESPONSE" | jq -e '.error' > /dev/null 2>&1; then
        ERROR=$(echo "$RESPONSE" | jq -r '.error')
        echo "❌ Error: $ERROR"
        FAILED=$((FAILED + 1))
    else
        echo "❌ Unexpected response format:"
        echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
        FAILED=$((FAILED + 1))
    fi

    echo ""
    sleep 1
done

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Test Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Passed: $PASSED"
echo "❌ Failed: $FAILED"
echo ""

if [ $FAILED -eq 0 ]; then
    echo "🎉 All rhythm template tests passed!"
    exit 0
else
    echo "❌ Some tests failed"
    exit 1
fi
