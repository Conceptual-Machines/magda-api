#!/bin/bash
# Switch back to git dependencies (for CI/testing)
# Removes local replace directives

set -e

cd "$(dirname "$0")/.."

echo "🔄 Switching to git dependencies (CI mode)..."

# Remove local replace directives
go mod edit -dropreplace github.com/Conceptual-Machines/magda-agents
go mod edit -dropreplace grammar-school

# Tidy up - this will fetch from git
go mod tidy

echo "✅ Using git dependencies (CI mode)"
echo "   Dependencies will be fetched from GitHub"
echo ""
echo "To switch back to local development:"
echo "  ./scripts/local-dev.sh"
