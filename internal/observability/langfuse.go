package observability

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/Conceptual-Machines/magda-api/internal/config"
	langfuse "github.com/henomis/langfuse-go"
	"github.com/henomis/langfuse-go/model"
	"github.com/openai/openai-go/responses"
)

// LangfuseClient wraps the Langfuse client with our configuration
type LangfuseClient struct {
	client  *langfuse.Langfuse
	enabled bool
	ctx     context.Context
}

var globalClient *LangfuseClient

// InitializeLangfuse initializes the global Langfuse client
func InitializeLangfuse(ctx context.Context, cfg *config.Config) *LangfuseClient {
	if !cfg.LangfuseEnabled || cfg.LangfuseSecretKey == "" {
		log.Println("⚠️  Langfuse not configured (LANGFUSE_ENABLED=false or LANGFUSE_SECRET_KEY not set)")
		globalClient = &LangfuseClient{enabled: false, ctx: ctx}
		return globalClient
	}

	// Set environment variables for the SDK
	// The henomis SDK reads from environment variables
	if cfg.LangfuseSecretKey != "" {
		// Note: The SDK may need these set differently - check SDK docs
		// For now, we'll create the client and it should read from env
		lf := langfuse.New(ctx)

		globalClient = &LangfuseClient{
			client:  lf,
			enabled: true,
			ctx:     ctx,
		}

		log.Printf("✅ Langfuse initialized (host: %s)", cfg.LangfuseHost)
		log.Printf("🔍 Langfuse: Public key set: %v, Secret key set: %v",
			os.Getenv("LANGFUSE_PUBLIC_KEY") != "",
			os.Getenv("LANGFUSE_SECRET_KEY") != "")
		return globalClient
	}

	globalClient = &LangfuseClient{enabled: false, ctx: ctx}
	return globalClient
}

// GetClient returns the global Langfuse client
func GetClient() *LangfuseClient {
	if globalClient == nil {
		return &LangfuseClient{enabled: false, ctx: context.Background()}
	}
	return globalClient
}

// IsEnabled returns whether Langfuse is enabled
func (c *LangfuseClient) IsEnabled() bool {
	return c.enabled && c.client != nil
}

// StartTrace starts a new trace in Langfuse
func (c *LangfuseClient) StartTrace(ctx context.Context, name string, metadata map[string]interface{}) *Trace {
	if !c.IsEnabled() {
		return &Trace{enabled: false, ctx: ctx}
	}

	trace, err := c.client.Trace(&model.Trace{
		Name:     name,
		Metadata: metadata,
	})
	if err != nil {
		log.Printf("⚠️  Failed to create Langfuse trace: %v", err)
		return &Trace{enabled: false, ctx: ctx}
	}

	log.Printf("🔍 Langfuse: Created trace %s (name: %s)", trace.ID, name)
	return &Trace{
		trace:   trace,
		enabled: true,
		ctx:     ctx,
		client:  c.client,
	}
}

// Trace represents a Langfuse trace
type Trace struct {
	trace   *model.Trace
	enabled bool
	ctx     context.Context
	client  *langfuse.Langfuse
}

// Generation creates a new generation span within the trace
func (t *Trace) Generation(name string, metadata map[string]interface{}) *Generation {
	if !t.enabled {
		return &Generation{enabled: false, ctx: t.ctx}
	}

	now := time.Now()
	gen, err := t.client.Generation(&model.Generation{
		TraceID:   t.trace.ID,
		Name:      name,
		StartTime: &now,
		Metadata:  metadata,
	}, nil)
	if err != nil {
		log.Printf("⚠️  Failed to create Langfuse generation: %v", err)
		return &Generation{enabled: false, ctx: t.ctx}
	}

	log.Printf("🔍 Langfuse: Created generation %s (trace: %s)", gen.ID, t.trace.ID)
	return &Generation{
		generation: gen,
		enabled:    true,
		ctx:        t.ctx,
		client:     t.client,
	}
}

// Finish completes the trace and flushes data to Langfuse
func (t *Trace) Finish() {
	if t.enabled && t.client != nil {
		// Flush ensures all batched events are sent
		// The SDK batches events and sends them asynchronously
		// Flush() waits for all queued events to be sent
		log.Printf("🔍 Langfuse: Flushing trace %s...", t.trace.ID)
		t.client.Flush(t.ctx)
		log.Printf("🔍 Langfuse: Flush completed for trace %s (check dashboard in a few seconds)", t.trace.ID)
	}
}

