#!/bin/bash

# Script to manually trigger the GitHub Actions "Deploy to AWS" workflow.
# Requires GitHub CLI (gh) to be installed and authenticated.
# Usage: ./scripts/deploy.sh [branch] [environment] [skip_build]

set -e

BRANCH=${1:-"main"}
ENVIRONMENT=${2:-"production"}
SKIP_BUILD=${3:-"false"} # Default to false, meaning build will run

REPO="lucaromagnoli/magda-api"
WORKFLOW_NAME="Deploy to AWS"

echo "🚀 Triggering Deployment"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Repository: $REPO"
echo "Workflow:   $WORKFLOW_NAME"
echo "Branch:     $BRANCH"
echo "Environment: $ENVIRONMENT"
echo "Skip Build: $SKIP_BUILD"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo "❌ GitHub CLI (gh) not found. Please install it: https://cli.github.com/"
    exit 1
fi

# Check if authenticated
if ! gh auth status &> /dev/null; then
    echo "❌ Not authenticated with GitHub CLI. Please run 'gh auth login'."
    exit 1
fi

echo "📡 Triggering workflow..."

# Trigger the workflow_dispatch event
gh workflow run "$WORKFLOW_NAME" \
  --repo "$REPO" \
  --ref "$BRANCH" \
  -F environment="$ENVIRONMENT" \
  -F skip_build="$SKIP_BUILD"

echo "✅ Workflow triggered!"
echo ""
echo "🔍 Watch progress:"
echo "   gh run watch --repo $REPO"
echo ""
echo "Or view in browser:"
echo "   gh run list --repo $REPO --limit 1"
echo "   https://github.com/$REPO/actions"
