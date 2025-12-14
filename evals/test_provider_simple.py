#!/usr/bin/env python3
"""
Simple integration test to verify the Provider refactoring works.
Tests both non-streaming and streaming generation.
"""

import json
import os
import requests
import sys

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

def test_non_streaming(token: str):
    """Test non-streaming generation."""
    print("🎵 Testing non-streaming generation...")

    payload = {
        "model": "gpt-5-mini",
        "reasoning_mode": "low",
        "input_array": [
            {
                "role": "user",
                "content": json.dumps({
                    "user_prompt": "Create a simple 2-bar chord progression in C major",
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

    response = requests.post(f"{API_BASE_URL}/api/v1/generations", json=payload, headers=headers)
    response.raise_for_status()
    result = response.json()

    assert "output_parsed" in result, "Missing output_parsed"
    assert "choices" in result["output_parsed"], "Missing choices"
    assert len(result["output_parsed"]["choices"]) > 0, "No choices returned"

    choice = result["output_parsed"]["choices"][0]
    assert "notes" in choice, "Missing notes"
    assert len(choice["notes"]) > 0, "No notes generated"

    print(f"✅ Non-streaming test PASSED - Generated {len(choice['notes'])} notes")
    print(f"   Description: {choice['description'][:80]}...")
    return True

def test_streaming(token: str):
    """Test streaming generation."""
    print("🎵 Testing streaming generation...")

    payload = {
        "model": "gpt-5-mini",
        "reasoning_mode": "minimal",
        "input_array": [
            {
                "role": "user",
                "content": json.dumps({
                    "user_prompt": "Create a simple bassline in C minor",
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

    # For streaming, we need to handle SSE
    response = requests.post(
        f"{API_BASE_URL}/api/v1/generations",
        json=payload,
        headers=headers,
        stream=True
    )
    response.raise_for_status()

    events_received = 0
    completed = False

    for line in response.iter_lines():
        if not line:
            continue

        line = line.decode('utf-8')
        if line.startswith('data: '):
            data_str = line[6:]  # Remove 'data: ' prefix
            try:
                event = json.loads(data_str)
                events_received += 1
                if event.get('type') == 'completed':
                    completed = True
                    print(f"✅ Streaming test PASSED - Received {events_received} events")
                    return True
            except json.JSONDecodeError:
                continue

    if not completed:
        print(f"❌ Streaming test FAILED - Stream ended without completion event")
        return False

if __name__ == "__main__":
    if not EMAIL or not PASSWORD:
        print("❌ Error: AIDEAS_EMAIL and AIDEAS_PASSWORD must be set")
        sys.exit(1)

    print(f"🚀 Testing Provider refactoring against {API_BASE_URL}")
    print(f"📧 Email: {EMAIL}")
    print()

    try:
        # Register or login
        print("🔐 Authenticating...")
        token = register_or_login()
        print("✅ Authentication successful")
        print()

        # Run tests
        results = []
        results.append(test_non_streaming(token))
        print()
        results.append(test_streaming(token))
        print()

        # Summary
        passed = sum(results)
        total = len(results)
        print(f"📊 Results: {passed}/{total} tests passed")

        if passed == total:
            print("✅ All tests PASSED!")
            sys.exit(0)
        else:
            print("❌ Some tests FAILED!")
            sys.exit(1)

    except Exception as e:
        print(f"❌ Test suite failed: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
