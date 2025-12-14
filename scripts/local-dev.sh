#!/bin/bash
# Enable local development with local paths
# Run this to use ../magda-agents and ../grammar-school for faster iteration

set -e

cd "$(dirname "$0")/.."

echo "🔧 Enabling local development mode..."

# Add local replace directives
go mod edit -replace github.com/Conceptual-Machines/magda-agents=../magda-agents/go
go mod edit -replace grammar-school=../grammar-school/go

# Tidy up
go mod tidy

echo "✅ Local development mode enabled!"
echo "   Using local paths: ../magda-agents/go and ../grammar-school/go"
echo ""
echo "To switch back to git dependencies:"
echo "  ./scripts/ci-dev.sh"
