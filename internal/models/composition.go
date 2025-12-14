package models

// GenerationRequest wraps the user's generation parameters
type GenerationRequest struct {
	UserPrompt     string      `json:"user_prompt"`
	MusicalContext []NoteEvent `json:"musical_context,omitempty"` // User's recorded notes (from music.go)
	Variations     int         `json:"variations"`                // How many variations to generate
	Seed           *int        `json:"seed,omitempty"`            // Optional seed for reproducibility

	// Musical parameters
	Spread  string `json:"spread"`  // "tight", "medium", "wide" (voicing width)
	Novelty string `json:"novelty"` // "low", "medium", "high" (harmonic adventurousness)

	// One-shot mode parameters
	KeepContext   bool   `json:"keep_context"`
	ReasoningMode string `json:"reasoning_mode"`
}
