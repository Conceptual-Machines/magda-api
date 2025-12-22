# MAGDA Agents Overview

## Directory Structure

```
internal/agents/
├── core/                    # Orchestration & shared config
│   ├── coordination/        # Agent orchestrator (routes requests)
│   └── config/             # Agent configuration
│
├── reaper/                  # REAPER-specific agents
│   ├── daw/                # REAPER DAW control agent
│   ├── jsfx/               # JSFX effect generator
│   └── plugin/             # Plugin management
│
├── shared/                  # DAW-agnostic agents
│   ├── drummer/            # Drum pattern generator
│   ├── arranger/           # Musical content (chords, melodies)
│   └── mix/                # Mix analysis agent
│
└── (future) ableton/        # Ableton-specific agents
    ├── daw/
    └── devices/
```

## Core Agents

### **Coordination/Orchestrator** (`core/coordination/`)
**Purpose**: Routes user requests to appropriate agents
**Knows**: Agent capabilities, request classification
**Generates**: Coordinated responses from multiple agents

## REAPER-Specific Agents (`reaper/`)

### **DAW Agent** (`reaper/daw/`) ✅
**Purpose**: Understands REAPER API structure and generates actions
**Knows**: Track creation, clip placement, FX routing, REAPER object hierarchy
**Generates**: REAPER API actions (create_track, add_fx, etc.)

### **JSFX Agent** (`reaper/jsfx/`) ✅
**Purpose**: Generates JSFX audio effects code
**Knows**: JSFX syntax, DSP algorithms, audio processing
**Generates**: Complete JSFX effect code

### **Plugin Agent** (`reaper/plugin/`) ✅
**Purpose**: Plugin management, deduplication, alias generation
**Knows**: Plugin formats, naming conventions, preferences
**Generates**: Plugin aliases and deduplication mappings

## Shared/DAW-Agnostic Agents (`shared/`)

### **Arranger Agent** (`shared/arranger/`) ✅
**Purpose**: Generates musical content (chords, melodies, progressions)
**Knows**: Music theory, chord progressions, Roman numerals, chord symbols
**Generates**: NoteEvent arrays from musical descriptions

### **Drummer Agent** (`shared/drummer/`) ✅
**Purpose**: Generates drum patterns
**Knows**: Drum patterns, rhythms, grooves
**Generates**: Grid-based drum patterns

### **Mix/Analysis Agent** (`shared/mix/`) ✅
**Purpose**: DSP analysis + mixing/mastering recommendations
**Knows**: Audio analysis, frequency spectrum, dynamics, mixing techniques
**Generates**: Analysis insights and mixing recommendations

## Designed But Not Implemented

### **Automation Agent** 📝 (Designed)
**Purpose**: Draws automation curves for volume, pan, FX parameters
**Knows**: Curve types, interpolation, musical timing, envelope shapes
**Generates**: AutomationCurve with points and interpolation settings

## Proposed Additional Agents

### 5. **Mix/Analysis Agent** 🎚️ (Unified)
**Purpose**: DSP analysis + mixing/mastering recommendations
**Knows**: Audio analysis, frequency spectrum, dynamics, mixing/mastering techniques, FX chains
**Generates**: Analysis insights and mixing/mastering recommendations

**Workflow:**
1. **Bounce** track(s) or master bus to audio
2. **DSP Analysis** using JSFX inside REAPER (frequency, loudness, dynamics, stereo)
3. **Agent Analysis** - LLM analyzes DSP data and provides recommendations
4. **Review & Accept** - User reviews recommendations with Accept/Reject/Modify workflow
5. **Apply Changes** - Accepted recommendations generate REAPER actions

**Modes:**
- **Track Mode**: Analyze individual track(s)
- **Multi-Track Mode**: Analyze all tracks and relationships (frequency masking, phase, etc.)
- **Master Mode**: Analyze master bus for mastering recommendations

**Use Cases:**
- "Make the bass sit better in the mix" (track analysis)
- "Analyze the whole mix and optimize it" (multi-track analysis)
- "Master to streaming standards" (master bus analysis)
- "Add subtle compression to the drums"
- "EQ out muddiness from the guitars"

**Input**: DSP analysis data (from bounced audio), mixing requests
**Output**: Recommendations with FX chains and parameter settings

**Coordination**:
- REAPER extension bounces audio and performs DSP analysis
- Works with DAW Agent to apply recommendations as actions
- Requires Accept/Reject workflow UI

**See**: `agents/mix/MIX_ANALYSIS_AGENT_DESIGN.md` for detailed workflow

---

### 6. **Sound Design Agent** 🎹
**Purpose**: Synthesizer programming and sound shaping
**Knows**: Synth parameters (oscillators, filters, envelopes, LFOs), sound types (bass, lead, pad, pluck)
**Generates**: Synth preset configurations

**Use Cases:**
- "Make a warm analog bass sound"
- "Create a bright lead synth with portamento"
- "Design a plucky pad with filter sweep"
- "Generate a dark, evolving pad sound"

**Input**: Sound descriptions
**Output**: Synth parameter mappings (oscillator waveforms, filter cutoff, ADSR settings, etc.)

**Coordination**: Works with DAW Agent to configure instrument parameters

---

### 7. **Performance Agent** 🎵
**Purpose**: Humanization, timing feel, velocity curves, groove
**Knows**: Timing variations, velocity patterns, swing, groove templates
**Generates**: Timing and velocity adjustments

**Use Cases:**
- "Add human feel to the drums"
- "Make the piano sound more expressive"
- "Add swing to the hi-hats"
- "Humanize the quantized MIDI"
- "Add velocity curves that follow the dynamics"

**Input**: Performance requests, MIDI data
**Output**: Timing offsets, velocity adjustments, groove templates

