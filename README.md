# MAGDA API

Go-based API service for MAGDA (Musical AI Digital Assistant) - Backend deployment for the MAGDA system.

## Architecture

- **Language**: Go 1.24+
- **Framework**: Gin
- **Database**: PostgreSQL (AWS RDS)
- **Auth**: JWT tokens
- **ORM**: GORM
- **Error Tracking**: Sentry
- **Observability**: Structured logging with request IDs

## Features

- 🎵 **Music Generation**: AI-powered musical sequence generation using OpenAI GPT-5-mini
- 🔐 **Authentication**: User registration and JWT-based authentication
- 🎯 **MCP Integration**: Optional Music Composition Platform server integration
- 📊 **Observability**: Sentry error tracking and structured logging
- 🚀 **Performance**: Minimal resource footprint (~30MB RAM)
- 🔍 **Request Tracking**: Unique request IDs for debugging
- ⚡ **Response Time Monitoring**: Request duration tracking and logging

## Setup

### Prerequisites

- Go 1.23+
- PostgreSQL (or AWS RDS)
- OpenAI API key

### Installation

1. Clone the repository
2. Copy `.env.example` to `.env` and configure:
   ```bash
   cp .env.example .env
   ```
3. Install dependencies:
   ```bash
   make install
   ```
4. Run the server:
   ```bash
   make dev
   ```

## API Endpoints

### Public Endpoints

- `GET /health` - Health check with MCP status
- `GET /mcp/status` - Detailed MCP server status
- `POST /auth/register` - User registration
- `POST /auth/login` - User login
- `POST /api/generate` - Generate music sequence (temporarily public for testing)

### Protected Endpoints (require JWT)

- `GET /api/me` - Get current user profile

## API Examples

All examples below are from our [smoke test suite](./smoke-test.sh).

### 1. Health Check

Check if the API is running and get MCP server status:

```bash
curl https://api.musicalaideas.com/health
```

**Response:**
```json
{
  "status": "healthy",
  "mcp_server": {
    "status": "enabled",
    "url": "https://mcp.musicalaideas.com"
  }
}
```