// SetMetadata adds metadata to the trace
func (t *Trace) SetMetadata(metadata map[string]interface{}) {
	if t.enabled && t.trace != nil {
		t.trace.Metadata = metadata
		// Note: The SDK may not support updating traces directly
		// This would need to be handled differently
	}
}

// Generation represents a Langfuse generation span
type Generation struct {
	generation *model.Generation
	enabled    bool
	ctx        context.Context
	client     *langfuse.Langfuse
}

// Input sets the input for the generation
func (g *Generation) Input(input interface{}) {
	if g.enabled && g.generation != nil {
		g.generation.Input = input
	}
}

// Output sets the output for the generation
func (g *Generation) Output(output interface{}) {
	if g.enabled && g.generation != nil {
		g.generation.Output = output
	}
}

// Usage sets the token usage for the generation
func (g *Generation) Usage(usage map[string]interface{}) {
	if g.enabled && g.generation != nil {
		// Convert usage map to model.Usage
		g.generation.Usage = convertUsageMap(usage)
	}
}

// Metadata adds metadata to the generation
func (g *Generation) Metadata(metadata map[string]interface{}) {
	if g.enabled && g.generation != nil {
		if g.generation.Metadata == nil {
			g.generation.Metadata = make(map[string]interface{})
		}
		if md, ok := g.generation.Metadata.(map[string]interface{}); ok {
			for k, v := range metadata {
				md[k] = v
			}
		} else {
			g.generation.Metadata = metadata
		}
	}
}

// Finish completes the generation and sends it to Langfuse
func (g *Generation) Finish() {
	if g.enabled && g.generation != nil && g.client != nil {
		now := time.Now()
		g.generation.EndTime = &now
		_, err := g.client.GenerationEnd(g.generation)
		if err != nil {
			log.Printf("⚠️  Failed to end Langfuse generation: %v", err)
		} else {
			log.Printf("🔍 Langfuse: Generation %s ended and queued for sending", g.generation.ID)
		}
	}
}

// SetLevel sets the level of the generation
func (g *Generation) SetLevel(level string) {
	if g.enabled && g.generation != nil {
		g.generation.Level = model.ObservationLevel(level)
	}
}

// LogOpenAIResponseStruct is a convenience method that takes the full OpenAI response struct
// and automatically extracts everything needed for Langfuse
func (g *Generation) LogOpenAIResponseStruct(
	modelName string,
	inputMessages []map[string]interface{},
	resp *responses.Response,
	metadata map[string]interface{},
) {
	if !g.enabled {
		return
	}

	// Extract output text
	outputText := resp.OutputText()

	// Convert usage to model.Usage
	usage := model.Usage{
		Input:      int(resp.Usage.InputTokens),
		Output:     int(resp.Usage.OutputTokens),
		Total:      int(resp.Usage.TotalTokens),
		Unit:       model.ModelUsageUnitTokens,
		TotalCost:  CalculateOpenAICost(modelName, resp.Usage),
		InputCost:  0, // Will be calculated if needed
		OutputCost: 0, // Will be calculated if needed
	}
	// Note: SDK may not support reasoning tokens directly yet
	// Calculate cost
	cost := CalculateOpenAICost(modelName, resp.Usage)

	// Merge metadata
	finalMetadata := map[string]interface{}{
		"model":    modelName,
		"cost_usd": cost,
	}
	for k, v := range metadata {
		finalMetadata[k] = v
	}

	// Set everything
	g.Input(inputMessages)
	if outputText != "" {
		g.Output(outputText)
	}
	g.generation.Usage = usage
	g.generation.Model = modelName
	g.Metadata(finalMetadata)
}

// convertUsageMap converts a usage map to model.Usage
func convertUsageMap(usage map[string]interface{}) model.Usage {
	result := model.Usage{
		Unit: model.ModelUsageUnitTokens,
	}

	if input, ok := usage["input_tokens"].(int); ok {
		result.Input = input
	} else if input, ok := usage["input_tokens"].(int64); ok {
		result.Input = int(input)
	}

	if output, ok := usage["output_tokens"].(int); ok {
		result.Output = output
	} else if output, ok := usage["output_tokens"].(int64); ok {
		result.Output = int(output)
	}

	if total, ok := usage["total_tokens"].(int); ok {
		result.Total = total
	} else if total, ok := usage["total_tokens"].(int64); ok {
		result.Total = int(total)
	}

	if cost, ok := usage["cost_usd"].(float64); ok {
		result.TotalCost = cost
	}

	return result
}
