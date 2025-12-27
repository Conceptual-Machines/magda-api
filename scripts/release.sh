#!/bin/bash
# Release script for magda-api
# Usage: ./scripts/release.sh 1.0.0 "Release description"

set -e

VERSION=$1
DESCRIPTION=${2:-"Release v$VERSION"}

if [ -z "$VERSION" ]; then
  echo "❌ Usage: ./scripts/release.sh <version> [description]"
  echo "   Example: ./scripts/release.sh 1.0.0 \"Add streaming support\""
  exit 1
fi

# Validate version format
if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "❌ Error: Version must be in format X.Y.Z (e.g., 1.0.0)"
  exit 1
fi

TAG="v$VERSION"

# Check if tag already exists
if git rev-parse "$TAG" >/dev/null 2>&1; then
  echo "❌ Error: Tag $TAG already exists"
  exit 1
fi

# Make sure we're on main and up to date
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "main" ]; then
  echo "⚠️  Warning: You're on branch '$CURRENT_BRANCH', not 'main'"
  read -p "Continue anyway? (y/N) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 1
  fi
fi

echo "📋 Creating release $TAG..."
echo "   Description: $DESCRIPTION"
echo ""

# Show recent commits
echo "📜 Recent commits:"
git log --oneline -5
echo ""

read -p "Proceed with release? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
  echo "Cancelled."
  exit 0
fi

# Create annotated tag
git tag -a "$TAG" -m "$DESCRIPTION"
echo "✅ Tag $TAG created locally"

# Push tag
git push origin "$TAG"
echo "✅ Tag $TAG pushed to origin"

echo ""
echo "🎉 Release $TAG created!"
echo ""
echo "📦 Docker image will be available at:"
echo "   docker pull lucaromagnoli/magda-api:$VERSION"
echo ""
echo "🔗 View release: https://github.com/Conceptual-Machines/magda-api/releases/tag/$TAG"
echo ""
echo "📝 Next steps:"
echo "   1. Wait for Docker image to build (~2-3 min)"
echo "   2. Deploy to magda-cloud:"
echo "      cd ../magda-cloud && ./scripts/deploy.sh $VERSION"
