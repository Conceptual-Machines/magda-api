"""
Content Type Classification Scorer
Evaluates whether the output contains the expected content type:
- chords_only
- chords_and_melody
- full_arrangement (chords + melody + bass)
"""

from music21 import converter, instrument, stream
from typing import Dict, Any, List
import base64
import io


def parse_midi_base64(midi_base64: str) -> stream.Stream:
    """Parse base64-encoded MIDI data into music21 Stream."""
    midi_bytes = base64.b64decode(midi_base64)
    return converter.parse(io.BytesIO(midi_bytes))


def analyze_content_type(score: stream.Stream) -> Dict[str, Any]:
    """
    Analyze the content type of a music21 score.

    Returns:
        Dict with detected content type and track info
    """
    parts = score.parts

    analysis = {
        'num_parts': len(parts),
        'has_chords': False,
        'has_melody': False,
        'has_bass': False,
        'parts_info': []
    }

    for part in parts:
        part_info = {
            'instrument': str(part.getInstrument()),
            'notes_count': len(part.flatten().notes),
            'avg_pitch': None,
            'pitch_range': None
        }

        # Get pitch statistics
        pitches = [n.pitch.midi for n in part.flatten().notes if hasattr(n, 'pitch')]
        if pitches:
            part_info['avg_pitch'] = sum(pitches) / len(pitches)
            part_info['pitch_range'] = max(pitches) - min(pitches)

            # Classify by pitch range
            avg_pitch = part_info['avg_pitch']
            if avg_pitch < 48:  # Below C3
                analysis['has_bass'] = True
                part_info['role'] = 'bass'
            elif avg_pitch > 72:  # Above C5
                analysis['has_melody'] = True
                part_info['role'] = 'melody'
            else:
                # Check if it's chordal (multiple simultaneous notes)
                chordified = part.chordify()
                if len(chordified.flatten().getElementsByClass('Chord')) > 0:
                    analysis['has_chords'] = True
                    part_info['role'] = 'chords'
                else:
                    # Could be melody in mid range
                    analysis['has_melody'] = True
                    part_info['role'] = 'melody'

        analysis['parts_info'].append(part_info)

    # Determine content type
    if analysis['has_chords'] and analysis['has_melody'] and analysis['has_bass']:
        content_type = 'full_arrangement'
    elif analysis['has_chords'] and analysis['has_melody']:
        content_type = 'chords_and_melody'
    elif analysis['has_chords']:
        content_type = 'chords'
    elif analysis['has_melody']:
        content_type = 'melody_only'
    else:
        content_type = 'unknown'

    analysis['detected_type'] = content_type
    return analysis


def score_content_type(output: Dict[str, Any], expected: Dict[str, Any]) -> Dict[str, Any]:
    """
    Score the content type accuracy.

    Args:
        output: The API response containing midi_base64
        expected: Expected properties (content_type)

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
        analysis = analyze_content_type(score_stream)

        expected_type = expected.get('content_type', 'chords')
        detected_type = analysis['detected_type']

        # Exact match
        if detected_type == expected_type:
            return {
                'score': 1.0,
                'detected_type': detected_type,
                'expected_type': expected_type,
                'analysis': analysis
            }

        # Partial credit for over-delivering
        # (e.g., full_arrangement when chords_and_melody was requested)
        upgrade_paths = {
            'chords': ['chords_and_melody', 'full_arrangement'],
            'chords_and_melody': ['full_arrangement']
        }

        if expected_type in upgrade_paths:
            if detected_type in upgrade_paths[expected_type]:
                return {
                    'score': 0.9,  # Slightly lower for over-delivering
                    'detected_type': detected_type,
                    'expected_type': expected_type,
                    'reason': 'More content than requested (acceptable)',
                    'analysis': analysis
                }

        # Partial credit for related types
        if expected_type == 'chords' and detected_type in ['chords_and_melody', 'full_arrangement']:
            score = 0.7
        elif expected_type == 'chords_and_melody' and detected_type == 'chords':
            score = 0.5  # Missing melody
        elif expected_type == 'full_arrangement' and detected_type == 'chords_and_melody':
            score = 0.7  # Missing bass
        else:
            score = 0.0

        return {
            'score': score,
            'detected_type': detected_type,
            'expected_type': expected_type,
            'analysis': analysis
        }

    except Exception as e:
        return {
            'score': 0.0,
            'reason': f'Error analyzing content: {str(e)}'
        }
