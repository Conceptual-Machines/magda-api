#!/usr/bin/env python3
"""
Integration test to compare DSL vs JSON Schema output formats.
Tests both formats with the same input and compares:
- Token usage (output tokens)
- Latency (time to first token, total time)
- Output correctness
"""

import json
import os
import requests
import sys
import time
from typing import Dict, Any, Tuple

API_BASE_URL = os.getenv("AIDEAS_API_URL", "http://localhost:8080")
EMAIL = os.getenv("AIDEAS_EMAIL")
PASSWORD = os.getenv("AIDEAS_PASSWORD")

def register_or_login() -> str:
    """Register beta user or login if already exists, return access token."""
    # Try to register as beta user first
    response = requests.post(
        f"{API_BASE_URL}/api/auth/register/beta",
        json={"email": EMAIL, "password": PASSWORD}
    )

    if response.status_code == 200:
        print("✅ Registered new beta user")
        data = response.json()
        return data["access_token"]

    # If registration failed, try login
    if "already exists" in response.text.lower():
        print("ℹ️  User exists, logging in...")
        response = requests.post(
            f"{API_BASE_URL}/api/auth/login",
            json={"email": EMAIL, "password": PASSWORD}
        )
        response.raise_for_status()
        data = response.json()
        return data["access_token"]

    # Something else went wrong
    response.raise_for_status()
    return ""

def test_output_format(token: str, output_format: str, test_name: str) -> Dict[str, Any]:
    """Test generation with a specific output format and return metrics."""
    print(f"\n🔧 Testing {test_name} (output_format={output_format})...")

    payload = {
        "model": "gpt-5-mini",
        "reasoning_mode": "low",
        "output_format": output_format,
        "input_array": [
            {
                "role": "user",
                "content": json.dumps({
                    "user_prompt": "Create a simple 2-bar C major chord progression with 3-4 notes per chord",
                    "bpm": 120,
                    "variations": 1
                })
            }
        ],
        "stream": False
    }

    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {token}"
    }

    # Measure timing
    start_time = time.time()
    response = requests.post(
        f"{API_BASE_URL}/api/v1/generations",
        json=payload,
        headers=headers
    )
    elapsed_time = time.time() - start_time
    response.raise_for_status()
    result = response.json()

    # Extract metrics
    usage = result.get("usage", {})
    output_tokens = usage.get("output_tokens", 0) if isinstance(usage, dict) else 0
    input_tokens = usage.get("input_tokens", 0) if isinstance(usage, dict) else 0
    total_tokens = usage.get("total_tokens", 0) if isinstance(usage, dict) else 0

    # Verify output structure
    assert "output_parsed" in result, f"{test_name}: Missing output_parsed"
    assert "choices" in result["output_parsed"], f"{test_name}: Missing choices"
    assert len(result["output_parsed"]["choices"]) > 0, f"{test_name}: No choices returned"

    choice = result["output_parsed"]["choices"][0]
    assert "notes" in choice, f"{test_name}: Missing notes"
    assert len(choice["notes"]) > 0, f"{test_name}: No notes generated"

    metrics = {
        "format": output_format,
        "elapsed_time": elapsed_time,
        "output_tokens": output_tokens,
        "input_tokens": input_tokens,
        "total_tokens": total_tokens,
        "choices_count": len(result["output_parsed"]["choices"]),
        "notes_count": len(choice["notes"]),
        "description": choice.get("description", "")[:80],
        "success": True
    }

    print(f"   ✅ Success - {metrics['notes_count']} notes, {metrics['output_tokens']} output tokens, {elapsed_time:.2f}s")
    return metrics

