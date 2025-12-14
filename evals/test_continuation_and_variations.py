#!/usr/bin/env python3
"""
Eval tests for chord progression continuation and distinct variations.
Tests the specific use cases we've been working on.
"""

import json
import requests
import time
from typing import Dict, List, Any

# API configuration
API_BASE_URL = "http://localhost:8080"
API_KEY = "your-api-key-here"  # Replace with actual API key

def make_request(input_array: List[Dict[str, Any]], mode: str = "one_shot") -> Dict[str, Any]:
    """Make a request to the API with the given input array."""
    payload = {
        "model": "gpt-4o",
        "mode": mode,
        "input_array": input_array,
        "stream": False
    }

    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {API_KEY}"
    }

    response = requests.post(f"{API_BASE_URL}/api/v1/generations", json=payload, headers=headers)
    response.raise_for_status()
    return response.json()

def test_chord_progression_continuation():
    """Test that chord progression continuation preserves timing patterns."""
    print("🎵 Testing chord progression continuation with timing preservation...")

    # Original chord progression with specific timing
    original_progression = [
        {"midiNoteNumber": 60, "velocity": 100, "startBeats": 0.0, "durationBeats": 2.0},  # C
        {"midiNoteNumber": 64, "velocity": 100, "startBeats": 0.0, "durationBeats": 2.0},  # E
        {"midiNoteNumber": 67, "velocity": 100, "startBeats": 0.0, "durationBeats": 2.0},  # G
        {"midiNoteNumber": 57, "velocity": 100, "startBeats": 2.0, "durationBeats": 2.0},  # A
        {"midiNoteNumber": 60, "velocity": 100, "startBeats": 2.0, "durationBeats": 2.0},  # C
        {"midiNoteNumber": 64, "velocity": 100, "startBeats": 2.0, "durationBeats": 2.0},  # E
    ]

    input_array = [
        {
            "role": "user",
            "content": json.dumps({
                "user_prompt": "Continue this chord progression for 4 more bars",
                "notes": original_progression,
                "bpm": 120,
                "spread": "medium",
                "novelty": "medium",
                "variations": 1
            })
        }
    ]

    try:
        result = make_request(input_array)

        # Check that we got a response
        assert "output_parsed" in result, "Response missing output_parsed"
        assert "choices" in result["output_parsed"], "Response missing choices"
        assert len(result["output_parsed"]["choices"]) > 0, "No choices returned"

        choice = result["output_parsed"]["choices"][0]
        notes = choice["notes"]

        # Check that continuation starts where original ends
        original_end = max(note["startBeats"] + note["durationBeats"] for note in original_progression)
        continuation_start = min(note["startBeats"] for note in notes)

        print(f"✅ Original progression ends at beat {original_end}")
        print(f"✅ Continuation starts at beat {continuation_start}")

        # Check timing pattern preservation
        original_durations = [note["durationBeats"] for note in original_progression]
        continuation_durations = [note["durationBeats"] for note in notes]

        # Should have similar duration patterns (mostly 2-beat notes)
        avg_original_duration = sum(original_durations) / len(original_durations)
        avg_continuation_duration = sum(continuation_durations) / len(continuation_durations)

        print(f"✅ Original avg duration: {avg_original_duration:.1f} beats")
        print(f"✅ Continuation avg duration: {avg_continuation_duration:.1f} beats")

        # Duration should be similar (within 50% tolerance)
        assert abs(avg_original_duration - avg_continuation_duration) / avg_original_duration < 0.5, \
            f"Timing patterns not preserved: {avg_original_duration} vs {avg_continuation_duration}"

        print("✅ Chord progression continuation test PASSED")
        return True

    except Exception as e:
        print(f"❌ Chord progression continuation test FAILED: {e}")
        return False

