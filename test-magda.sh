#!/bin/bash

# MAGDA Integration Tests Script
# This script loads environment variables and runs MAGDA integration tests

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}🧪 Running MAGDA Integration Tests${NC}"
echo ""

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Always load from .env file first
if [ -f .env ]; then
    echo -e "${YELLOW}📦 Loading environment from .env...${NC}"
    set -a
    while IFS= read -r line || [ -n "$line" ]; do
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ -z "${line// }" ]] && continue
        export "$line" 2>/dev/null || true
    done < .env
    set +a
fi

# Check if OPENAI_API_KEY is set
if [ -z "$OPENAI_API_KEY" ]; then
    echo -e "${RED}❌ ERROR: OPENAI_API_KEY is not set!${NC}"
    echo "   Please ensure .env file contains OPENAI_API_KEY"
    exit 1
fi

echo -e "${GREEN}✓ OPENAI_API_KEY is set (${#OPENAI_API_KEY} chars)${NC}"
echo ""

# Run the tests
echo -e "${GREEN}🚀 Running tests...${NC}"
echo ""

# Run all MAGDA tests
go test -v ./internal/api/handlers -run "TestMagda|TestHealth" "$@"

echo ""
echo -e "${GREEN}✅ Tests completed!${NC}"
