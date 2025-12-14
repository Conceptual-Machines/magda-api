"""
Key/Scale Validation Scorer
Evaluates whether the generated MIDI matches the requested key and scale.
"""

from music21 import converter, key, stream
from typing import Dict, Any
import base64
import io


def parse_midi_base64(midi_base64: str) -> stream.Stream:
    """Parse base64-encoded MIDI data into music21 Stream."""
    midi_bytes = base64.b64decode(midi_base64)
    return converter.parse(io.BytesIO(midi_bytes))


def detect_key(score: stream.Stream) -> key.Key:
    """Detect the key of a music21 score."""
    # Try to get key from score
    existing_key = score.analyze('key')
    return existing_key


def normalize_key_name(key_name: str) -> str:
    """Normalize key names for comparison (e.g., 'C#' -> 'C♯', 'Bb' -> 'B♭')."""
    replacements = {
        '#': '♯',
        'b': '♭',
        'sharp': '♯',
        'flat': '♭'
    }

    normalized = key_name
    for old, new in replacements.items():
        normalized = normalized.replace(old, new)

    return normalized.strip()


def keys_match(detected: str, expected: str) -> bool:
    """Check if two key names represent the same key."""
    detected_norm = normalize_key_name(detected).upper()
    expected_norm = normalize_key_name(expected).upper()

    # Direct match
    if detected_norm == expected_norm:
        return True

    # Handle enharmonic equivalents
    enharmonics = {
        'C♯': 'D♭',
        'D♭': 'C♯',
        'D♯': 'E♭',
        'E♭': 'D♯',
        'F♯': 'G♭',
        'G♭': 'F♯',
        'G♯': 'A♭',
        'A♭': 'G♯',
        'A♯': 'B♭',
        'B♭': 'A♯',
    }

    for norm in [detected_norm, expected_norm]:
        # Remove 'MAJOR' or 'MINOR' suffix for comparison
        clean = norm.replace(' MAJOR', '').replace(' MINOR', '')
        if clean in enharmonics and enharmonics[clean] in [detected_norm, expected_norm]:
            return True

    return False


def score_key_validation(output: Dict[str, Any], expected: Dict[str, Any]) -> Dict[str, Any]:
    """
    Score the key/scale accuracy.

    Args:
        output: The API response containing midi_base64
        expected: Expected properties (key, scale)

    Returns:
        Dict with score (0-1) and details
    """
    try:
        # Parse MIDI
        midi_data = output.get('midi_base64', '')
        if not midi_data:
            return {
                'score': 0.0,
                'reason': 'No MIDI data in output'
            }

        score_stream = parse_midi_base64(midi_data)
        detected_key = detect_key(score_stream)

        # Expected key and scale
        expected_key = expected.get('key', '')
        expected_scale = expected.get('scale', 'major')

        # Build expected key string
        expected_key_full = f"{expected_key} {expected_scale}"

        # Check if keys match
        detected_tonic = detected_key.tonic.name
        detected_mode = detected_key.mode
        detected_key_str = f"{detected_tonic} {detected_mode}"

        # Compare tonic
        tonic_match = keys_match(detected_tonic, expected_key)

        # Compare mode/scale
        mode_match = detected_mode.lower() == expected_scale.lower()

        # Special handling for modes (dorian, phrygian, etc.)
        if expected_scale.lower() in ['dorian', 'phrygian', 'lydian', 'mixolydian', 'aeolian', 'locrian']:
            # For modes, just check if tonic matches (mode detection is harder)
            if tonic_match:
                return {
                    'score': 0.8,  # Partial credit for modal ambiguity
                    'detected_key': detected_key_str,
                    'expected_key': expected_key_full,
                    'reason': 'Tonic matches (modal detection is approximate)'
                }

        # Calculate score
        if tonic_match and mode_match:
            score = 1.0
        elif tonic_match:
            score = 0.6  # Right tonic, wrong mode
        else:
            score = 0.0  # Wrong key entirely

        return {
            'score': score,
            'detected_key': detected_key_str,
            'expected_key': expected_key_full,
            'tonic_match': tonic_match,
            'mode_match': mode_match
        }

    except Exception as e:
        return {
            'score': 0.0,
            'reason': f'Error analyzing key: {str(e)}'
        }
