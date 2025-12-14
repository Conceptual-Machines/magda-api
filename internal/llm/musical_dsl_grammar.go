package llm

// GetMusicalDSLGrammar returns the Lark grammar definition for Musical Composition DSL
// The DSL uses a simple functional format: choice("description", [note(midi,vel,start,dur), ...])
func GetMusicalDSLGrammar() string {
	return `
// Musical Composition DSL Grammar - Simple functional syntax for fast generation
// Format: choice("desc", [note(60,100,0,2), ...]) or multiple: composition().addChoice(...).addChoice(...)

// ---------- Start rule ----------
start: composition

// ---------- Main composition structure ----------
composition: choice_list | single_choice

// Multiple choices with chaining
choice_list: "composition" "(" ")" chain_item+
chain_item: ".addChoice" "(" choice_params ")"

// Single choice (simpler form)
single_choice: "choice" "(" choice_params ")"

// ---------- Choice parameters ----------
choice_params: STRING "," notes_array
             | "description" "=" STRING "," "notes" "=" notes_array
             | "description" "=" STRING ("," choice_param)*

choice_param: "notes" "=" notes_array
            | "chords" "=" chords_array

// ---------- Notes array ----------
notes_array: "[" (note_item ("," SP note_item)*)? "]"

// Note: positional args (midi, velocity, start, duration) for speed
// Or named: note(midi=60, velocity=100, start=0, duration=2)
note_item: "note" "(" note_params ")"

note_params: NUMBER "," NUMBER "," NUMBER "," NUMBER  // positional: midi, vel, start, dur
           | note_named_params

note_named_params: note_named_param ("," SP note_named_param)*
note_named_param: "midi" "=" NUMBER
                | "velocity" "=" NUMBER
                | "start" "=" NUMBER
                | "duration" "=" NUMBER

// ---------- Chords array (optional) ----------
chords_array: "[" (chord_item ("," SP chord_item)*)? "]"
chord_item: "chord" "(" chord_params ")"
chord_params: STRING "," NUMBER "," NUMBER  // positional: symbol, start, duration
            | chord_named_params

chord_named_params: chord_named_param ("," SP chord_named_param)*
chord_named_param: "symbol" "=" STRING
                 | "start" "=" NUMBER
                 | "duration" "=" NUMBER

// ---------- Terminals ----------
SP: " "+
STRING: /"[^"]*"/
NUMBER: /-?\d+(\.\d+)?/
`
}
