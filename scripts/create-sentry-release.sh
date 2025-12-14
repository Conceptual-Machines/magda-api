#!/bin/bash

# Create Sentry Release Manually
# Usage: ./scripts/create-sentry-release.sh [version]
# Example: ./scripts/create-sentry-release.sh test-v1

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if sentry-cli is installed
if ! command -v sentry-cli &> /dev/null; then
    echo -e "${RED}❌ sentry-cli not found${NC}"
    echo ""
    echo "Install it with:"
    echo "  brew install getsentry/tools/sentry-cli"
    echo "  # or"
    echo "  curl -sL https://sentry.io/get-cli/ | bash"
    exit 1
fi

# Check for required environment variables
if [ -z "$SENTRY_AUTH_TOKEN" ]; then
    echo -e "${RED}❌ SENTRY_AUTH_TOKEN not set${NC}"
    echo ""
    echo "Set it with:"
    echo "  export SENTRY_AUTH_TOKEN=<your-token>"
    exit 1
fi

if [ -z "$SENTRY_ORG" ]; then
    echo -e "${RED}❌ SENTRY_ORG not set${NC}"
    echo ""
    echo "Set it with:"
    echo "  export SENTRY_ORG=<your-org-slug>"
    exit 1
fi

# Configuration
PROJECT="magda-api"
ENVIRONMENT="production"

# Get version
if [ -n "$1" ]; then
    VERSION="$1"
else
    # Use git SHA (first 8 chars)
    VERSION=$(git rev-parse --short=8 HEAD)
fi

RELEASE="magda-api@$VERSION"

echo -e "${GREEN}🚀 Creating Sentry Release${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Organization: $SENTRY_ORG"
echo "Project:      $PROJECT"
echo "Release:      $RELEASE"
echo "Environment:  $ENVIRONMENT"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Create the release
echo -e "${YELLOW}📦 Creating release...${NC}"
sentry-cli releases new "$RELEASE" --project "$PROJECT"

# Associate commits
echo -e "${YELLOW}🔗 Linking commits...${NC}"
sentry-cli releases set-commits "$RELEASE" --auto --project "$PROJECT"

# Finalize the release
echo -e "${YELLOW}✅ Finalizing release...${NC}"
sentry-cli releases finalize "$RELEASE" --project "$PROJECT"

# Mark as deployed
echo -e "${YELLOW}🚀 Marking as deployed to $ENVIRONMENT...${NC}"
sentry-cli releases deploys "$RELEASE" new -e "$ENVIRONMENT" --project "$PROJECT"

echo ""
echo -e "${GREEN}✅ Release created successfully!${NC}"
echo ""
echo "🔍 View in Sentry:"
echo "   https://sentry.io/organizations/$SENTRY_ORG/projects/$PROJECT/releases/$RELEASE/"
echo ""
echo "📊 Check errors by this release:"
echo "   https://sentry.io/organizations/$SENTRY_ORG/issues/?project=$PROJECT&query=release:$RELEASE"