def test_streaming_output_format(token: str, output_format: str, test_name: str) -> Dict[str, Any]:
    """Test streaming generation with a specific output format and return metrics."""
    print(f"\n🔧 Testing STREAMING {test_name} (output_format={output_format})...")

    payload = {
        "model": "gpt-5-mini",
        "reasoning_mode": "minimal",
        "output_format": output_format,
        "input_array": [
            {
                "role": "user",
                "content": json.dumps({
                    "user_prompt": "Create a simple bassline in C minor with 4-6 notes",
                    "bpm": 120,
                    "variations": 1
                })
            }
        ],
        "stream": True
    }

    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {token}"
    }

    # Measure timing
    start_time = time.time()
    first_token_time = None

    response = requests.post(
        f"{API_BASE_URL}/api/v1/generations",
        json=payload,
        headers=headers,
        stream=True
    )
    response.raise_for_status()

    events_received = 0
    completed = False
    final_result = None

    for line in response.iter_lines():
        if not line:
            continue

        if first_token_time is None:
            first_token_time = time.time() - start_time

        line = line.decode('utf-8')
        if line.startswith('data: '):
            data_str = line[6:]  # Remove 'data: ' prefix
            try:
                event = json.loads(data_str)
                events_received += 1

                if event.get('type') == 'completed':
                    completed = True
                    final_result = event.get('data', {})
                    break
                elif event.get('type') == 'done':
                    completed = True
                    final_result = event.get('data', {})
                    break
            except json.JSONDecodeError:
                continue

    elapsed_time = time.time() - start_time

    if not completed or not final_result:
        print(f"   ❌ FAILED - Stream ended without completion event")
        return {
            "format": output_format,
            "success": False,
            "error": "Stream incomplete"
        }

    # Extract metrics from final result
    usage = final_result.get("usage", {})
    output_tokens = usage.get("output_tokens", 0) if isinstance(usage, dict) else 0
    input_tokens = usage.get("input_tokens", 0) if isinstance(usage, dict) else 0
    total_tokens = usage.get("total_tokens", 0) if isinstance(usage, dict) else 0

    output_parsed = final_result.get("output_parsed", {})
    choices = output_parsed.get("choices", [])
    notes_count = len(choices[0].get("notes", [])) if choices else 0

    metrics = {
        "format": output_format,
        "elapsed_time": elapsed_time,
        "first_token_time": first_token_time if first_token_time else 0,
        "output_tokens": output_tokens,
        "input_tokens": input_tokens,
        "total_tokens": total_tokens,
        "events_received": events_received,
        "choices_count": len(choices),
        "notes_count": notes_count,
        "description": choices[0].get("description", "")[:80] if choices else "",
        "success": True
    }

    print(f"   ✅ Success - {metrics['notes_count']} notes, {metrics['output_tokens']} output tokens")
    print(f"      First token: {first_token_time:.2f}s, Total: {elapsed_time:.2f}s, Events: {events_received}")
    return metrics

def compare_results(dsl_metrics: Dict[str, Any], json_metrics: Dict[str, Any], test_type: str):
    """Compare and display results between DSL and JSON formats."""
    print(f"\n{'='*60}")
    print(f"📊 {test_type.upper()} COMPARISON RESULTS")
    print(f"{'='*60}")

    if not dsl_metrics.get("success") or not json_metrics.get("success"):
        print("❌ One or both tests failed - cannot compare")
        return

    # Token comparison
    dsl_output_tokens = dsl_metrics.get("output_tokens", 0)
    json_output_tokens = json_metrics.get("output_tokens", 0)

    if json_output_tokens > 0:
        token_savings = ((json_output_tokens - dsl_output_tokens) / json_output_tokens) * 100
        print(f"\n📉 OUTPUT TOKENS:")
        print(f"   DSL:     {dsl_output_tokens:,} tokens")
        print(f"   JSON:    {json_output_tokens:,} tokens")
        print(f"   Savings: {token_savings:.1f}% ({json_output_tokens - dsl_output_tokens:,} tokens)")

    # Time comparison
    dsl_time = dsl_metrics.get("elapsed_time", 0)
    json_time = json_metrics.get("elapsed_time", 0)

    if json_time > 0:
        time_improvement = ((json_time - dsl_time) / json_time) * 100
        print(f"\n⏱️  LATENCY:")
        print(f"   DSL:     {dsl_time:.2f}s")
        print(f"   JSON:    {json_time:.2f}s")
        print(f"   Faster:  {time_improvement:.1f}% ({json_time - dsl_time:.2f}s saved)")

    # Streaming first token comparison
    if "first_token_time" in dsl_metrics and "first_token_time" in json_metrics:
        dsl_first = dsl_metrics.get("first_token_time", 0)
        json_first = json_metrics.get("first_token_time", 0)

        if json_first > 0:
            first_token_improvement = ((json_first - dsl_first) / json_first) * 100
            print(f"\n🚀 FIRST TOKEN TIME:")
            print(f"   DSL:     {dsl_first:.2f}s")
            print(f"   JSON:    {json_first:.2f}s")
            print(f"   Faster:  {first_token_improvement:.1f}% ({json_first - dsl_first:.2f}s saved)")

    # Output quality comparison
    print(f"\n✅ OUTPUT QUALITY:")
    print(f"   DSL choices:  {dsl_metrics.get('choices_count', 0)}")
    print(f"   JSON choices: {json_metrics.get('choices_count', 0)}")
    print(f"   DSL notes:    {dsl_metrics.get('notes_count', 0)}")
    print(f"   JSON notes:   {json_metrics.get('notes_count', 0)}")

