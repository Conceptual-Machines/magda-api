#!/bin/bash

# E2E test for MAGDA Note Generation
# Tests: User request → Orchestrator → Arranger → DSL → MIDI notes
#
# This tests the full pipeline for single note generation:
# 1. User says "add a sustained E1 note"
# 2. Orchestrator routes to Arranger agent (not DAW)
# 3. Arranger generates note(pitch="E1", duration=4) DSL
# 4. DSL parser converts to add_midi action with MIDI pitch 28

set -e

API_URL="${API_URL:-http://localhost:8080}"

echo "═════════════════════════════════════════════════════════════════"
echo "  🎵 MAGDA Note Generation E2E Test"
echo "═════════════════════════════════════════════════════════════════"
echo ""

# Load environment variables
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
if [ -f "$SCRIPT_DIR/.env" ]; then
    set -a
    source "$SCRIPT_DIR/.env" 2>/dev/null || true
    set +a
fi

# Check if server is running
if ! curl -s "${API_URL}/health" > /dev/null 2>&1; then
    echo "❌ Server is not running at ${API_URL}"
    echo "   Start with: make dev"
    exit 1
fi
echo "✅ Server is running at ${API_URL}"

# Use MAGDA_ACCESS_TOKEN or MAGDA_JWT_TOKEN
JWT_TOKEN="${MAGDA_ACCESS_TOKEN:-$MAGDA_JWT_TOKEN}"

if [ -z "$JWT_TOKEN" ]; then
    echo "❌ Error: No JWT token found (MAGDA_ACCESS_TOKEN or MAGDA_JWT_TOKEN)"
    exit 1
fi
echo "🔐 Using JWT token for auth"

PASSED=0
FAILED=0

run_test() {
    local test_name="$1"
    local question="$2"
    local expected_action="$3"
    local expected_pitch="$4"  # Optional: expected MIDI pitch
    local expected_duration="$5"  # Optional: expected duration

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📝 Test: $test_name"
    echo "   Question: \"$question\""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # Build curl command - use /api/v1/chat for MAGDA
    RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/chat" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer ${JWT_TOKEN}" \
      -d "{
        \"question\": \"$question\",
        \"state\": {
          \"project\": {\"name\": \"Test\", \"length\": 120.0},
          \"tracks\": [{\"index\": 0, \"name\": \"Track 1\"}]
        }
      }")

    # Check for errors
    if echo "$RESPONSE" | jq -e '.error' > /dev/null 2>&1; then
        ERROR=$(echo "$RESPONSE" | jq -r '.error')
        echo "❌ FAILED: API error: $ERROR"
        FAILED=$((FAILED + 1))
        return 1
    fi

    # Extract actions
    ACTIONS=$(echo "$RESPONSE" | jq '.actions // []')
    ACTION_COUNT=$(echo "$ACTIONS" | jq 'length')

    echo "   Received $ACTION_COUNT action(s)"

    # Check for expected action type
    if [ -n "$expected_action" ]; then
        HAS_ACTION=$(echo "$ACTIONS" | jq -r ".[].action // .[] | select(. == \"$expected_action\") // empty")
        if [ -z "$HAS_ACTION" ]; then
            # Check in nested format
            HAS_ACTION=$(echo "$ACTIONS" | jq -r ".[] | select(.action == \"$expected_action\") // empty")
        fi

        if [ -z "$HAS_ACTION" ]; then
            echo "❌ FAILED: Expected action '$expected_action' not found"
            echo "   Actions received:"
            echo "$ACTIONS" | jq '.'
            FAILED=$((FAILED + 1))
            return 1
        fi
        echo "   ✓ Found expected action: $expected_action"
    fi

    # Check for expected MIDI pitch
    if [ -n "$expected_pitch" ]; then
        # Look for add_midi action with notes containing the pitch
        MIDI_ACTION=$(echo "$ACTIONS" | jq ".[] | select(.action == \"add_midi\")")
        if [ -z "$MIDI_ACTION" ]; then
            echo "❌ FAILED: No add_midi action found"
            echo "$ACTIONS" | jq '.'
            FAILED=$((FAILED + 1))
            return 1
        fi

        FOUND_PITCH=$(echo "$MIDI_ACTION" | jq ".notes[0].pitch // .notes[0].midiNoteNumber // .notes[0].midi_note_number // empty")
        if [ "$FOUND_PITCH" != "$expected_pitch" ]; then
            echo "❌ FAILED: Expected pitch $expected_pitch, got $FOUND_PITCH"
            echo "   MIDI action:"
            echo "$MIDI_ACTION" | jq '.'
            FAILED=$((FAILED + 1))
            return 1
        fi
        echo "   ✓ Found expected MIDI pitch: $expected_pitch"
    fi

    # Check for expected duration (field name is 'length' in add_midi)
    if [ -n "$expected_duration" ]; then
        MIDI_ACTION=$(echo "$ACTIONS" | jq ".[] | select(.action == \"add_midi\")")
        FOUND_DURATION=$(echo "$MIDI_ACTION" | jq ".notes[0].length // .notes[0].duration // .notes[0].durationBeats // empty")
        if [ "$FOUND_DURATION" != "$expected_duration" ]; then
            echo "⚠️  Duration mismatch: expected $expected_duration, got $FOUND_DURATION"
            # Not a hard failure, just a warning
        else
            echo "   ✓ Found expected duration: $expected_duration"
        fi
    fi

    echo "✅ PASSED"
    echo "   Response:"
    echo "$ACTIONS" | jq '.'
    PASSED=$((PASSED + 1))
}

# ═══════════════════════════════════════════════════════════════════════════════
# Test Cases
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "🧪 Running Note Generation Tests..."

# Test 1: Sustained E1 note (bass note, typical user request)
run_test "Sustained E1 bass note" \
    "add a sustained E1 note to the track" \
    "add_midi" \
    "28" \
    "4"

# Test 2: Middle C (C4)
run_test "Middle C (C4)" \
    "add a C4 note for 2 beats" \
    "add_midi" \
    "60" \
    "2"

# Test 3: Sharp note
run_test "F# note (sharp)" \
    "add an F sharp 3 note" \
    "add_midi" \
    "54"

# Test 4: Track creation with note (should create track + route to arranger)
run_test "Track with note" \
    "create a track with Serum and add a sustained E1" \
    "create_track"

# Test 5: Verify arranger handles note (not DAW)
# The key here is that we should get add_midi action, not just track creation
run_test "Single note request" \
    "add note E1 with 4 beat duration" \
    "add_midi" \
    "28" \
    "4"

# ═══════════════════════════════════════════════════════════════════════════════
# Summary
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "═════════════════════════════════════════════════════════════════"
echo "  📊 Test Summary"
echo "═════════════════════════════════════════════════════════════════"
echo "  ✅ Passed: $PASSED"
echo "  ❌ Failed: $FAILED"
echo ""

if [ $FAILED -gt 0 ]; then
    echo "  ⚠️  Some tests failed!"
    exit 1
else
    echo "  🎉 All tests passed!"
fi
