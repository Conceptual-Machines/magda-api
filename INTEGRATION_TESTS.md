# Integration Tests

## Quick Start

```bash
# Start the server
make dev

# Run smoke tests
./tests/smoke/run-all.sh http://localhost:8080

# Run Go integration tests
go test -v ./internal/api/handlers -run TestMagda
```

## Prerequisites

1. **Server running**: Start with `make dev` or `docker compose up`
2. **OPENAI_API_KEY**: Set in `.env` file

## Test Suites

### Smoke Tests (`tests/smoke/`)

Quick API verification tests:

```bash
./tests/smoke/run-all.sh [base-url]
```

Individual tests:
- `health.sh` - Health endpoint
- `metrics.sh` - Metrics endpoint
- `chat.sh` - MAGDA chat endpoint
- `jsfx.sh` - JSFX generation endpoint

### Go Integration Tests

```bash
# All MAGDA tests
go test -v ./internal/api/handlers -run TestMagda

# Specific test
go test -v ./internal/api/handlers -run TestMagdaChat
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/api/metrics` | GET | Runtime metrics |
| `/mcp/status` | GET | MCP server status |
| `/api/v1/chat` | POST | MAGDA DAW chat |
| `/api/v1/chat/stream` | POST | MAGDA streaming |
| `/api/v1/jsfx/generate` | POST | JSFX generation |
| `/api/v1/jsfx/generate/stream` | POST | JSFX streaming |
| `/api/v1/drummer/generate` | POST | Drum pattern generation |
| `/api/v1/mix/analyze` | POST | Mix analysis |
| `/api/v1/generations` | POST | Music arrangement (Arranger agent) |

## Example Requests

### Health Check

```bash
curl http://localhost:8080/health
```

### MAGDA Chat

```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "question": "create a track called Bass",
    "state": {"tracks": []}
  }'
```

### JSFX Generation

```bash
curl -X POST http://localhost:8080/api/v1/jsfx/generate \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Create a compressor",
    "code": "",
    "filename": "comp.jsfx"
  }'
```

## Debugging

### Check server logs

```bash
make dev
# Logs will show:
# 📨 MAGDA Chat: Received request
# 🚀 MAGDA Chat: Calling Orchestrator.GenerateActions
# ✅ MAGDA Chat: GenerateActions succeeded
```

### Verbose test output

```bash
go test -v ./internal/api/handlers -run TestMagda 2>&1 | tee test.log
```

## CI Integration

Tests run automatically on push via GitHub Actions. See `.github/workflows/ci.yml`.
