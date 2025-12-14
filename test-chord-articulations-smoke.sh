#!/bin/bash

# Chord Articulations Smoke Test
# Tests that the API correctly replays chord progressions with rhythm variations
# Usage: ./test-chord-articulations-smoke.sh [base-url]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

BASE_URL=${1:-"https://api.musicalaideas.com"}
FAILED=0
WARNINGS=0

# Load environment variables for authentication
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$SCRIPT_DIR"

# Load environment variables from .envrc
if [ -f "$PROJECT_ROOT/.envrc" ]; then
    set -a
    source "$PROJECT_ROOT/.envrc" >/dev/null 2>&1 || true
    set +a
fi

# Debug: Check if credentials were loaded
if [ -z "$AIDEAS_EMAIL" ] || [ -z "$AIDEAS_PASSWORD" ]; then
    echo -e "${YELLOW}⚠️  Debug: Credentials not loaded from .envrc${NC}"
    if grep -q "AIDEAS_EMAIL" "$PROJECT_ROOT/.envrc" 2>/dev/null; then
        eval "$(grep "^export AIDEAS_" "$PROJECT_ROOT/.envrc" 2>/dev/null)" || true
    fi
fi

# Get auth token
TOKEN=""
if [ -n "$AIDEAS_EMAIL" ] && [ -n "$AIDEAS_PASSWORD" ]; then
    echo -e "${BLUE}🔐 Authenticating...${NC}"
    AUTH_PAYLOAD=$(jq -n --arg email "$AIDEAS_EMAIL" --arg password "$AIDEAS_PASSWORD" '{email: $email, password: $password}')

    AUTH_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/register/beta" \
        -H "Content-Type: application/json" \
        -d "$AUTH_PAYLOAD")

    if echo "$AUTH_RESPONSE" | grep -q "access_token"; then
        TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.access_token' 2>/dev/null)
        echo -e "${GREEN}✅ Authenticated${NC}"
    else
        AUTH_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
            -H "Content-Type: application/json" \
            -d "$AUTH_PAYLOAD")
        if echo "$AUTH_RESPONSE" | grep -q "access_token"; then
            TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.access_token' 2>/dev/null)
            if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
                echo -e "${GREEN}✅ Logged in and authenticated${NC}"
            fi
        fi
    fi
    echo ""
fi

if [ -z "$TOKEN" ]; then
    echo -e "${RED}❌ No authentication token. Cannot run tests.${NC}"
    exit 1
fi

echo -e "${BLUE}🧪 Chord Articulations Smoke Tests${NC}"
echo "Base URL: $BASE_URL"
echo ""

# Define original chord progression: C major (C-E-G) and Am (A-C-E)
# C major: MIDI 60 (C4), 64 (E4), 67 (G4)
# Am: MIDI 57 (A3), 60 (C4), 64 (E4)
ORIGINAL_CHORD_C="60,64,67"
ORIGINAL_CHORD_AM="57,60,64"

# Test 1: Replay simple chord progression with articulations
echo -e "${YELLOW}Test 1: Replay Chord Progression with Articulations${NC}"
GENERATION_PAYLOAD='{
  "model": "gpt-5-mini",
  "input_array": [
    {
      "role": "user",
      "content": "{\"user_prompt\": \"Replay these chords with rhythm variations, syncopation, and continuous retriggering like a keyboard player would. Use the exact same chord tones but play them at different beats with syncopation and some arpeggios.\", \"notes\": [{\"midiNoteNumber\": 60, \"velocity\": 100, \"startBeats\": 0.0, \"durationBeats\": 4.0}, {\"midiNoteNumber\": 64, \"velocity\": 100, \"startBeats\": 0.0, \"durationBeats\": 4.0}, {\"midiNoteNumber\": 67, \"velocity\": 100, \"startBeats\": 0.0, \"durationBeats\": 4.0}, {\"midiNoteNumber\": 57, \"velocity\": 100, \"startBeats\": 4.0, \"durationBeats\": 4.0}, {\"midiNoteNumber\": 60, \"velocity\": 100, \"startBeats\": 4.0, \"durationBeats\": 4.0}, {\"midiNoteNumber\": 64, \"velocity\": 100, \"startBeats\": 4.0, \"durationBeats\": 4.0}]}"
    }
  ],
  "stream": true,
  "reasoning_mode": "minimal"
}'

echo "  Sending generation request with chord progression..."
echo "  Original chords: C major (beats 0-4), Am (beats 4-8)"
GENERATION_RESPONSE=$(curl -s -N \
  -X POST "$BASE_URL/api/v1/generations" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "$GENERATION_PAYLOAD" 2>&1)

# Extract complete event
RESULT_EVENT=$(echo "$GENERATION_RESPONSE" | grep '"type":"complete"' | head -1 | sed 's/data: //' 2>/dev/null || echo "")

if [ -z "$RESULT_EVENT" ]; then
    # Try "result" instead
    RESULT_EVENT=$(echo "$GENERATION_RESPONSE" | grep '"type":"result"' | head -1 | sed 's/data: //' 2>/dev/null || echo "")
