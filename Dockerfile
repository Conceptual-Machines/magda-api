# Build stage
FROM golang:1.24-alpine AS builder

# Accept build argument for release version
ARG RELEASE_VERSION=dev
ENV RELEASE_VERSION=${RELEASE_VERSION}

WORKDIR /app

# Install build dependencies and templ
# hadolint ignore=DL3018
RUN apk add --no-cache git ca-certificates tzdata && \
    go install github.com/a-h/templ/cmd/templ@v0.3.943

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies (this layer will be cached if go.mod/go.sum don't change)
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Generate templ files and build binary with release version embedded
# Only run templ if .templ files exist
RUN if [ -n "$(find . -name '*.templ' -type f)" ]; then \
      find . -name "*_templ.go" -delete; \
      /go/bin/templ generate; \
    fi && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-X main.releaseVersion=${RELEASE_VERSION} -s -w" \
    -o main .

# Final stage - use alpine for healthcheck support
# hadolint ignore=DL3007
FROM alpine:latest

WORKDIR /

# Install curl for healthchecks and ca-certificates for HTTPS
# hadolint ignore=DL3018
RUN apk add --no-cache ca-certificates curl

# Copy binary from builder
COPY --from=builder /app/main /main

# Copy static files
COPY --from=builder /app/static /static

# Expose port
EXPOSE 8080

# Run
ENTRYPOINT ["/main"]
