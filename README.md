# MAGDA API

Stateless Go API for MAGDA (Musical AI Digital Assistant) - AI-powered music production assistant for REAPER.

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
| LLM | OpenAI GPT-5 |
| Observability | Sentry, Langfuse |

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
│   │   │   ├── magda.go       # DAW agent (chat, plugins, mix)
│   │   │   ├── jsfx.go        # JSFX agent
│   │   │   ├── drummer.go     # Drummer agent
│   │   │   ├── generation.go  # Arranger agent
│   │   │   └── mix.go         # Mix analysis
│   │   └── middleware/        # Auth, CORS, Sentry
│   ├── config/                # Configuration
│   ├── llm/                   # LLM providers (OpenAI, Gemini)
│   ├── observability/         # Langfuse integration
│   └── services/              # DSL parser
├── pkg/embedded/              # Embedded prompt resources
├── docker-compose.yml
└── Dockerfile
```

## Related Projects

- **[magda-reaper](https://github.com/Conceptual-Machines/magda-reaper)** - REAPER extension (C++)
- **[magda-agents-go](https://github.com/Conceptual-Machines/magda-agents-go)** - AI agents library
- **magda-cloud** - Hosted gateway (auth, billing) - private

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/your-feature`
3. Run checks: `make ci`
4. Commit with conventional commits: `git commit -m "feat: add feature"`
5. Push and create a PR
