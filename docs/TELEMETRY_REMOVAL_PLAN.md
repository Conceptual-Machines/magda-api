# Telemetry Removal from Agents

## Decision

**Telemetry (metrics, Sentry) should NOT be part of agents.** Agents must be framework-agnostic.

## Current Issues

1. **Agents in `magda-agents` import deployment-specific packages:**
   - `config` - deployment-specific configuration
   - `metrics` - deployment-specific telemetry (SentryMetrics)
   - `sentry-go` - deployment-specific observability

2. **Agents have telemetry code:**
   - Sentry transactions
   - Metrics recording
   - These should be in the deployment layer, not the framework

## Solution

### 1. Remove from `magda-agents` (Framework):

**Remove:**
- ❌ All `sentry` imports and code
- ❌ All `metrics` imports and fields
- ❌ `config` imports (use interfaces/parameters instead)
- ❌ All telemetry calls (`sentry.StartTransaction`, `metrics.Record*`, etc.)

**Keep:**
- ✅ Standard `log.Printf()` (stdlib)
- ✅ Core agent logic
- ✅ LLM provider interface
- ✅ Models and prompt builders

### 2. Add to `magda-api` (Deployment):

**Option A: Handler-level telemetry** (Recommended)
- Add Sentry transactions in handlers
- Record metrics after agent calls
- Wrap agent calls with telemetry

**Option B: Optional metrics interface**
- Pass optional `MetricsRecorder` interface to agents
- Default to no-op if not provided
- Deployment can inject their own implementation

## Migration Steps

1. ✅ Remove telemetry imports from `magda-agents` agents
2. ✅ Remove all Sentry/metrics code from agents
3. ✅ Remove config dependencies (use parameters/interfaces)
4. ⚠️ Add telemetry wrapper in `magda-api` handlers
5. ⚠️ Test that agents work without telemetry

## Files to Clean in `magda-agents`:

- `go/agents/arranger/arranger_agent.go`
- `go/agents/arranger/arranger_stream.go`
- `go/agents/daw/daw_agent.go`
- Any other agent files

## Example: Before vs After

**Before (❌ Bad):**
```go
type ArrangerAgent struct {
    provider llm.Provider
    metrics  *metrics.SentryMetrics  // ❌ Deployment-specific
}

func (a *ArrangerAgent) Generate(ctx context.Context, ...) {
    transaction := sentry.StartTransaction(...)  // ❌ Deployment-specific
    defer transaction.Finish()

    // ... agent logic ...

    a.metrics.RecordGenerationDuration(...)  // ❌ Deployment-specific
}
```

**After (✅ Good):**
```go
type ArrangerAgent struct {
    provider llm.Provider
    // No metrics field - framework-agnostic!
}

func (a *ArrangerAgent) Generate(ctx context.Context, ...) {
    // ... agent logic only, no telemetry ...
    log.Printf("✅ Generation complete")  // ✅ Stdlib logging OK
}
```

**In `magda-api` handler (✅ Deployment layer):**
```go
func (h *GenerationHandler) Generate(c *gin.Context) {
    // Telemetry at deployment layer
    transaction := sentry.StartTransaction(...)
    defer transaction.Finish()

    result, err := h.arrangerAgent.Generate(...)

    // Record metrics at deployment layer
    h.metrics.RecordGenerationDuration(...)

    // ... return response ...
}
```