[See full test in smoke-test.sh](./smoke-test.sh#L23-L36)

### 2. Generate Music

Generate a musical sequence with AI:

```bash
curl -X POST https://api.musicalaideas.com/api/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5-mini",
    "input_array": [
      {
        "role": "user",
        "content": "{\"user_prompt\": \"Generate a simple C major chord progression with 2 chords\"}"
      }
    ]
  }'
```

**Response:**
```json
{
  "genResults": [
    {
      "description": "C major to G major chord progression",
      "notes": [
        {
          "midiNoteNumber": 60,
          "velocity": 100,
          "startBeats": 0.0,
          "durationBeats": 2.0
        },
        {
          "midiNoteNumber": 64,
          "velocity": 100,
          "startBeats": 0.0,
          "durationBeats": 2.0
        },
        {
          "midiNoteNumber": 67,
          "velocity": 100,
          "startBeats": 0.0,
          "durationBeats": 2.0
        }
      ]
    }
  ],
  "usage": {
    "total_tokens": 1234,
    "input_tokens": 567,
    "output_tokens": 667,
    "output_tokens_details": {
      "reasoning_tokens": 256
    }
  },
  "mcpUsed": true,
  "mcpCalls": 2
}
```

[See full test in smoke-test.sh](./smoke-test.sh#L39-L81)

### 3. Generate with Musical Context

Include multiple inputs or musical context:

```bash
curl -X POST https://api.musicalaideas.com/api/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5-mini",
    "input_array": [
      {
        "role": "user",
        "content": "{\"user_prompt\": \"Continue this melody in C major\"}"
      },
      {
        "role": "context",
        "content": "Previous notes: C D E"
      }
    ]
  }'
```

**Note**: The `input_array` format follows the OpenAI Responses API structure. Each item can have:
- `role`: "user", "context", or "system"
- `content`: String or JSON string with musical data

### 4. MCP Server Status

Get detailed MCP server configuration:

```bash
curl https://api.musicalaideas.com/mcp/status
```

**Response:**
```json
{
  "status": "enabled",
  "url": "https://mcp.musicalaideas.com",
  "label": "mcp-musicalaideas-com",
  "details": "MCP server is configured and enabled."
}
```

## Development

```bash
# Run in development mode
make dev

# Run tests with coverage
make test-coverage

# Run linter
make lint

# Format code
make fmt

# Build binary
make build

# Build Docker image
make docker-build

# Run Docker container
make docker-run

# Run all checks (lint + test + build)
make ci
```

## Testing

### Run All Tests
```bash
make test
```

### Smoke Tests
Quick verification that the API is working after deployment:

```bash
# Test production
./smoke-test.sh https://api.musicalaideas.com

# Test local
./smoke-test.sh http://localhost:8080
```

The smoke test verifies:
- ✅ Health endpoint responds
- ✅ Generation endpoint works
- ✅ CORS headers are present
- ✅ SSL certificate is valid (for HTTPS)

[View complete smoke test suite →](./smoke-test.sh)

## Deployment

### Quick Deploy

The API is deployed automatically via GitHub Actions on push to `main`:

1. CI runs tests and linters
2. Docker image is built and pushed to ECR
3. EC2 instance pulls and restarts containers
4. Smoke tests verify deployment

See [DEPLOYMENT.md](./DEPLOYMENT.md) for detailed deployment guide.

### Infrastructure

- **EC2**: t4g.nano ($3/month)
- **Database**: RDS PostgreSQL
- **DNS**: Cloudflare (proxied)
- **SSL**: Let's Encrypt (auto-renewal)
- **Monitoring**: CloudWatch + Sentry

## Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `ENVIRONMENT` | Environment (development/production) | No | `development` |
| `PORT` | Server port | No | `8080` |
| `DATABASE_URL` | PostgreSQL connection string | Yes | - |
| `JWT_SECRET` | JWT signing secret | Yes | - |
| `OPENAI_API_KEY` | OpenAI API key | Yes | - |
| `MCP_SERVER_URL` | MCP server endpoint (optional) | No | - |
| `SENTRY_DSN` | Sentry error tracking DSN (optional) | No | - |
| `RELEASE_VERSION` | Release version for Sentry | No | `dev` |

### Setting Up Sentry

Sentry provides error tracking and performance monitoring:

1. Sign up at https://sentry.io (free tier: 5K errors/month)
2. Create a project named `magda-api`
3. Copy your DSN
4. Add to `.env`: `SENTRY_DSN=https://your-dsn@sentry.io/project-id`
5. Add to GitHub Secrets: `SENTRY_DSN`

See [docs/SENTRY_SETUP.md](./docs/SENTRY_SETUP.md) for complete setup guide.

## Observability

### Logging

All requests are logged with structured fields:

```
[INFO] Request completed {request_id=abc123, duration_ms=145, status_code=200, method=POST, path=/api/generate}
```

### Request Tracking

Every request gets a unique ID returned in the `X-Request-ID` header:

```bash
curl -i https://api.musicalaideas.com/health
# X-Request-ID: 550e8400-e29b-41d4-a716-446655440000
```

Use this ID to track requests across logs and Sentry.

### Performance Monitoring

Request duration is automatically logged:

```
⏱️  OPENAI API CALL COMPLETED in 2.3s
⏱️  TOTAL GENERATION TIME: 2.5s
```

### Error Tracking

Errors are automatically sent to Sentry with:
- Stack traces
- Request context
- User information
- Breadcrumbs

Dashboard: https://sentry.io/organizations/[your-org]/projects/magda-api/

## Project Structure

```
aideas-api/
├── main.go                 # Application entry point
├── internal/
│   ├── api/
│   │   ├── handlers/      # HTTP request handlers
│   │   ├── middleware/    # Gin middleware (auth, CORS, Sentry)
│   │   └── router.go      # Route definitions
│   ├── config/            # Configuration management
│   ├── database/          # Database connection and migrations
│   ├── logger/            # Structured logging utilities
│   ├── models/            # Data models
│   ├── prompt/            # Prompt building and loading
│   └── services/          # Business logic
├── data/                  # Prompt data and music theory files
├── docs/                  # Documentation
├── infra/                 # Terraform infrastructure
└── smoke-test.sh          # Deployment verification

## Contributing

1. Create a feature branch: `git checkout -b feat/your-feature`
2. Make changes and add tests
3. Run checks: `make ci`
4. Commit: `git commit -m "feat: add your feature"`
5. Push and create PR
6. Wait for CI to pass
7. Merge to `main`

## License

Proprietary - All rights reserved

## Support

- **Issues**: https://github.com/Conceptual-Machines/magda-api/issues
- **Sentry**: https://sentry.io
- **API Docs**: https://api.musicalaideas.com/health