def main():
    if not EMAIL or not PASSWORD:
        print("❌ Error: AIDEAS_EMAIL and AIDEAS_PASSWORD must be set")
        sys.exit(1)

    print(f"🚀 Testing DSL vs JSON Schema output formats")
    print(f"📡 API URL: {API_BASE_URL}")
    print(f"📧 Email: {EMAIL}")
    print()

    try:
        # Authenticate
        print("🔐 Authenticating...")
        token = register_or_login()
        print("✅ Authentication successful")
        print()

        results = {
            "non_streaming": {},
            "streaming": {}
        }

        # Test non-streaming DSL
        print("=" * 60)
        print("NON-STREAMING TESTS")
        print("=" * 60)
        results["non_streaming"]["dsl"] = test_output_format(token, "dsl", "DSL Format")

        # Small delay between tests
        time.sleep(2)

        # Test non-streaming JSON Schema
        results["non_streaming"]["json"] = test_output_format(token, "json_schema", "JSON Schema Format")

        # Compare non-streaming results
        compare_results(
            results["non_streaming"]["dsl"],
            results["non_streaming"]["json"],
            "Non-Streaming"
        )

        # Test streaming DSL
        print("\n" + "=" * 60)
        print("STREAMING TESTS")
        print("=" * 60)
        results["streaming"]["dsl"] = test_streaming_output_format(token, "dsl", "DSL Format")

        # Small delay between tests
        time.sleep(2)

        # Test streaming JSON Schema
        results["streaming"]["json"] = test_streaming_output_format(token, "json_schema", "JSON Schema Format")

        # Compare streaming results
        compare_results(
            results["streaming"]["dsl"],
            results["streaming"]["json"],
            "Streaming"
        )

        # Final summary
        print("\n" + "=" * 60)
        print("📊 FINAL SUMMARY")
        print("=" * 60)

        if results["non_streaming"]["dsl"].get("success") and results["non_streaming"]["json"].get("success"):
            dsl_tokens = results["non_streaming"]["dsl"].get("output_tokens", 0)
            json_tokens = results["non_streaming"]["json"].get("output_tokens", 0)
            if json_tokens > 0:
                savings = ((json_tokens - dsl_tokens) / json_tokens) * 100
                print(f"✅ Token savings: {savings:.1f}%")

        if results["non_streaming"]["dsl"].get("success") and results["non_streaming"]["json"].get("success"):
            dsl_time = results["non_streaming"]["dsl"].get("elapsed_time", 0)
            json_time = results["non_streaming"]["json"].get("elapsed_time", 0)
            if json_time > 0:
                improvement = ((json_time - dsl_time) / json_time) * 100
                print(f"✅ Latency improvement: {improvement:.1f}%")

        print("\n✅ All comparison tests completed!")
        sys.exit(0)

    except Exception as e:
        print(f"❌ Test suite failed: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

if __name__ == "__main__":
    main()
