"""
Chord Detection Scorer
Evaluates whether the generated MIDI contains valid chords at expected positions.
"""

from music21 import converter, chord as m21_chord, note, stream
from typing import Dict, Any, List


def parse_midi_base64(midi_base64: str) -> stream.Stream:
    """Parse base64-encoded MIDI data into music21 Stream."""
    import base64
    import io

    midi_bytes = base64.b64decode(midi_base64)
    return converter.parse(io.BytesIO(midi_bytes))


def extract_chords(score: stream.Stream) -> List[Dict[str, Any]]:
    """
    Extract chords from a music21 score.
    Returns list of dicts with {beat, chord_symbol, notes}.
    """
    chords = []

    # Flatten the score to get all notes in sequence
    flat = score.flatten()

    # Try to extract explicit chord symbols first
    chord_symbols = flat.getElementsByClass(m21_chord.Chord)

    if chord_symbols:
        for chord_obj in chord_symbols:
            chords.append({
                'beat': chord_obj.offset,
                'chord_symbol': chord_obj.pitchedCommonName,
                'notes': [p.nameWithOctave for p in chord_obj.pitches]
            })
    else:
        # Analyze vertical slices to find chords
        chordified = flat.chordify()
        for element in chordified.flatten().notesAndRests:
            if isinstance(element, m21_chord.Chord):
                chords.append({
                    'beat': element.offset,
                    'chord_symbol': element.pitchedCommonName,
                    'notes': [p.nameWithOctave for p in element.pitches]
                })

    return chords


def score_chord_detection(output: Dict[str, Any], expected: Dict[str, Any]) -> Dict[str, Any]:
    """
    Score the chord detection quality.

    Args:
        output: The API response containing midi_base64
        expected: Expected properties (min_chords, etc.)

    Returns:
        Dict with score (0-1) and details
    """
    try:
        # Parse MIDI
        midi_data = output.get('midi_base64', '')
        if not midi_data:
            return {
                'score': 0.0,
                'reason': 'No MIDI data in output',
                'chords_found': 0
            }

        score_stream = parse_midi_base64(midi_data)
        chords = extract_chords(score_stream)

        # Check minimum chord count
        min_chords = expected.get('min_chords', 4)
        chord_count = len(chords)

        if chord_count == 0:
            return {
                'score': 0.0,
                'reason': 'No chords detected',
                'chords_found': 0
            }

        # Score based on meeting minimum requirement
        meets_minimum = chord_count >= min_chords

        # Check for 7th chords if required
        has_7ths_required = expected.get('requires_7ths', False)
        if has_7ths_required:
            has_7ths = any('7' in c['chord_symbol'] for c in chords)
            if not has_7ths:
                return {
                    'score': 0.5,
                    'reason': 'Missing required 7th chords',
                    'chords_found': chord_count,
                    'chords': chords
                }

        # Calculate final score
        if meets_minimum:
            score = 1.0
        else:
            # Partial credit for having some chords
            score = chord_count / min_chords

        return {
            'score': score,
            'chords_found': chord_count,
            'min_required': min_chords,
            'chords': chords
        }

    except Exception as e:
        return {
            'score': 0.0,
            'reason': f'Error parsing MIDI: {str(e)}',
            'chords_found': 0
        }
