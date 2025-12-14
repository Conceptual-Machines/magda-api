# MAGDA Future Enhancements

This document captures all future enhancement ideas discussed for the MAGDA REAPER extension.

## Table of Contents

1. [MIDI Generation Agent](#midi-generation-agent)
2. [Query/Filter Operations](#queryfilter-operations)
3. [Mix/Master Analysis](#mixmaster-analysis)
4. [State/API Access Improvements](#stateapi-access-improvements)
5. [UI Improvements](#ui-improvements)
6. [Sample Selection in Chat](#sample-selection-in-chat)
7. [Track + Sampler Workflow](#track--sampler-workflow)
8. [Plugin Autocomplete & Preferences](#plugin-autocomplete--preferences)
9. [LLM-Generated Plugin Aliases](#llm-generated-plugin-aliases)
10. [Plugin Analysis Menu](#plugin-analysis-menu)
11. [Project Housekeeping Routines](#project-housekeeping-routines)
12. [Housekeeping UI Integration](#housekeeping-ui-integration)
13. [Review & Approve Workflow](#review--approve-workflow)

---

## MIDI Generation Agent

### Concept
Separate agent for generating MIDI sequences (chord progressions, patterns, melodies) that works in parallel with the main MAGDA agent.

### Architecture: Parallel Agent Calls

**Flow:**
```
User: "add clip at bar 4 for 8 bars with I VI IV chord progression"
     ↓
Main Agent (MAGDA) → Identifies: needs MIDI generation
     ↓
     ├─→ MIDI Agent (parallel): Generates notes for I VI IV progression
     └─→ MAGDA Agent (parallel): Generates track/clip actions
     ↓
     Merge: track(1).newClip(bar=4, length_bars=8).addMidi(notes=[...])
```

### Implementation
- Create `/api/v1/midi/generate` endpoint
- MAGDA agent detects musical requirements
- Spawn both requests in parallel
- Merge results: MIDI notes → DSL `.addMidi(notes=[...])`

### Benefits
- Faster (parallel execution)
- Separation of concerns
- Reusable MIDI agent
- Can cache MIDI patterns

---

## Query/Filter Operations

### Concept
Operations that require querying/filtering items in REAPER, then applying actions to the filtered set.

### Examples
- "Select all clips longer than 4 bars"
- "Mute all tracks with 'Drums' in the name"
- "Delete all clips before bar 8"
- "Apply reverb to all tracks with Serum"

### Approach: Function Calling + DSL

**Function Tools:**
```go
tools := []Tool{
    {
        Name: "query_tracks",
        Description: "Find tracks matching criteria",
        Parameters: {
            "name_contains": "track name contains",
            "has_fx": "has specific FX",
            "is_muted": "muted state",
        }
    },
    {
        Name: "query_clips",
        Description: "Find clips matching criteria",
        Parameters: {
            "track_id": "filter by track",
            "min_length_bars": "minimum length",
            "before_bar": "starts before this bar",
        }
    }
}
```

**DSL Extensions:**
- Support batch operations: `tracks([1, 5, 7]).mute()`
- Query syntax: `tracks(name_contains="Drums").mute()`

### Benefits
- Flexible: function calling for complex queries, DSL for simple cases
- Efficient: can batch operations
- Extensible: easy to add new query criteria

---

## Mix/Master Analysis

### Concept
Multi-step workflow: bounce audio → DSP analysis → LLM suggestions → actions

### Workflow
1. Create bounce track
2. Bounce master/selected tracks to audio
3. Run DSP analysis (spectrum, dynamics, stereo, etc.)
4. Send analysis + context to LLM
5. LLM suggests tweaks
6. Convert suggestions to actions

### DSL Method
```
analyzeMix()
analyzeMaster()
analyzeTrack(track=1)
```

### DSP Analysis Using JSFX

**Approach:**
- Use REAPER's built-in JSFX analysis plugins
- `JS: Analysis/spectrograph` - Spectrum analysis
- `JS: Analysis/stereo field` - Stereo width/correlation
- `JS: Analysis/loudness` - Loudness analysis
- Or create custom JSFX for specific metrics

**Implementation:**
```cpp
// Add JSFX to track
int fx = TrackFX_AddByName(track, "JS: Analysis/spectrograph", false, -1);

// Read analysis parameters
double rms = TrackFX_GetParam(track, fx, 0, &min, &max);
double peak = TrackFX_GetParam(track, fx, 1, &min, &max);
```

### Benefits
- No custom DSP code needed
- Real-time analysis
- Easy to extend with custom JSFX
- Standard API

---

## State/API Access Improvements

### Problem
Sometimes need data from REAPER that's not in the state snapshot (e.g., FX chains, clip contents, automation).

### Solution: Hybrid Approach

**State Snapshot (always sent):**
- Basic track info
- Selected tracks
- Play state
- Common metadata

**Function Tools (on-demand):**
- `get_track_fx_chain(track_id)`
- `get_clip_midi_data(clip_id)`
- `get_project_tempo()`
- `get_track_sends(track_id)`

### Benefits
- Fast path for common operations (state snapshot)
- Flexibility for deep queries (function calling)
- No external dependencies
- Secure (local only)

---

## UI Improvements

### Third Panel: Context Panel

**Purpose:** Display current REAPER state and context - read-only information

**Content:**
- **Selected Tracks:** Show all selected tracks (handles multiple selection)
- **Current Position/Time:** Bar, beat, BPM, time signature
- **Active Context:** Playing, recording, time selection
- **Recent Actions:** What was just done
- **Track Summary:** Quick overview (optional, collapsible)

**Layout:**
```
┌─────────────┬─────────────┬─────────────┐
│   Chat      │  Actions    │  Context    │
│   Input     │  Display    │  Panel     │
│             │             │             │
│  [input]    │  Actions:   │ Selected:   │
│  History    │  - create   │ [1] Drums   │
│             │  - add FX   │ [3] Bass    │
│             │             │             │
│             │             │ Position:   │
│             │             │ Bar 4.1     │
│             │             │ BPM: 120    │
└─────────────┴─────────────┴─────────────┘
```

**Additional Use Cases:**
- Mix analysis results display
- MIDI generation status
- Query results preview
- FX chain display
- Error context
- Action queue/progress
- Project statistics
- Track relationships
- Automation state
- Recent commands history

**Smart Context Switching:**
- Normal chat → Show selected tracks + position
- Analysis running → Show analysis progress + results
- Query executed → Show query results
- Error occurred → Show error context
- MIDI generation → Show MIDI generation status

---

## Sample Selection in Chat

### Concept
Allow users to select samples directly from the chat interface when creating tracks with samples.

### Approaches

**Option 1: Interactive Chat Messages**
- LLM detects sample needed → Shows interactive element
- `[Browse Samples...]` button
- Or user types sample name → Auto-completes

**Option 2: Inline Sample Picker**
- When user mentions sample, show inline picker
- Display available samples as buttons
- User clicks to select

**Option 3: Autocomplete/Suggestions**
- Real-time autocomplete in chat input
- Search project folder + sample library
- Show matching samples in dropdown

**Option 4: File Picker Dialog**
- Use OS native file picker
- Cross-platform (SWELL on Mac/Linux, Win32 on Windows)
- Filter by audio file types

**Option 5: Drag & Drop**
- User drags sample file into chat
- System parses and uses it

### Recommended: Hybrid Approach
- **File picker** (easy, works immediately)
- **Autocomplete** (medium, great UX)
- **Inline picker** (medium, nice polish)

### Sample Mapping Configuration

**Problem:** User says "kick" → System needs `kick3.wav` or `kick_ninja.wav`

**Solution:**
```json
// .magda-samples.json (project-level)
{
  "drum_samples": {
    "kick": ["kick3.wav", "kick_ninja.wav", "kick_808.wav"],
    "snare": ["snare1.wav", "snare_crack.wav"]
  },
  "sample_paths": {
    "kick": "/Samples/Drums/Kick/"
  },
  "default_selection": "first"
}
```

**Or user-level config:**
```json
// ~/.magda/sample-mapping.json
{
  "kick": {
    "preferred": "kick_ninja.wav",
    "alternatives": ["kick3.wav", "kick_808.wav"],
    "path": "~/Samples/Drums/"
  }
}
```

---

## Track + Sampler Workflow

### Concept
When creating a track with a sample, automatically add a sampler and load the sample.

### DSL Syntax
```
track(name="Kick").addSample("kick_ninja.wav")
```

**What happens:**
1. Creates track "Kick"
2. Adds ReaSamplomatic5000 (or configured sampler)
3. Loads `kick_ninja.wav` into sampler slot 1
4. Track ready for MIDI triggering

### Implementation
- Default to ReaSamplomatic5000 for single samples
- Support custom samplers via config
- Load sample into sampler via REAPER API
- Map samples to MIDI notes

### Configuration
```json
{
  "default_sampler": "JS: ReaSamplomatic5000",
  "sampler_preferences": {
    "single_sample": "JS: ReaSamplomatic5000",
    "multi_sample": "VST: Kontakt",
    "drum_kit": "VST: Battery"
  }
}
```

---

## Plugin Autocomplete & Preferences

### Concept
Autocomplete for plugins in chat interface + configuration for plugin preferences (VST3 over VST, newer versions, etc.)

### Plugin Discovery
- Scan REAPER's plugin list using API
- `CountInstalledFX()`, `GetInstalledFX()`
- Cache plugin database

### Autocomplete Implementation
```
User types: "add Serum"
            ↓
Autocomplete shows:
  - VST3: Serum (Xfer Records) ⭐
  - VST: Serum (Xfer Records)

User selects: VST3: Serum
```

### Configuration
```json
{
  "format_preferences": {
    "order": ["VST3", "VST", "AU", "JS"],
    "prefer_newer": true
  },
  "plugin_aliases": {
    "serum": "VST3: Serum (Xfer Records)",
    "kontakt": "VST3: Kontakt (Native Instruments)",
    "reaeq": "JS: ReaEQ"
  },
  "plugin_overrides": {
    "Serum": {
      "preferred_format": "VST3",
      "preferred_version": "latest"
    }
  }
}
```

### Plugin Selection Logic
1. User types: "add Serum"
2. Search plugins matching "Serum"
3. Filter by preferences (VST3 > VST, newer > older)
4. Sort by preference score
5. Show top matches in autocomplete

---

## LLM-Generated Plugin Aliases

### Concept
Use LLM to automatically generate smart aliases when scanning plugins, instead of manual configuration.

### Workflow
1. Extension scans plugins
2. Sends list to API: "Generate aliases for these plugins"
3. LLM generates aliases
4. Saves to config file
5. Ready to use

### LLM Prompt
```
Analyze plugin list and generate common aliases/short names.
Consider:
- Short names (Serum → "serum")
- Manufacturer variations (Xfer Records → "xfer serum")
- Common abbreviations (ReaEQ → "reaeq", "rea-eq", "eq")
- Version numbers (Kontakt 7 → "kontakt", "kontakt7")
- Category names (synth, eq, compressor)
```

### Example Output
```json
{
  "serum": "VST3: Serum (Xfer Records)",
  "xfer serum": "VST3: Serum (Xfer Records)",
  "reaeq": "JS: ReaEQ",
  "rea-eq": "JS: ReaEQ",
  "eq": "JS: ReaEQ",
  "kontakt": "VST3: Kontakt (Native Instruments)",
  "kontakt7": "VST3: Kontakt (Native Instruments)"
}
```

### Benefits
- Zero configuration
- Smart aliases (LLM understands naming patterns)
- Handles variations
- Context-aware
- Self-improving (can learn from usage)

---

## Plugin Analysis Menu

### Menu Structure
```
Extensions → MAGDA
  ├─ Open MAGDA
  ├─ Login
  ├─ Settings
  ├─ About
  └─ Analyze Plugins
      ├─ Find Duplicates
      ├─ Find Old Versions
      ├─ Generate Aliases
      └─ Plugin Report
```

### Features

**1. Find Duplicate Plugins**
- Finds same plugin in multiple formats (VST, VST3, AU)
- Identifies version conflicts
- Shows which to keep/remove
- Recommends based on preferences

**2. Find Old Versions**
- Identifies outdated plugin versions
- Compares version numbers
- Suggests updates

**3. Generate Aliases** (Manual Trigger)
- Re-scans plugins
- Generates new aliases via LLM
- Updates alias config

**4. Plugin Report**
- Full analysis of plugin library
- Statistics and recommendations
- Export report

### Implementation
- Duplicate detection algorithm
- Version comparison
- Preference-based recommendations
- LLM integration for complex cases

---

## Project Housekeeping Routines

### Concept
Automated maintenance tasks for REAPER projects: renaming tracks, color coding, organization, cleanup.

### Features

**1. Smart Track Renaming**
- Rename based on content (instrument, sample, pattern)
- Apply naming conventions (e.g., "01_Kick", "02_Snare")
- Fix inconsistent naming
- LLM-powered suggestions

**2. Color Coding**
- By type (Drums: Red, Bass: Blue, Melody: Green)
- By config rules (instrument-based, name pattern, track number)
- Visual organization

**3. Track Organization**
- Group related tracks (drums, bass, melody)
- Create folder tracks
- Organize by type/category
- Sort by various criteria

**4. Cleanup Tasks**
- Remove empty tracks
- Remove unused FX
- Clean up empty folders
- Remove duplicate clips
- Optimize project

**5. Standardization**
- Naming conventions
- Folder structure
- FX chain order
- Volume/panning rules

### Configuration
```json
{
  "housekeeping": {
    "auto_rename": {
      "enabled": true,
      "convention": "##_Name",
      "use_llm": true,
      "rules": {
        "by_instrument": true,
        "by_sample": true,
        "by_pattern": true
      }
    },
    "auto_color": {
      "enabled": true,
      "rules": {
        "by_instrument": {
          "Serum": "#FF6B6B",
          "Kontakt": "#4ECDC4"
        },
        "by_name_pattern": {
          "kick|kick": "#FF0000",
          "snare|snr": "#00FF00"
        }
      }
    },
    "organization": {
      "enabled": true,
      "create_folders": true,
      "folder_structure": "by_category",
      "sort_order": "by_category"
    },
    "cleanup": {
      "enabled": true,
      "remove_empty_tracks": true,
      "remove_unused_fx": false
    }
  }
}
```

### Additional Routines
- Track numbering (auto-number tracks)
- Send organization (standardize send levels/routing)
- FX chain templates (apply standard FX chains by type)
- Volume normalization (set consistent track volumes)
- Panning rules (auto-pan by track type)
- Marker creation (auto-create markers for sections)
- Time signature detection
- Tempo mapping

---

## Housekeeping UI Integration

### Recommended: Multi-Method Approach

**1. Chat Commands (Primary)**
- Natural language: "rename tracks", "color by type"
- Most flexible and discoverable
- Fits MAGDA's chat-first design

**2. Quick Access Button (Secondary)**
- Top bar dropdown for quick access
- Visual reminder
- Fast for power users

**3. Context Panel (Tertiary)**
- Show housekeeping suggestions
- "3 tracks need renaming" → Click to fix
- Context-aware recommendations

### UI Design

**Top Bar Button:**
```
┌─────────────────────────────────────────────┐
│ MAGDA Chat                    [⚙ Settings]  │
│                              [🧹 Housekeep ▼]│
├─────────────────────────────────────────────┤
│ Chat:                                       │
│ "add kick track"                            │
│ [Send]                                      │
└─────────────────────────────────────────────┘

Dropdown Menu:
  🏷️  Rename Tracks
  🎨 Color Tracks
  📁 Organize Tracks
  🧹 Cleanup Project
  ────────────────
  ⚙️  Configure Housekeeping
```

**Context Panel Integration:**
```
Context Panel:
  Housekeeping:
    → 3 tracks need renaming
    → 5 tracks need coloring
    → 2 empty tracks found

  [Run Housekeeping] button
```

### User Experience Flows

**Flow 1: Chat Command**
```
User: "rename all tracks"
     ↓
MAGDA: Analyzing tracks...
       Suggested names:
       - Track 1 → "Kick"
       - Track 2 → "Snare"
       [Preview] [Apply] [Cancel]
     ↓
User: [Apply]
     ↓
MAGDA: ✓ Renamed 2 tracks
```

**Flow 2: Quick Button**
```
User: Clicks [Housekeeping ▼] → "Rename Tracks"
     ↓
Shows preview dialog
     ↓
User: [Apply]
     ↓
Done
```

**Flow 3: Context Panel Suggestion**
```
Context Panel shows:
  "3 tracks need renaming"
  [Rename Now] button
     ↓
User: Clicks [Rename Now]
     ↓
Shows preview → User approves
```

---

## User Workflow Examples

### 1. Drum Pattern Creation
```
"add a kick track with a 4/4 beat for 64 bars from bar 32"
Steps:
1. Create track (or find existing "kick" track)
2. Add clip at bar 32, 64 bars long
3. Generate MIDI pattern (4/4 kick pattern)
4. Map "kick" → actual sample (kick3.wav, etc.)
5. Apply sample to clip
```

### 2. Full Drum Kit Setup
```
"create a full drum kit with kick, snare, hihat, and crash"
Steps:
1. Create 4 tracks
2. Add clips to each
3. Generate MIDI patterns
4. Map to appropriate samples
5. Route to drum bus (optional)
```

### 3. Bass Line Creation
```
"add a bass line following the kick pattern for 32 bars"
Steps:
1. Find/create bass track
2. Analyze kick pattern
3. Generate complementary bass line
4. Add clip with MIDI
5. Apply bass instrument
```

### 4. Chord Progression with Melody
```
"create a piano track with I-VI-IV-V progression and add a melody on top"
Steps:
1. Create piano track
2. Generate chord progression
3. Add clip with chord MIDI
4. Create melody track
5. Generate melody that follows chords
```

### 5. Sample-Based Loop Creation
```
"add a breakbeat loop from bar 16, 4 bars long"
Steps:
1. Find/create track
2. Add clip at bar 16, 4 bars
3. Map "breakbeat" → sample
4. Apply sample to clip
5. Time-stretch if needed
```

### 6. Layered Instrumentation
```
"add a pad layer with strings and add reverb"
Steps:
1. Create track
2. Add instrument (strings)
3. Generate pad MIDI pattern
4. Add clip with MIDI
5. Add reverb FX
6. Set reverb parameters
```

### 7. Variation Creation
```
"create a variation of the main loop at bar 17, make it more intense"
Steps:
1. Find "main loop" clip
2. Copy to bar 17
3. Analyze original
4. Generate variation (higher velocity, more notes)
5. Apply variation
```

### 8. Fill Creation
```
"add a drum fill at bar 31, 1 bar long"
Steps:
1. Find drum tracks
2. Add clips at bar 31, 1 bar
3. Generate fill pattern
4. Map to appropriate samples
5. Apply to clips
```

---

## Review & Approve Workflow

### Concept
A safety and control mechanism where the LLM suggests changes, the user reviews them, and then explicitly approves or rejects before changes are applied.

### Workflow Pattern

**Standard Flow:**
```
1. User requests action
2. LLM generates suggestions/changes
3. System shows preview to user
4. User reviews and accepts/rejects
5. If accepted → Apply changes
6. If rejected → User can modify or cancel
```

### Use Cases

**1. Mix Analysis Suggestions**
```
User: "analyze the mix and suggest improvements"
     ↓
LLM: Analyzes mix → Generates suggestions
     ↓
Preview shows:
  Suggested Changes:
  1. Add EQ to master: high-pass at 30Hz
  2. Add compressor: ratio 4:1, threshold -12dB
  3. Reduce 2kHz by 2dB on track 3

  [Accept All] [Accept Selected] [Reject] [Modify]
     ↓
User: Selects changes 1 and 2, rejects 3
     ↓
System: Applies only accepted changes
```

**2. Track Renaming**
```
User: "rename all tracks"
     ↓
LLM: Analyzes tracks → Suggests names
     ↓
Preview shows:
  Track Renaming:
  Track 1: "Track 1" → "Kick" ✓
  Track 2: "Track 2" → "Snare" ✓
  Track 3: "serum" → "Serum Lead" ✓

  [Accept All] [Accept Selected] [Reject] [Edit]
     ↓
User: Edits "Serum Lead" → "Serum Pad", accepts
     ↓
System: Applies renaming
```

**3. Housekeeping Operations**
```
User: "organize project"
     ↓
LLM: Analyzes project → Suggests organization
     ↓
Preview shows:
  Organization Plan:
  - Create "Drums" folder
    → Move tracks: Kick, Snare, HiHat
  - Create "Bass" folder
    → Move tracks: Bass, Sub
  - Create "Melody" folder
    → Move tracks: Lead, Pad

  [Preview Structure] [Accept] [Reject] [Modify]
     ↓
User: Reviews structure, accepts
     ↓
System: Creates folders and moves tracks
```

**4. Plugin Cleanup**
```
User: "find duplicate plugins"
     ↓
System: Analyzes plugins → Finds duplicates
     ↓
Preview shows:
  Duplicate Plugins:
  Serum:
    ✓ Keep: VST3: Serum (Xfer Records)
    ✗ Remove: VST: Serum (Xfer Records)

  Kontakt:
    ✓ Keep: VST3: Kontakt 7 (NI)
    ✗ Remove: VST: Kontakt 5 (NI)
    ✗ Remove: VST3: Kontakt 6 (NI)

  [Remove Selected] [Keep All] [Cancel]
     ↓
User: Reviews, clicks "Remove Selected"
     ↓
System: Removes only selected duplicates
```

### UI Implementation

**Preview Dialog/Window:**
```
┌─────────────────────────────────────┐
│ Suggested Changes                   │
├─────────────────────────────────────┤
│                                     │
│ Mix Analysis Suggestions:           │
│                                     │
│ ☑ Add EQ to master                 │
│    High-pass at 30Hz                │
│                                     │
│ ☑ Add compressor                   │
│    Ratio 4:1, threshold -12dB      │
│                                     │
│ ☐ Reduce 2kHz by 2dB on track 3    │
│    (You rejected this)              │
│                                     │
│ [Accept Selected] [Accept All]      │
│ [Reject] [Modify]                   │
└─────────────────────────────────────┘
```

**In Chat Interface:**
```
MAGDA: I've analyzed your mix. Here are my suggestions:

  1. Add EQ to master: high-pass at 30Hz
  2. Add compressor: ratio 4:1, threshold -12dB
  3. Reduce 2kHz by 2dB on track 3

  [Preview Changes] [Accept All] [Accept Selected] [Reject]
```

**Context Panel Integration:**
```
Context Panel:
  Pending Changes:
    → 3 mix improvements suggested
    → 5 tracks ready to rename
    → 2 duplicate plugins found

  [Review Changes] button
```

### Implementation

**Action Queue System:**
```cpp
// In REAPER extension
class ActionQueue {
public:
    struct PendingAction {
        std::string id;
        std::string description;
        std::vector<Action> actions;  // DSL actions to execute
        bool approved;
    };

    void AddSuggestion(const PendingAction& action);
    void ShowPreview(const std::vector<PendingAction>& actions);
    void ApplyApproved(const std::vector<std::string>& action_ids);
    void Reject(const std::vector<std::string>& action_ids);
};
```

**API Response Format:**
```go
// In API
type SuggestionResponse struct {
    Suggestions []Suggestion `json:"suggestions"`
    Preview     string       `json:"preview"`  // Human-readable preview
}

type Suggestion struct {
    ID          string                 `json:"id"`
    Description string                 `json:"description"`
    Actions     []map[string]interface{} `json:"actions"`  // DSL actions
    Confidence  float64                `json:"confidence"`
    Reasoning   string                  `json:"reasoning"`  // Why this suggestion
}
```

**Chat Integration:**
```cpp
// In chat window
void OnSuggestionReceived(const SuggestionResponse& response) {
    // Show suggestions in chat
    DisplaySuggestions(response);

    // Show preview button
    ShowPreviewButton();

    // Store actions for later execution
    m_pendingActions = response.Suggestions;
}

void OnUserApproves(const std::vector<std::string>& suggestion_ids) {
    // Get approved actions
    auto actions = GetApprovedActions(suggestion_ids);

    // Execute actions
    ExecuteActions(actions);

    // Show confirmation
    ShowConfirmation("Applied " + std::to_string(actions.size()) + " changes");
}
```

### Features

**1. Selective Approval**
- User can approve/reject individual suggestions
- Checkbox for each suggestion
- "Accept All" / "Reject All" buttons

**2. Modification**
- User can edit suggestions before applying
- "Modify" button opens editor
- Changes tracked and re-previewed

**3. Preview**
- Show what will change before applying
- Visual diff for track renaming
- Structure preview for organization
- Before/after comparison

**4. Undo Support**
- All approved changes are undo-able
- Group changes for single undo
- Clear undo history

**5. Confidence Indicators**
- Show LLM confidence for each suggestion
- Highlight high-confidence suggestions
- Warn about low-confidence suggestions

**6. Reasoning Display**
- Show why LLM made each suggestion
- "Because: Track has too much low end"
- Helps user understand and decide

### Configuration

```json
{
  "review_workflow": {
    "require_approval": true,  // Always require approval
    "auto_approve_high_confidence": false,  // Auto-approve if confidence > 0.9
    "show_reasoning": true,  // Show LLM reasoning
    "show_preview": true,  // Show preview before applying
    "group_similar_changes": true,  // Group related changes
    "undo_support": true  // Enable undo for approved changes
  }
}
```

### Benefits

1. **Safety** - User has control over all changes
2. **Transparency** - See what will happen before it happens
3. **Learning** - Understand LLM reasoning
4. **Flexibility** - Modify suggestions before applying
5. **Confidence** - Only apply changes user approves

### Example Flows

**Flow 1: Mix Analysis**
```
User: "analyze mix"
     ↓
LLM: Analyzes → 5 suggestions
     ↓
Preview: Shows 5 suggestions with checkboxes
     ↓
User: Selects 3, rejects 2
     ↓
System: Applies 3 approved changes
     ↓
Confirmation: "Applied 3 mix improvements"
```

**Flow 2: Batch Renaming**
```
User: "rename tracks"
     ↓
LLM: Analyzes → Suggests names for 10 tracks
     ↓
Preview: Table showing old → new names
     ↓
User: Edits 2 names, accepts all
     ↓
System: Renames all tracks (with edits)
```

**Flow 3: Organization**
```
User: "organize project"
     ↓
LLM: Analyzes → Suggests folder structure
     ↓
Preview: Tree view of proposed structure
     ↓
User: Modifies folder names, accepts
     ↓
System: Creates folders and moves tracks
```

---

## Implementation Priority

### Phase 1 (High Priority)
- [ ] Context Panel (selected tracks, position, recent actions)
- [ ] Sample selection in chat (file picker + autocomplete)
- [ ] Plugin autocomplete (basic discovery + fuzzy search)
- [ ] Track + sampler workflow (ReaSamplomatic5000)
- [ ] Basic housekeeping (rename, color, organize)

### Phase 2 (Medium Priority)
- [ ] MIDI generation agent (parallel calls)
- [ ] Query/filter operations (function calling)
- [ ] Mix/master analysis (JSFX-based)
- [ ] LLM-generated plugin aliases
- [ ] Plugin analysis menu (duplicates, old versions)

### Phase 3 (Low Priority)
- [ ] Advanced housekeeping (cleanup, standardization)
- [ ] Sample mapping configuration
- [ ] Plugin preference learning
- [ ] Advanced UI features (drag & drop, previews)

---

## Notes

- All features should maintain backward compatibility
- Configuration should be user-friendly and well-documented
- LLM integration should be optional where possible (fallback to rules)
- UI should be consistent with REAPER's design language
- Performance should be considered (caching, lazy loading)
- Error handling and user feedback are critical

---

*Last updated: 2024-01-15*