**Coordination**: Works with DAW Agent to adjust MIDI timing/velocity

---


---

### 10. **Rhythm Agent** 🥁
**Purpose**: Groove, drum patterns, quantization, rhythm programming
**Knows**: Drum patterns, rhythm styles, groove templates, quantization grids
**Generates**: Drum patterns and rhythm sequences

**Use Cases:**
- "Add a 4/4 kick pattern"
- "Create a syncopated hi-hat pattern"
- "Add a swing groove to everything"
- "Generate a trap drum pattern"
- "Quantize to 16th notes with 70% strength"

**Input**: Rhythm descriptions, style requests
**Output**: MIDI patterns (drum sequences), quantization settings, groove templates

**Coordination**: Works with DAW Agent to place MIDI, with Performance Agent for feel

---

### 11. **Structure Agent** 📐
**Purpose**: Song arrangement, section placement, form
**Knows**: Song forms (verse-chorus, ABAB, etc.), section types, arrangement patterns
**Generates**: Structure markers and arrangement suggestions

**Use Cases:**
- "Arrange this as verse-chorus-verse-chorus-bridge-chorus"
- "Double the chorus at the end"
- "Add an 8-bar intro before the verse"
- "Repeat the bridge twice"
- "Fade out the last 4 bars"

**Input**: Structure requests, existing sections
**Output**: Section markers, arrangement instructions

**Coordination**: Works with DAW Agent to place markers, with Automation Agent for transitions

---

### 12. **Harmony Agent** 🎼
**Purpose**: Chord progressions, voice leading, harmonic analysis
**Knows**: Harmony theory, voice leading rules, chord functions, substitutions
**Generates**: Chord progressions with proper voice leading

**Use Cases:**
- "Create a jazz progression with smooth voice leading"
- "Add chord extensions to this progression"
- "Substitute dominant chords for more color"
- "Create a modal progression in Dorian"
- "Analyze the harmony of track 1"

**Note**: Might overlap with Arranger Agent - could be a specialized subset

---

### 13. **Lyrics/Vocal Agent** 🎤
**Purpose**: Lyrics placement, vocal phrasing, melody-to-lyrics alignment
**Knows**: Syllable mapping, phrasing, vocal ranges, lyric timing
**Generates**: Lyrics placement and vocal part structure

**Use Cases:**
- "Place these lyrics on the melody line"
- "Align lyrics with the rhythm"
- "Generate a vocal harmony part"
- "Create backing vocal arrangements"

**Coordination**: Works with Arranger Agent for melody, with DAW Agent for track placement

---

### 14. **Sample/Asset Agent** 📦
**Purpose**: Sample management, loop finding, asset loading
**Knows**: Sample libraries, loop categories, BPM matching, key matching
**Generates**: Sample selection and placement instructions

**Use Cases:**
- "Find a kick drum sample"
- "Load a synth pad loop at 120 BPM"
- "Find loops in key of C major"
- "Replace this kick with a punchier one"

**Coordination**: Works with DAW Agent to load samples into clips

---

### 15. **Reference Agent** 🎧
**Purpose**: Compare with reference tracks, match characteristics
**Knows**: Audio comparison, frequency matching, loudness matching
**Generates**: Analysis and matching recommendations

**Use Cases:**
- "Match the loudness of this reference track"
- "Analyze the frequency balance of the reference"
- "Match the stereo width"
- "Compare the mix to this track"

**Coordination**: Works with Mix/Analysis Agent (could be integrated as a mode)

---

## Agent Priority Ranking

### High Priority (Most Useful)
1. **Mix/Analysis Agent** - Critical for production quality (unified mixing/mastering/analysis)
2. **Sound Design Agent** - Essential for electronic music
3. **Performance Agent** - Makes MIDI sound more musical

### Medium Priority
5. **Rhythm Agent** - Useful for drum programming
6. **Structure Agent** - Helps with arrangement

### Lower Priority (Nice to Have)
7. **Harmony Agent** - Overlaps with Arranger Agent
8. **Lyrics/Vocal Agent** - Only needed if working with vocals
9. **Sample/Asset Agent** - Depends on workflow
10. **Reference Agent** - Advanced mixing feature (could integrate with Mix/Analysis Agent)

## Agent Coordination Patterns

### Simple Chain
```
User Request → DAW Agent → Actions
```

### Musical Content Chain
```
User Request → DAW Agent (with placeholder) → Arranger Agent → Inject Notes
```

### Complex Multi-Agent
```
User Request
  → Analysis Agent (detect key/tempo)
  → DAW Agent (create structure)
  → Arranger Agent (generate chords)
  → Sound Design Agent (configure synth)
  → Mix Agent (add FX)
  → Performance Agent (humanize)
  → Automation Agent (add dynamics)
```

## Implementation Considerations

### Agent Interface Standardization
All agents should follow a consistent pattern:
```go
type Agent interface {
    Generate(ctx context.Context, request AgentRequest) (*AgentResult, error)
}
```

### Placeholder System
Agents that need coordination should use placeholders:
- `<PLACEHOLDER_MUSICAL_CONTENT>` - Arranger Agent
- `<PLACEHOLDER_AUTOMATION_CURVE>` - Automation Agent
- `<PLACEHOLDER_FX_CHAIN>` - Mix Agent
- `<PLACEHOLDER_SYNTH_PARAMS>` - Sound Design Agent
- etc.

### Context Sharing
Agents may need to share context:
- Project state (BPM, time signature, key)
- Track information (what's on each track)
- Analysis results (frequency content, structure)

## Next Steps

1. Prioritize which agents to implement first (likely Mix Agent and Sound Design Agent)
2. Design agent interfaces and coordination patterns
3. Implement placeholder system for agent coordination
4. Build orchestrator to coordinate multiple agents
