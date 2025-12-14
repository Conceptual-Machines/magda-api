#!/bin/bash

# Test Generation API directly
# Usage: ./test-generation.sh [base-url]

BASE_URL=${1:-"https://api.musicalaideas.com"}

echo "🎵 Testing AIDEAS API Generation Endpoint"
echo "URL: $BASE_URL/api/v1/aideas/generations"
echo ""

# Get auth token if credentials are available
TOKEN=""
if [ -n "$AIDEAS_EMAIL" ] && [ -n "$AIDEAS_PASSWORD" ]; then
    AUTH_PAYLOAD=$(jq -n --arg email "$AIDEAS_EMAIL" --arg password "$AIDEAS_PASSWORD" '{email: $email, password: $password}')
    AUTH_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/register/beta" \
        -H "Content-Type: application/json" \
        -d "$AUTH_PAYLOAD")
    if echo "$AUTH_RESPONSE" | grep -q "access_token"; then
        TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.access_token' 2>/dev/null)
    else
        AUTH_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
            -H "Content-Type: application/json" \
            -d "$AUTH_PAYLOAD")
        if echo "$AUTH_RESPONSE" | grep -q "access_token"; then
            TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.access_token' 2>/dev/null)
        fi
    fi
fi

if [ -z "$TOKEN" ]; then
    echo "⚠️  Warning: No auth token. Request may fail."
    echo ""
fi

echo "Sending request..."
HEADERS=("-H" "Content-Type: application/json")
if [ -n "$TOKEN" ]; then
    HEADERS+=("-H" "Authorization: Bearer $TOKEN")
fi

RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/aideas/generations" \
    "${HEADERS[@]}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5-mini",
    "input_array": [
      {
        "role": "user",
        "content": "{\"user_prompt\": \"Generate a simple C major chord progression with 4 chords\"}"
      }
    ]
  }')

echo "Response:"
echo "$RESPONSE" | jq .

echo ""
echo "Summary:"
# Check for both old format (genResults) and new format (output_parsed.choices)
if echo "$RESPONSE" | jq -e '.genResults' > /dev/null 2>&1; then
  RESULT_COUNT=$(echo "$RESPONSE" | jq '.genResults | length')
  echo "✅ Results returned: $RESULT_COUNT"

  for i in $(seq 0 $((RESULT_COUNT - 1))); do
    NOTE_COUNT=$(echo "$RESPONSE" | jq -r ".genResults[$i].notes | length")
    DESCRIPTION=$(echo "$RESPONSE" | jq -r ".genResults[$i].description")
    echo "  Result $((i+1)): $NOTE_COUNT notes - $DESCRIPTION"

    if [ "$NOTE_COUNT" -gt 0 ]; then
      echo "    First note:"
      echo "$RESPONSE" | jq ".genResults[$i].notes[0]"
    fi
  done
elif echo "$RESPONSE" | jq -e '.output_parsed.choices' > /dev/null 2>&1; then
  RESULT_COUNT=$(echo "$RESPONSE" | jq '.output_parsed.choices | length')
  echo "✅ Results returned: $RESULT_COUNT"

  for i in $(seq 0 $((RESULT_COUNT - 1))); do
    NOTE_COUNT=$(echo "$RESPONSE" | jq -r ".output_parsed.choices[$i].notes | length")
    DESCRIPTION=$(echo "$RESPONSE" | jq -r ".output_parsed.choices[$i].description")
    echo "  Result $((i+1)): $NOTE_COUNT notes - $DESCRIPTION"

    if [ "$NOTE_COUNT" -gt 0 ]; then
      echo "    First note:"
      echo "$RESPONSE" | jq ".output_parsed.choices[$i].notes[0]"
    fi
  done
else
  echo "❌ No results in response"
  if echo "$RESPONSE" | jq -e '.error' > /dev/null 2>&1; then
    ERROR=$(echo "$RESPONSE" | jq -r '.error')
    echo "Error: $ERROR"
  fi
fi
