# Project Setup Guide

## Quick Start

```bash
# Clone the repository
git clone https://github.com/Conceptual-Machines/magda-api.git
cd magda-api

# Create .env file
cat > .env << EOF
OPENAI_API_KEY=sk-your-openai-key
AUTH_MODE=none
EOF

# Run development server
make dev

# Or with Docker
docker compose up
```

The API will be available at `http://localhost:8080`

## Available Commands

```bash
# Development
make dev              # Run development server
make setup            # Setup development environment

# Code Quality
make fmt              # Format code
make lint             # Run linters
make tidy             # Tidy dependencies
make check            # Run all checks (lint + test)

# Testing
make test             # Run tests
make test-coverage    # Run tests with coverage report

# Building
make build            # Build binary
make clean            # Clean build artifacts

# Docker
make docker-build     # Build Docker image
make docker-run       # Run Docker container

# Docker Compose
make dc-up            # Start services
make dc-down          # Stop services
make dc-logs          # View logs
make dc-rebuild       # Rebuild and restart

# All-in-one
make all              # tidy + fmt + check + build
make ci               # CI pipeline (tidy + fmt + check + build)
```

## Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `OPENAI_API_KEY` | OpenAI API key | Yes | - |
| `AUTH_MODE` | `none` or `gateway` | No | `none` |
| `PORT` | Server port | No | `8080` |
| `ENVIRONMENT` | `development` or `production` | No | `development` |
| `MCP_SERVER_URL` | MCP server endpoint | No | - |
| `SENTRY_DSN` | Sentry error tracking | No | - |
| `LANGFUSE_ENABLED` | Enable LLM tracing | No | `false` |

## Running Tests

```bash
# Unit tests
make test

# Integration tests (requires OPENAI_API_KEY)
go test -v ./internal/api/handlers -run TestMagda

# Smoke tests
./tests/smoke/run-all.sh http://localhost:8080
```

## Code Style

- Use `gofmt` and `goimports` for formatting
- Follow `golangci-lint` rules
- Write tests for new functionality
- Use conventional commit messages

## Pull Request Process

1. Create feature branch from `main`
2. Make changes and add tests
3. Run `make ci` to verify
4. Create pull request
5. Wait for CI checks to pass
6. Merge to `main`