fi

if [ -z "$RESULT_EVENT" ]; then
    echo -e "  ${RED}❌ No result event found in response${NC}"
    FAILED=$((FAILED + 1))
else
    # Parse the result event
    NOTES=$(echo "$RESULT_EVENT" | jq -r '.data.output_parsed.choices[0].notes // []' 2>/dev/null || echo "[]")

    if [ -n "$NOTES" ] && [ "$NOTES" != "[]" ] && [ "$NOTES" != "null" ]; then
        NOTE_COUNT=$(echo "$NOTES" | jq 'length')
        echo -e "  ${GREEN}✅ Generation completed with $NOTE_COUNT notes${NC}"

        # Test 2: Verify original chord tones are preserved
        echo ""
        echo -e "${YELLOW}Test 2: Verify Original Chord Tones Are Preserved${NC}"

        # Extract all MIDI note numbers from the response
        RESPONSE_NOTES=$(echo "$NOTES" | jq -r '[.[] | .midiNoteNumber] | unique | sort | join(",")' 2>/dev/null || echo "")

        # Check if C major chord tones (60, 64, 67) are present
        HAS_C_MAJOR=true
        for note in 60 64 67; do
            if ! echo "$RESPONSE_NOTES" | grep -q "$note"; then
                HAS_C_MAJOR=false
                break
            fi
        done

        # Check if Am chord tones (57, 60, 64) are present
        HAS_AM=true
        for note in 57 60 64; do
            if ! echo "$RESPONSE_NOTES" | grep -q "$note"; then
                HAS_AM=false
                break
            fi
        done

        if [ "$HAS_C_MAJOR" = "true" ] && [ "$HAS_AM" = "true" ]; then
            echo -e "  ${GREEN}✅ Original chord tones preserved (C major: 60,64,67 and Am: 57,60,64)${NC}"
        else
            echo -e "  ${RED}❌ Original chord tones not fully preserved${NC}"
            echo "     Found notes: $RESPONSE_NOTES"
            if [ "$HAS_C_MAJOR" = "false" ]; then
                echo "     Missing C major tones (60, 64, 67)"
            fi
            if [ "$HAS_AM" = "false" ]; then
                echo "     Missing Am tones (57, 60, 64)"
            fi
            FAILED=$((FAILED + 1))
        fi

        # Test 3: Verify syncopation (chords played at off-beats)
        echo ""
        echo -e "${YELLOW}Test 3: Verify Syncopation (Off-Beat Playing)${NC}"

        # Get all startBeats values
        START_BEATS=$(echo "$NOTES" | jq -r '[.[] | .startBeats] | unique | sort | join(",")' 2>/dev/null || echo "")

        # Check for off-beat values (0.5, 1.5, 2.5, 3.5, 4.5, 5.5, 6.5, 7.5)
        HAS_OFFBEATS=false
        OFFBEAT_COUNT=0
        for beat in 0.5 1.5 2.5 3.5 4.5 5.5 6.5 7.5; do
            if echo "$START_BEATS" | grep -q "$beat"; then
                HAS_OFFBEATS=true
                OFFBEAT_COUNT=$((OFFBEAT_COUNT + 1))
            fi
        done

        if [ "$HAS_OFFBEATS" = "true" ]; then
            echo -e "  ${GREEN}✅ Syncopation detected: chords played at off-beats ($OFFBEAT_COUNT off-beat positions)${NC}"
            echo "     Start beats found: $START_BEATS"
        else
            echo -e "  ${YELLOW}⚠️  No syncopation detected (no off-beat playing at 0.5, 1.5, 2.5, etc.)${NC}"
            echo "     Start beats found: $START_BEATS"
            WARNINGS=$((WARNINGS + 1))
        fi

        # Test 4: Verify retriggering (same chord played multiple times)
        echo ""
        echo -e "${YELLOW}Test 4: Verify Retriggering (Same Chord Multiple Times)${NC}"

        # Group notes by their MIDI note numbers and check for multiple occurrences
        # For C major, check if 60, 64, or 67 appear multiple times
        C_MAJOR_NOTE_COUNT=$(echo "$NOTES" | jq -r '[.[] | select(.midiNoteNumber == 60 or .midiNoteNumber == 64 or .midiNoteNumber == 67)] | length' 2>/dev/null || echo "0")
        AM_NOTE_COUNT=$(echo "$NOTES" | jq -r '[.[] | select(.midiNoteNumber == 57 or .midiNoteNumber == 60 or .midiNoteNumber == 64)] | length' 2>/dev/null || echo "0")

        # Original had 3 notes per chord, so retriggering should have more
        if [ "$C_MAJOR_NOTE_COUNT" -gt 3 ] || [ "$AM_NOTE_COUNT" -gt 3 ]; then
            echo -e "  ${GREEN}✅ Retriggering detected: chord tones appear multiple times${NC}"
            echo "     C major chord tones (60,64,67) appear $C_MAJOR_NOTE_COUNT times"
            echo "     Am chord tones (57,60,64) appear $AM_NOTE_COUNT times"
        else
            echo -e "  ${YELLOW}⚠️  Limited retriggering detected (chord tones appear same or fewer times than original)${NC}"
            echo "     C major tones: $C_MAJOR_NOTE_COUNT occurrences"
            echo "     Am tones: $AM_NOTE_COUNT occurrences"
            WARNINGS=$((WARNINGS + 1))
        fi

        # Test 5: Verify arpeggios (sequential notes from same chord)
        echo ""
        echo -e "${YELLOW}Test 5: Verify Arpeggios (Sequential Chord Tones)${NC}"

        # Simple check: if we have many notes and they're not all simultaneous, likely has arpeggios
        SIMULTANEOUS_COUNT=$(echo "$NOTES" | jq -r '[group_by(.startBeats) | .[] | select(length > 2)] | length' 2>/dev/null || echo "0")
        TOTAL_START_BEATS=$(echo "$NOTES" | jq -r '[.[] | .startBeats] | unique | length' 2>/dev/null || echo "0")

        # Check for sequential patterns using jq (more reliable than shell arithmetic)
        # Look for notes from same chord that appear sequentially (not at same startBeats)
        SEQUENTIAL_PATTERNS=$(echo "$NOTES" | jq -r '
          [group_by(.startBeats) | .[] |
           select(length == 1) |
           .[0] |
           select(.midiNoteNumber == 60 or .midiNoteNumber == 64 or .midiNoteNumber == 67 or .midiNoteNumber == 57)
          ] | length' 2>/dev/null || echo "0")

        if [ "$TOTAL_START_BEATS" -gt 4 ]; then
            # More than 4 unique start beats suggests rhythm variation
            echo -e "  ${GREEN}✅ Arpeggios/rhythm variation detected: $TOTAL_START_BEATS unique start beats${NC}"
            echo "     Simultaneous chord groups: $SIMULTANEOUS_COUNT"
            echo "     Sequential single-note patterns: $SEQUENTIAL_PATTERNS"
        else
            echo -e "  ${YELLOW}⚠️  Limited arpeggio/rhythm variation detected${NC}"
            echo "     Unique start beats: $TOTAL_START_BEATS"
            echo "     Sequential patterns: $SEQUENTIAL_PATTERNS"
            WARNINGS=$((WARNINGS + 1))
        fi

        # Test 6: Verify no new chord tones added
        echo ""
        echo -e "${YELLOW}Test 6: Verify No New Chord Tones Added${NC}"

        # Original chords only had: 60, 64, 67 (C major) and 57, 60, 64 (Am)
        # Unique original notes: 57, 60, 64, 67
        ORIGINAL_NOTES="57,60,64,67"

        # Check if response contains notes outside the original set
        EXTRA_NOTES=$(echo "$NOTES" | jq -r "[.[] | .midiNoteNumber] | unique | sort | .[] | select(. != 57 and . != 60 and . != 64 and . != 67)" 2>/dev/null || echo "")

        if [ -z "$EXTRA_NOTES" ]; then
            echo -e "  ${GREEN}✅ No new chord tones added (only original notes: $ORIGINAL_NOTES)${NC}"
        else
            EXTRA_COUNT=$(echo "$EXTRA_NOTES" | wc -l | tr -d ' ')
            echo -e "  ${YELLOW}⚠️  Found $EXTRA_COUNT note(s) outside original chord tones${NC}"
            echo "     Extra notes: $(echo "$EXTRA_NOTES" | tr '\n' ',' | sed 's/,$//')"
            echo "     This might be acceptable if they're passing tones or extensions"
            WARNINGS=$((WARNINGS + 1))
        fi

        # Summary of findings
        echo ""
        echo -e "${BLUE}Summary of Generated Music:${NC}"
        echo "  Total notes: $NOTE_COUNT"
        echo "  Unique MIDI notes: $RESPONSE_NOTES"
        echo "  Unique start beats: $TOTAL_START_BEATS"
        echo "  Off-beat positions: $OFFBEAT_COUNT"

    else
        echo -e "  ${RED}❌ No notes found in final generation${NC}"
        FAILED=$((FAILED + 1))
    fi
fi

echo ""

# Summary
echo "======================================"
if [ $FAILED -eq 0 ]; then
    if [ $WARNINGS -eq 0 ]; then
        echo -e "${GREEN}✅ All chord articulation tests passed!${NC}"
        exit 0
    else
        echo -e "${GREEN}✅ All critical tests passed!${NC}"
        echo -e "${YELLOW}⚠️  $WARNINGS warning(s)${NC}"
        exit 0
    fi
else
    echo -e "${RED}❌ $FAILED test(s) failed${NC}"
    if [ $WARNINGS -gt 0 ]; then
        echo -e "${YELLOW}⚠️  $WARNINGS warning(s)${NC}"
    fi
    exit 1
fi
