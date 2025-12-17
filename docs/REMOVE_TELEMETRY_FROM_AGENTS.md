# Remove Telemetry from Agents

## Decision

**Telemetry (metrics, sentry) should NOT be part of the agent framework.** Agents should be framework-agnostic.

## Why

- Agents are meant to be open-source, reusable framework components
- Telemetry is deployment-specific (different deployments may use different monitoring solutions)
- Makes agents harder to test and reuse
- Violates separation of concerns

## What to Remove

### From `magda-agents` (Framework):

1. **Remove Sentry dependencies:**
   - Remove `github.com/getsentry/sentry-go` imports
   - Remove all `sentry.StartTransaction()` calls
   - Remove all `sentry.CaptureException()` calls
   - Remove Sentry breadcrumbs

2. **Remove Metrics dependencies:**
   - Remove `metrics` package imports
   - Remove `*metrics.SentryMetrics` fields from agent structs
   - Remove all `metrics.Record*()` method calls

3. **Keep logging:**
   - Standard `log.Printf()` is fine - it's part of Go stdlib
   - No external logging frameworks

### In `magda-api` (Deployment):

- Can wrap agents with telemetry
- Add metrics in handler layer
- Use middleware for observability

## Migration Strategy

1. Remove telemetry from `magda-agents` files
2. Add telemetry wrapper in `magda-api` handlers
3. Or pass optional metrics interface to agents (if needed)

## Files to Update in `magda-agents`:

- `go/agents/arranger/arranger_agent.go`
- `go/agents/arranger/arranger_stream.go`
- `go/agents/daw/daw_agent.go`
- Any other agent files
