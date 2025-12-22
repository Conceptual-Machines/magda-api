# MAGDA API

[![CI](https://github.com/Conceptual-Machines/magda-api/actions/workflows/ci.yml/badge.svg)](https://github.com/Conceptual-Machines/magda-api/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Conceptual-Machines/magda-api)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Stateless Go API for **MAGDA** (**M**ulti-**A**gent **G**enerative **D**AW **A**utomation) - AI-powered music production assistant for REAPER.

## Overview

MAGDA API provides AI agents for:
- 🎹 **DAW Control** - Natural language commands for REAPER
- 🎸 **JSFX Generation** - AI-assisted audio effect creation
- 🥁 **Drum Pattern Generation** - Intelligent drum programming
- 🎚️ **Mix Analysis** - AI-powered mixing suggestions
- 🎵 **Music Arrangement** - Chord progressions and melodies

## Architecture

This is a **stateless API** with no database. Authentication and user management are handled externally:

- **Self-hosted**: No auth required (`AUTH_MODE=none`)
- **Hosted (magda-cloud)**: Gateway handles auth (`AUTH_MODE=gateway`)

| Component | Tech |
|-----------|------|
| Language | Go 1.24+ |
| Framework | Gin |
| LLM | OpenAI GPT-4.1/5 |
| Observability | Sentry, Langfuse |

### Agent Architecture

Agents are organized by DAW to support multiple DAWs in the future:

```
agents/
├── core/           # Orchestration (DAW-agnostic)
├── reaper/         # REAPER-specific (daw, jsfx, plugin)
├── shared/         # Works on any DAW (drummer, arranger, mix)
└── ableton/        # (Future) Ableton Live support
```

### CFG-Constrained DSL Generation

Most agents use **Context-Free Grammar (CFG)** with GPT-5 to constrain the LLM to generate valid DSL code. We use the [grammar-school-go](https://github.com/Conceptual-Machines/grammar-school-go) library to build CFG tool payloads.

| Agent | DSL | Example |
|-------|-----|---------|
| DAW | MAGDA DSL | `track(instrument="Serum").new_clip(bar=1, length_bars=4)` |
| Arranger | Arranger DSL | `chord(root="C", type="maj7", duration=4)` |
| Drummer | Drum DSL | `pattern(drum=kick, grid="x---x---x---x---")` |
| JSFX | JSFX/EEL2 | Complete effect code with `@init`, `@sample`, `@slider` |

Each DSL has a Lark grammar definition that specifies valid syntax. GPT-5's CFG tool ensures output conforms to the grammar - no hallucinated function names or invalid syntax.

📖 **Learn more**: [GPT-5 Context-Free Grammar (CFG)](https://cookbook.openai.com/examples/gpt-5/gpt-5_new_params_and_tools#3-contextfree-grammar-cfg)

## Quick Start

### Docker (Recommended)

```bash
# Clone the repo
git clone https://github.com/Conceptual-Machines/magda-api.git
cd magda-api

# Create .env file
cat > .env << EOF
OPENAI_API_KEY=sk-your-key-here
AUTH_MODE=none
EOF

# Run with Docker Compose
docker compose up -d
```

The API will be available at `http://localhost:8080`

### From Source

```bash
# Prerequisites: Go 1.24+
go mod download
go run main.go
```

## API Endpoints

### Health & Status

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Health check |
| `GET /mcp/status` | MCP server status |
| `GET /api/metrics` | Runtime metrics |

### AI Agents (all POST)

| Endpoint | Description |
|----------|-------------|
| `/api/v1/chat` | DAW control via natural language |
| `/api/v1/chat/stream` | Streaming DAW control |
| `/api/v1/jsfx/generate` | Generate JSFX effects |
| `/api/v1/jsfx/generate/stream` | Streaming JSFX generation |
| `/api/v1/drummer/generate` | Generate drum patterns |
| `/api/v1/mix/analyze` | Analyze mix and get suggestions |
| `/api/v1/plugins/process` | Process plugin list for aliases |
| `/api/v1/aideas/generations` | Music arrangement generation |

## Usage Examples

### Health Check

```bash
curl http://localhost:8080/health
```

```json
{
  "status": "healthy",
  "mcp_server": {
    "status": "disabled",
    "url": ""
  }
}
```

### DAW Chat (MAGDA)

```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "question": "Create a new track called Bass with Serum",
    "state": {}
  }'
```

### JSFX Generation

```bash
curl -X POST http://localhost:8080/api/v1/jsfx/generate \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Create a simple compressor with threshold and ratio controls",
    "code": "",
    "filename": "compressor.jsfx"
  }'
```

### Drum Pattern Generation

```bash
curl -X POST http://localhost:8080/api/v1/drummer/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.1",
    "input_array": [
      {"role": "user", "content": "Create a rock drum pattern at 120 BPM"}
    ]
  }'
```

## Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `OPENAI_API_KEY` | OpenAI API key | Yes | - |
| `AUTH_MODE` | Auth mode: `none` or `gateway` | No | `none` |
| `PORT` | Server port | No | `8080` |
| `ENVIRONMENT` | `development` or `production` | No | `development` |
| `MCP_SERVER_URL` | MCP server endpoint | No | - |
| `SENTRY_DSN` | Sentry error tracking | No | - |
| `LANGFUSE_ENABLED` | Enable Langfuse tracing | No | `false` |
| `LANGFUSE_PUBLIC_KEY` | Langfuse public key | No | - |
| `LANGFUSE_SECRET_KEY` | Langfuse secret key | No | - |

## Auth Modes

### `AUTH_MODE=none` (Default)

No authentication required. Use for:
- Local development
- Self-hosted deployments
- Testing

### `AUTH_MODE=gateway`

Trusts `X-User-*` headers from upstream gateway. Use when running behind `magda-cloud`:

```
X-User-ID: 123
X-User-Email: user@example.com
X-User-Role: user
```

## Development

```bash
# Run locally
make dev

# Run tests
make test

# Run linter
make lint

# Build binary
make build

# Build Docker image
make docker-build
```

## Project Structure

```
magda-api/
├── main.go                    # Entry point
├── internal/
│   ├── api/
│   │   ├── router.go          # Route definitions
│   │   ├── handlers/          # Request handlers
│   │   └── middleware/        # Auth, CORS, Sentry
│   ├── agents/                # AI Agents (organized by DAW)
│   │   ├── core/              # Orchestration & shared config
│   │   │   ├── coordination/  # Routes requests to agents
│   │   │   └── config/        # Agent configuration
│   │   ├── reaper/            # REAPER-specific agents
│   │   │   ├── daw/           # REAPER control (tracks, clips, FX)
│   │   │   ├── jsfx/          # JSFX effect generator
│   │   │   └── plugin/        # Plugin management
│   │   └── shared/            # DAW-agnostic agents
│   │       ├── drummer/       # Drum pattern generator
│   │       ├── arranger/      # Chords, melodies, progressions
│   │       └── mix/           # Mix analysis
│   ├── config/                # App configuration
│   ├── llm/                   # LLM providers (OpenAI)
│   ├── prompt/                # Prompt builders
│   └── services/              # DSL parser
├── pkg/embedded/              # Embedded prompt resources
├── docker-compose.yml
└── Dockerfile
```

## Local Deployment

### Option 1: Docker Compose (Recommended)

```bash
# 1. Clone the repo
git clone https://github.com/Conceptual-Machines/magda-api.git
cd magda-api

# 2. Create .env file
cat > .env << 'EOF'
OPENAI_API_KEY=sk-your-key-here
AUTH_MODE=none
PORT=8080
ENVIRONMENT=development
EOF

# 3. Start the API
docker compose up -d

# 4. Verify it's running
curl http://localhost:8080/health
```

### Option 2: Build from Source

```bash
# 1. Prerequisites: Go 1.24+
go version  # Should show 1.24+

# 2. Clone and setup
git clone https://github.com/Conceptual-Machines/magda-api.git
cd magda-api
go mod download

# 3. Set environment
export OPENAI_API_KEY=sk-your-key-here
export AUTH_MODE=none

# 4. Run
go run main.go

# Or build and run binary
go build -o magda-api main.go
./magda-api
```

### Option 3: With REAPER Extension

To use MAGDA with REAPER, you also need the REAPER extension:

```bash
# 1. Start the API (see above)
docker compose up -d

# 2. Install magda-reaper extension
# Download from: https://github.com/Conceptual-Machines/magda-reaper/releases
# Copy to REAPER UserPlugins folder

# 3. Configure extension to point to API
# In REAPER: Extensions > MAGDA > Settings
# Set API URL to: http://localhost:8080
```

### Verifying Installation

```bash
# Health check
curl http://localhost:8080/health
# Expected: {"status":"healthy",...}

# Test DAW agent
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"question": "Create a track called Test", "state": {}}'

# Test JSFX generator
curl -X POST http://localhost:8080/api/v1/jsfx/generate \
  -H "Content-Type: application/json" \
  -d '{"message": "Simple gain plugin", "code": "", "filename": "test.jsfx"}'
```

## Related Projects

- **[magda-reaper](https://github.com/Conceptual-Machines/magda-reaper)** - REAPER extension (C++)
- **magda-cloud** - Hosted gateway (auth, billing) - private

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/your-feature`
3. Run checks: `make ci`
4. Commit with conventional commits: `git commit -m "feat: add feature"`
5. Push and create a PR