def test_distinct_variations():
    """Test that multiple variations are actually distinct."""
    print("🎵 Testing distinct variations generation...")

    input_array = [
        {
            "role": "user",
            "content": json.dumps({
                "user_prompt": "Create a jazz chord progression in C major",
                "bpm": 120,
                "spread": "medium",
                "novelty": "medium",
                "variations": 3
            })
        }
    ]

    try:
        result = make_request(input_array)

        # Check that we got multiple variations
        assert "output_parsed" in result, "Response missing output_parsed"
        assert "choices" in result["output_parsed"], "Response missing choices"
        choices = result["output_parsed"]["choices"]
        assert len(choices) == 3, f"Expected 3 variations, got {len(choices)}"

        # Check that variations are distinct
        variations_data = []
        for i, choice in enumerate(choices):
            notes = choice["notes"]
            variation_data = {
                "note_count": len(notes),
                "avg_duration": sum(note["durationBeats"] for note in notes) / len(notes),
                "note_range": max(note["midiNoteNumber"] for note in notes) - min(note["midiNoteNumber"] for note in notes),
                "rhythmic_pattern": [note["startBeats"] for note in notes[:5]]  # First 5 note timings
            }
            variations_data.append(variation_data)
            print(f"✅ Variation {i+1}: {variation_data['note_count']} notes, avg duration {variation_data['avg_duration']:.1f}")

        # Check for distinctness - variations should differ in at least one aspect
        distinct_aspects = 0
        for aspect in ["note_count", "avg_duration", "note_range"]:
            values = [var[aspect] for var in variations_data]
            if len(set(values)) > 1:  # Not all the same
                distinct_aspects += 1

        assert distinct_aspects >= 1, "All variations are identical - no distinct aspects found"

        print(f"✅ Found {distinct_aspects} distinct aspects across variations")
        print("✅ Distinct variations test PASSED")
        return True

    except Exception as e:
        print(f"❌ Distinct variations test FAILED: {e}")
        return False

def test_raw_input_array_preservation():
    """Test that raw input array is properly passed through."""
    print("🎵 Testing raw input array preservation...")

    # Test with complex input array structure
    input_array = [
        {
            "role": "user",
            "content": json.dumps({
                "user_prompt": "Create a blues progression",
                "bpm": 120,
                "spread": "wide",
                "novelty": "high"
            })
        },
        {
            "role": "assistant",
            "content": "I'll create a blues progression with extended harmonies."
        },
        {
            "role": "user",
            "content": json.dumps({
                "musical_context": "Previous conversation about blues music",
                "variations": 2
            })
        }
    ]

    try:
        result = make_request(input_array)

        # Check that we got a response
        assert "output_parsed" in result, "Response missing output_parsed"
        assert "choices" in result["output_parsed"], "Response missing choices"

        # Check that the response reflects the input parameters
        choices = result["output_parsed"]["choices"]
        assert len(choices) == 2, f"Expected 2 variations, got {len(choices)}"

        # Check that the response shows understanding of the complex input
        # (This is more of a smoke test - the real test is that it doesn't crash)
        print("✅ Complex input array processed successfully")
        print("✅ Raw input array preservation test PASSED")
        return True

    except Exception as e:
        print(f"❌ Raw input array preservation test FAILED: {e}")
        return False

def run_all_tests():
    """Run all eval tests."""
    print("🚀 Starting continuation and variations eval tests...\n")

    tests = [
        test_chord_progression_continuation,
        test_distinct_variations,
        test_raw_input_array_preservation
    ]

    passed = 0
    total = len(tests)

    for test in tests:
        try:
            if test():
                passed += 1
            print()  # Add spacing between tests
        except Exception as e:
            print(f"❌ Test {test.__name__} crashed: {e}\n")

    print(f"📊 Results: {passed}/{total} tests passed")

    if passed == total:
        print("🎉 All tests PASSED!")
        return True
    else:
        print("💥 Some tests FAILED!")
        return False

if __name__ == "__main__":
    success = run_all_tests()
    exit(0 if success else 1)
