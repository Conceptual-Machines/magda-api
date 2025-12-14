#!/usr/bin/env python3
"""
Integration test specifically for Gemini provider.
"""

import json
import os
import requests
import sys

API_BASE_URL = "http://localhost:8080"
EMAIL = os.getenv("AIDEAS_EMAIL", "admin@musicalaideas.com")
PASSWORD = os.getenv("AIDEAS_PASSWORD")

def register_or_login() -> str:
    """Register beta user or login if already exists."""
    response = requests.post(
        f"{API_BASE_URL}/api/auth/register/beta",
        json={"email": EMAIL, "password": PASSWORD}
    )

    if response.status_code == 200:
        return response.json()["access_token"]

    if "already exists" in response.text.lower():
        response = requests.post(
            f"{API_BASE_URL}/api/auth/login",
            json={"email": EMAIL, "password": PASSWORD}
        )
        response.raise_for_status()
        return response.json()["access_token"]

    response.raise_for_status()
    return ""

def test_gemini_generation(token: str, model: str, stream: bool):
    """Test Gemini generation (streaming or non-streaming)."""
    mode = "streaming" if stream else "non-streaming"
    print(f"🎵 Testing Gemini {mode} with model: {model}")

    payload = {
        "model": model,
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
        "stream": stream
    }

    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {token}"
    }

    if stream:
        # Handle streaming response
        response = requests.post(
            f"{API_BASE_URL}/api/v1/generations",
            json=payload,
            headers=headers,
            stream=True
        )
        response.raise_for_status()

        events_received = 0
        for line in response.iter_lines():
            if not line:
                continue

            line = line.decode('utf-8')
            if line.startswith('data: '):
                data_str = line[6:]
                try:
                    event = json.loads(data_str)
                    events_received += 1
                    if event.get('type') == 'completed':
                        print(f"✅ Gemini streaming test PASSED - {events_received} events")
                        return True
                except json.JSONDecodeError:
                    continue

        print(f"❌ Stream ended without completion")
        return False
    else:
        # Non-streaming response
        response = requests.post(f"{API_BASE_URL}/api/v1/generations", json=payload, headers=headers)
        response.raise_for_status()
        result = response.json()

        assert "output_parsed" in result
        assert "choices" in result["output_parsed"]
        assert len(result["output_parsed"]["choices"]) > 0

        choice = result["output_parsed"]["choices"][0]
        note_count = len(choice["notes"])
        print(f"✅ Gemini non-streaming test PASSED - Generated {note_count} notes")
        print(f"   Description: {choice['description'][:80]}...")
        return True

if __name__ == "__main__":
    if not PASSWORD:
        print("❌ AIDEAS_PASSWORD must be set")
        sys.exit(1)

    print(f"🚀 Testing Gemini Provider")
    print(f"📧 Email: {EMAIL}")
    print()

    try:
        # Login
        print("🔐 Authenticating...")
        token = register_or_login()
        print("✅ Authenticated")
        print()

        # Test both Gemini models (non-streaming and streaming)
        results = []

        # Test gemini-2.0-flash-exp
        results.append(test_gemini_generation(token, "gemini-2.0-flash-exp", False))
        print()
        results.append(test_gemini_generation(token, "gemini-2.0-flash-exp", True))
        print()

        # Test gemini-exp-1206
        results.append(test_gemini_generation(token, "gemini-exp-1206", False))
        print()
        results.append(test_gemini_generation(token, "gemini-exp-1206", True))
        print()

        # Summary
        passed = sum(results)
        total = len(results)
        print(f"📊 Results: {passed}/{total} tests passed")

        if passed == total:
            print("✅ All Gemini tests PASSED!")
            sys.exit(0)
        else:
            print("❌ Some Gemini tests FAILED!")
            sys.exit(1)

    except Exception as e:
        print(f"❌ Test failed: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
