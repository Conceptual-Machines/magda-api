package prompt

// Builder builds prompts for the arranger agent
type Builder struct{}

// NewPromptBuilder creates a new prompt builder
func NewPromptBuilder() *Builder {
	return &Builder{}
}

// OutputFormatDSL represents DSL output format
const OutputFormatDSL = "dsl"

// OutputFormatJSON represents JSON output format
const OutputFormatJSON = "json"

// BuildPromptWithFormat builds a system prompt for the given output format
func (b *Builder) BuildPromptWithFormat(format string) (string, error) {
	// TODO: Implement proper prompt building
	return "", nil
}

