# Project Setup Guide

## Branch Protection Setup

To protect the `main` branch and enforce CI checks, follow these steps:

### 1. Enable Branch Protection Rules

1. Go to your repository on GitHub: `https://github.com/lucaromagnoli/aideas-api`
2. Navigate to **Settings** → **Branches**
3. Click **Add rule**
4. Configure the following settings:

#### Branch name pattern:
```
main
```

#### Required status checks:
- ✅ **Require status checks to pass before merging**
- ✅ **Require branches to be up to date before merging**
- Enable these specific status checks:
  - `test` (Test & Lint)
  - `build` (Build)

#### Optional settings to enable:
- ✅ **Require pull request reviews before merging**
  - Required number of reviewers: `1`
  - ✅ Dismiss stale pull request approvals when new commits are pushed
  - ✅ Require review from code owners
- ✅ **Restrict pushes that create files**
- ✅ **Restrict pushes to matching branches**
- ✅ **Allow force pushes** → **No one**
- ✅ **Do not allow bypassing the above settings**

### 2. Repository Rules (Alternative Method)

If Branch Protection Rules are not sufficient, check if GitHub has "Repository Rules" available:

1. Go to **Settings** → **Rules**
2. Click **Add rule**
3. Choose **Branch protection rule**
4. Configure similar settings as above

## Development Workflow

### Quick Start

```bash
# Clone and setup
git clone https://github.com/lucaromagnoli/aideas-api.git
cd aideas-api

# Setup development environment
make setup

# Run development server
make dev
```

### Available Commands

```bash
# Development
make dev              # Run development server
make setup           # Setup development environment

# Code Quality
make fmt             # Format code
make lint            # Run linters
make tidy            # Tidy dependencies
make check           # Run all checks (lint + test)

# Testing
make test            # Run tests
make test-coverage   # Run tests with coverage report

# Building
make build           # Build binary
make clean           # Clean build artifacts

# Docker
make docker-build    # Build Docker image
make docker-run      # Run Docker container

# Docker Compose
make dc-up           # Start services
make dc-down         # Stop services
make dc-logs         # View logs
make dc-rebuild      # Rebuild and restart

# All-in-one
make all             # tidy + fmt + check + build
make ci              # CI pipeline (tidy + fmt + check + build)
```

## Environment Variables

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

Required variables:
- `OPENAI_API_KEY`: Your OpenAI API key
- `DATABASE_URL`: PostgreSQL connection string
- `JWT_SECRET`: Secret key for JWT tokens
- `MCP_SERVER_URL`: MCP server URL (leave empty to disable)
- `ENVIRONMENT`: deployment environment

## Available GitHub Workflows

The CI/CD pipeline includes:

1. **Test & Lint Job**:
   - Runs tests with PostgreSQL service
   - Executes golangci-lint with comprehensive rules
   - Generates code coverage reports
   - Uploads coverage to Codecov

2. **Build Job**:
   - Builds the binary artifact
   - Uploads artifacts for releases
   - Runs after successful test pipeline

3. **Docker Job** (main branch only):
   - Builds and pushes Docker images
   - Uses Docker Hub secrets
   - Implements caching for faster builds

## Pull Request Process

1. Create feature branch from `main`
2. Make changes and commit
3. Push feature branch
4. Create pull request
5. All CI checks must pass:
   - Linting ✅
   - Tests ✅
   - Build ✅
6. Code review approval required
7. Merge to `main` (protected branch)

## Code Style Guidelines

- Use `gofmt` and `goimports` for formatting
- Follow `golangci-lint` rules
- Write tests for new functionality
- Document public APIs with Go comments
- Use conventional commit messages

## Security

- All environment variables should be stored as GitHub secrets
- Never commit API keys or sensitive data
- Use `.env.example` for template variables
- Require authenticated reviews for security-sensitive changes

## Monitoring

- Check GitHub Actions tab for CI/CD status
- Monitor Codecov coverage reports
- Review security alerts in GitHub Security tab
- Monitor Docker image scans
