# Integration Tests for Refactored API

This document describes the integration tests available for verifying the refactored API endpoints.

## Quick Start

Run all integration tests:
```bash
./run-all-integration-tests.sh
```

Run specific test suite:
```bash
./test-integration-refactoring.sh
```

## Prerequisites

1. **Server must be running**: Start the API server with `make dev`
2. **Environment variables**: Set in `.env` or `.envrc`:
   - `OPENAI_API_KEY` - Required for LLM calls
   - `AIDEAS_EMAIL` - Optional, for authentication
   - `AIDEAS_PASSWORD` - Optional, for authentication
   - `AIDEAS_API_URL` - Optional, defaults to `http://localhost:8080`

## Test Suites

### 1. Comprehensive Refactoring Tests (`test-integration-refactoring.sh`)

**Purpose**: Verify all refactored endpoints are working correctly

**Tests**:
- ✅ Health Check
- ✅ AIDEAS Generation (non-streaming) - `/api/v1/aideas/generations`
- ✅ AIDEAS Generation (streaming) - `/api/v1/aideas/generations` with `stream: true`
- ✅ MAGDA Chat - `/api/v1/magda/chat`
- ✅ MAGDA Chat Streaming - `/api/v1/magda/chat/stream`
- ✅ MAGDA DSL Parser - `/api/v1/magda/dsl`
- ✅ CFG Grammar Cleaning - Verifies grammar-school integration

**Usage**:
```bash
./test-integration-refactoring.sh
```

### 2. AIDEAS Generation Test (`test-generation.sh`)

**Purpose**: Test the AIDEAS music generation endpoint

**Tests**:
- Non-streaming generation
- Response format validation
- Note generation verification

**Usage**:
```bash
./test-generation.sh [base-url]
```

### 3. MAGDA Integration Test (`test-magda.sh`)

**Purpose**: Test MAGDA endpoints using Go test framework

**Tests**:
- Health check
- MAGDA chat endpoint
- Error handling
- State management

**Usage**:
```bash
./test-magda.sh
```

### 4. MAGDA DSL E2E Test (`test-magda-dsl-e2e.sh`)

**Purpose**: End-to-end test of MAGDA DSL flow

**Tests**:
- LLM generates DSL
- Go parser translates DSL to actions
- Streaming response handling
- Timing measurements

**Usage**:
```bash
./test-magda-dsl-e2e.sh
```

### 5. MAGDA Streaming Test (`test-magda-stream.sh`)

**Purpose**: Test MAGDA streaming functionality

**Tests**:
- SSE event handling
- Action streaming
- Stream completion

**Usage**:
```bash
./test-magda-stream.sh
```

## Endpoint Changes

### Before Refactoring
- `/api/v1/generations` - Music generation (AIDEAS)

### After Refactoring
- `/api/v1/aideas/generations` - Music generation (AIDEAS) ✨ **NEW**
- `/api/v1/magda/chat` - MAGDA chat (uses magda-agents) ✨ **UPDATED**
- `/api/v1/magda/chat/stream` - MAGDA streaming
- `/api/v1/magda/dsl` - DSL parser test

## What's Being Tested

### AIDEAS Endpoints
- ✅ Uses local `arranger` agent (not magda-agents)
- ✅ Supports both DSL and JSON Schema output formats
- ✅ Streaming and non-streaming modes
- ✅ CFG grammar cleaning via grammar-school

### MAGDA Endpoints
- ✅ Uses `magda-agents` package (imported dependency)
- ✅ DAW agent from magda-agents
- ✅ Plugin agent from magda-agents
- ✅ CFG grammar cleaning via grammar-school
- ✅ DSL parsing and translation

### Grammar-School Integration
- ✅ CFG grammars are cleaned before sending to OpenAI
- ✅ Removes Lark directives and comments
- ✅ Works for both AIDEAS and MAGDA agents

## Expected Results

### Successful Test Run
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Test Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ Passed: 7
❌ Failed: 0
⚠️  Skipped: 0

🎉 All tests passed!
```

### Common Issues

1. **Server not running**
   ```
   ❌ Server is not running at http://localhost:8080
   ```
   **Solution**: Run `make dev` to start the server

2. **Authentication failed**
   ```
   ❌ Failed to authenticate
   ```
   **Solution**: Set `AIDEAS_EMAIL` and `AIDEAS_PASSWORD` in `.env`

3. **OpenAI API key missing**
   ```
   ❌ ERROR: OPENAI_API_KEY is not set!
   ```
   **Solution**: Set `OPENAI_API_KEY` in `.env` or `.envrc`

4. **Endpoint not found (404)**
   ```
   ❌ Expected 200 OK, got 404
   ```
   **Solution**: Verify you're using the new endpoint paths:
   - `/api/v1/aideas/generations` (not `/api/v1/generations`)
   - `/api/v1/magda/chat` (unchanged)

## Debugging

### Enable verbose output
```bash
DEBUG=1 ./test-integration-refactoring.sh
```

### Check server logs
The server logs will show:
- `📝 Grammar cleaned for CFG: X chars (original: Y chars)` - Confirms grammar-school is working
- `🔧 CFG GRAMMAR CONFIGURED` - CFG is being used
- `🤖 DAW AGENT INITIALIZED` - MAGDA agent from magda-agents is loaded
- `🎵 GENERATION SERVICE INITIALIZED` - AIDEAS arranger agent is loaded

### Test individual endpoints
```bash
# Health check
curl http://localhost:8080/health

# AIDEAS generation (requires auth)
curl -X POST http://localhost:8080/api/v1/aideas/generations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"model": "gpt-5-mini", "input_array": [...], "stream": false}'

# MAGDA chat (requires auth)
curl -X POST http://localhost:8080/api/v1/magda/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"question": "Create a track", "state": {}}'
```

## Continuous Integration

These tests can be integrated into CI/CD pipelines. Ensure:
1. Server is started before tests run
2. Environment variables are set
3. Database is available (for auth tests)
4. OpenAI API key is configured (for real LLM calls)

## Next Steps

After running tests successfully:
1. ✅ Verify all endpoints respond correctly
2. ✅ Check server logs for grammar-school integration
3. ✅ Verify magda-agents is being used (check logs)
4. ✅ Test with real REAPER extension (if available)
