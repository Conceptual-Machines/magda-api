.PHONY: run build test clean install dev lint fmt tidy check ci smoke-test

# Default target
all: tidy fmt check build

# Development
dev:
	go run main.go

# Build
build:
	go build -o bin/magda-api main.go

# Run
run: build
	./bin/magda-api

# Test
test:
	go test -v -race -coverprofile=coverage.out ./...

# Test with coverage report
test-coverage: test
	go tool cover -html=coverage.out -o coverage.html

# Smoke tests (requires server running)
smoke-test:
	./tests/smoke/run-all.sh http://localhost:8080

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Lint code
lint:
	golangci-lint run

# Lint code in Docker (matches CI environment)
lint-docker:
	docker run --rm -v $$(pwd):/app -w /app golangci/golangci-lint:v1.64.8 golangci-lint run --config=golangci.yml

# Run all checks
check: lint test

# CI pipeline
ci: tidy fmt check build

# Install dependencies
install:
	go mod download
	go mod tidy

# Tidy dependencies
tidy:
	go mod tidy

# Install linting tools
install-tools:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Clean
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean

# Docker
docker-build:
	docker build -t magda-api:latest .

docker-run:
	docker run -p 8080:8080 --env-file .env magda-api:latest

# Docker Compose
dc-up:
	docker compose up -d

dc-down:
	docker compose down

dc-logs:
	docker compose logs -f

dc-rebuild:
	docker compose up --build -d

# Development setup
setup: install-tools install tidy fmt
	@echo "✅ Development environment ready!"
