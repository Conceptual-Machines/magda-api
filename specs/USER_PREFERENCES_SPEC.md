# User Preferences & Autocomplete Specification

## Overview

System for managing user preferences, plugin aliases, and drum kit mappings. Enables `@mention` autocomplete in the Reaper extension and context-aware LLM responses.

---

## 1. Database Schema

### 1.1 Plugin Aliases

```sql
CREATE TABLE plugin_aliases (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,  -- NULL = global/preset
    alias VARCHAR(64) NOT NULL,           -- "addictivedrums" (without @)
    plugin_name VARCHAR(255) NOT NULL,    -- "Addictive Drums 2"
    plugin_type VARCHAR(32) NOT NULL,     -- "drums", "synth", "fx", "sampler"
    icon VARCHAR(8),                      -- "🥁"
    mapping_id VARCHAR(64),               -- FK to drum_mappings (for drums only)
    is_favorite BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(user_id, alias),  -- User can't have duplicate aliases
    INDEX idx_alias_prefix (alias)  -- For prefix search
);
```

### 1.2 Drum Mappings

```sql
CREATE TABLE drum_mappings (
    id VARCHAR(64) PRIMARY KEY,           -- "addictive_drums_v2"
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,  -- NULL = preset
    name VARCHAR(255) NOT NULL,           -- "Addictive Drums 2"
    notes JSONB NOT NULL,                 -- {"kick": 36, "snare": 38, ...}
    is_preset BOOLEAN DEFAULT FALSE,      -- TRUE for built-in mappings
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### 1.3 User Preferences

```sql
CREATE TABLE user_preferences (
    id SERIAL PRIMARY KEY,
    user_id INTEGER UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    default_drum_kit VARCHAR(64) REFERENCES drum_mappings(id),
    default_bpm INTEGER DEFAULT 120,
    default_time_signature VARCHAR(8) DEFAULT '4/4',
    theme VARCHAR(32) DEFAULT 'dark',
    preferences JSONB DEFAULT '{}',       -- Extensible key-value store
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

---

## 2. API Endpoints

### 2.1 Autocomplete

```
GET /api/v1/autocomplete

Query params:
  - prefix: string (e.g., "addi") - required
  - type: "plugin" | "drumkit" | "all" (default: "all")
  - limit: int (default: 10, max: 50)

Response:
{
  "suggestions": [
    {
      "alias": "addictivedrums",
      "display_name": "Addictive Drums 2",
      "type": "drums",
      "icon": "🥁",
      "mapping_id": "addictive_drums_v2",
      "is_favorite": true
    },
    {
      "alias": "addictivekeys",
      "display_name": "Addictive Keys",
      "type": "synth",
      "icon": "🎹",
      "mapping_id": null,
      "is_favorite": false
    }
  ]
}

Notes:
- Returns global presets + user's custom aliases
- User's favorites appear first
- Fuzzy match on alias and display_name
- Case-insensitive
```

### 2.2 Plugin Aliases Management

```
GET /api/v1/aliases
Response: { "aliases": [...] }

POST /api/v1/aliases
Body: {
  "alias": "mysynth",
  "plugin_name": "Serum",
  "plugin_type": "synth",
  "icon": "🎹"
}
Response: { "id": 123, "alias": "mysynth", ... }

PUT /api/v1/aliases/:id
Body: { "alias": "mysynth2", ... }

DELETE /api/v1/aliases/:id

POST /api/v1/aliases/:id/favorite
Body: { "is_favorite": true }
```

### 2.3 Drum Mappings

```
GET /api/v1/mappings
Query: ?include_presets=true (default true)
Response: {
  "mappings": [
    {
      "id": "addictive_drums_v2",
      "name": "Addictive Drums 2",
      "is_preset": true,
      "notes": { "kick": 36, "snare": 38, ... }
    },
    {
      "id": "my_custom_kit",
      "name": "My Custom Kit",
      "is_preset": false,
      "notes": { ... }
    }
  ]
}

GET /api/v1/mappings/:id
Response: { "id": "...", "name": "...", "notes": {...} }

POST /api/v1/mappings
Body: {
  "id": "my_kit",  // optional, auto-generated if omitted
  "name": "My Custom Kit",
  "notes": {
    "kick": 36,
    "snare": 40,
    ...
  }
}

PUT /api/v1/mappings/:id
Body: { "name": "...", "notes": {...} }

DELETE /api/v1/mappings/:id
(Only user's own mappings, not presets)
```

### 2.4 User Preferences

```
GET /api/v1/preferences
Response: {
  "default_drum_kit": "addictive_drums_v2",
  "default_bpm": 120,
  "default_time_signature": "4/4",
  "theme": "dark",
  "custom": { ... }  // extensible
}

PUT /api/v1/preferences
Body: {
  "default_drum_kit": "superior_drummer_v3",
  "default_bpm": 95
}
Response: { updated preferences }

PATCH /api/v1/preferences
Body: { "default_bpm": 110 }  // Partial update
```

---

## 3. Preset Data (Seed)

### 3.1 Preset Plugin Aliases

```json
[
  {"alias": "addictivedrums", "plugin_name": "Addictive Drums 2", "type": "drums", "icon": "🥁", "mapping_id": "addictive_drums_v2"},
  {"alias": "superior", "plugin_name": "Superior Drummer 3", "type": "drums", "icon": "🥁", "mapping_id": "superior_drummer_v3"},
  {"alias": "ezdrummer", "plugin_name": "EZDrummer 3", "type": "drums", "icon": "🥁", "mapping_id": "ezdrummer_v3"},
  {"alias": "battery", "plugin_name": "Battery 4", "type": "drums", "icon": "🥁", "mapping_id": "battery_v4"},
  {"alias": "serum", "plugin_name": "Serum", "type": "synth", "icon": "🎹"},
  {"alias": "vital", "plugin_name": "Vital", "type": "synth", "icon": "🎹"},
  {"alias": "massive", "plugin_name": "Massive X", "type": "synth", "icon": "🎹"},
  {"alias": "kontakt", "plugin_name": "Kontakt 7", "type": "sampler", "icon": "🎼"},
  {"alias": "omnisphere", "plugin_name": "Omnisphere 2", "type": "synth", "icon": "🎹"},
  {"alias": "reaeq", "plugin_name": "ReaEQ", "type": "fx", "icon": "📊"},
  {"alias": "reacomp", "plugin_name": "ReaComp", "type": "fx", "icon": "📊"},
  {"alias": "fabfilter", "plugin_name": "FabFilter Pro-Q 3", "type": "fx", "icon": "📊"}
]
```

### 3.2 Preset Drum Mappings

```json
{
  "addictive_drums_v2": {
    "name": "Addictive Drums 2",
    "notes": {
      "kick": 36,
      "snare": 38,
      "snare_rim": 40,
      "snare_xstick": 37,
      "hi_hat": 42,
      "hi_hat_open": 46,
      "hi_hat_pedal": 44,
      "tom_high": 50,
      "tom_mid": 47,
      "tom_low": 45,
      "crash": 49,
      "crash_2": 57,
      "ride": 51,
      "ride_bell": 53,
      "china": 52,
      "splash": 55
    }
  },
  "general_midi": {
    "name": "General MIDI Drums",
    "notes": {
      "kick": 36,
      "snare": 38,
      "snare_rim": 40,
      "snare_xstick": 37,
      "hi_hat": 42,
      "hi_hat_open": 46,
      "hi_hat_pedal": 44,
      "tom_high": 50,
      "tom_mid": 47,
      "tom_low": 45,
      "crash": 49,
      "ride": 51
    }
  }
}
```

---

## 4. Context Injection Flow

### 4.1 Parse @Mentions

When processing user message:

```go
func ParseMentions(message string) []Mention {
    // Find all @word patterns
    // Resolve each to alias data
    // Return structured mentions
}

type Mention struct {
    Alias     string
    PluginName string
    Type      string
    MappingID *string
    Mapping   *DrumMapping  // Loaded if needed
}
```

### 4.2 Inject into LLM Context

```go
func BuildContext(message string, mentions []Mention) string {
    context := ""

    for _, m := range mentions {
        if m.Type == "drums" && m.Mapping != nil {
            context += fmt.Sprintf(
                "User selected %s (drums). Use canonical drum names: %s\n",
                m.PluginName,
                strings.Join(m.Mapping.CanonicalNames(), ", "),
            )
        } else {
            context += fmt.Sprintf(
                "User selected %s (%s).\n",
                m.PluginName, m.Type,
            )
        }
    }

    return context
}
```

### 4.3 Strip @Mentions from User Message

Before sending to LLM, optionally strip or transform @mentions:

```
Original: "add a beat to @addictivedrums"
Cleaned:  "add a beat to Addictive Drums 2"
```

---

## 5. Reaper Extension Integration

### 5.1 Autocomplete Data Flow

```
1. User types "@add"
2. Extension calls GET /api/v1/autocomplete?prefix=add
3. API returns suggestions
4. Extension shows popup
5. User selects "@addictivedrums"
6. Extension inserts into text
7. On send, extension includes mention context in request
```

### 5.2 Request Format (with mentions)

```json
{
  "question": "add a funky beat to @addictivedrums",
  "state": { ... },
  "mentions": [
    {
      "alias": "addictivedrums",
      "mapping_id": "addictive_drums_v2"
    }
  ]
}
```

---

## 6. Files to Create

### API (magda-api)

```
internal/
  models/
    plugin_alias.go
    drum_mapping.go
    user_preference.go
  api/handlers/
    autocomplete.go
    aliases.go
    mappings.go
    preferences.go
  database/
    migrations/
      00X_add_aliases_mappings_prefs.sql
    seeds/
      preset_aliases.json
      preset_mappings.json
```

### Reaper Extension (magda-reaper)

```
include/
  magda_autocomplete.h    // API client for autocomplete
src/
  magda_autocomplete.cpp
```

---

## 7. Migration Path

1. **Phase 1**: Add database tables and seed presets
2. **Phase 2**: Implement API endpoints
3. **Phase 3**: Add autocomplete to Reaper ImGui chat
4. **Phase 4**: Implement context injection in chat handler
5. **Phase 5**: Add preferences UI in Reaper extension
