#!/bin/bash
#
# Run AIDEAS evaluation tests
# Automatically loads credentials from .env or .envrc
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Find project root (parent of evals directory)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo -e "${GREEN}AIDEAS Evaluation Test Runner${NC}"
echo "================================"
echo ""

# Load environment variables from .env or .envrc
if [ -f "$PROJECT_ROOT/.env" ]; then
    echo -e "${YELLOW}Loading environment from .env${NC}"
    set -a  # automatically export all variables
    source "$PROJECT_ROOT/.env"
    set +a
elif [ -f "$PROJECT_ROOT/.envrc" ]; then
    echo -e "${YELLOW}Loading environment from .envrc${NC}"
    set -a
    source "$PROJECT_ROOT/.envrc"
    set +a
else
    echo -e "${RED}Warning: No .env or .envrc file found${NC}"
    echo "Create one with AIDEAS_EMAIL and AIDEAS_PASSWORD"
fi

# Check if credentials are set
if [ -z "$AIDEAS_EMAIL" ] || [ -z "$AIDEAS_PASSWORD" ]; then
    echo -e "${RED}Error: AIDEAS_EMAIL and AIDEAS_PASSWORD must be set${NC}"
    echo ""
    echo "Either:"
    echo "  1. Create a .env file with these variables"
    echo "  2. Export them manually: export AIDEAS_EMAIL=your@email.com"
    exit 1
fi

# Default API URL
API_URL="${AIDEAS_API_URL:-https://api.musicalaideas.com}"

echo "Email: $AIDEAS_EMAIL"
echo "API URL: $API_URL"
echo ""

# Parse arguments
TEST_TYPE="${1:-spread}"  # Default to spread test

case "$TEST_TYPE" in
    spread)
        echo -e "${GREEN}Running spread parameter test...${NC}"
        python "$SCRIPT_DIR/test_spread.py" --api-url "$API_URL"
        ;;
    continuation)
        echo -e "${GREEN}Running continuation and variations tests...${NC}"
        python "$SCRIPT_DIR/test_continuation_and_variations.py"
        ;;
    timing)
        echo -e "${GREEN}Running timing preservation tests...${NC}"
        python "$SCRIPT_DIR/test_timing_preservation.py"
        ;;
    eval)
        echo -e "${GREEN}Running full evaluation suite...${NC}"
        MODE="${2:-both}"
        python "$SCRIPT_DIR/openai_evals/run_eval.py" --api-url "$API_URL" --mode "$MODE"
        ;;
    all)
        echo -e "${GREEN}Running all tests...${NC}"
        python "$SCRIPT_DIR/test_spread.py" --api-url "$API_URL"
        echo ""
        python "$SCRIPT_DIR/test_continuation_and_variations.py"
        echo ""
        python "$SCRIPT_DIR/test_timing_preservation.py"
        echo ""
        python "$SCRIPT_DIR/openai_evals/run_eval.py" --api-url "$API_URL" --mode both
        ;;
    both)
        echo -e "${GREEN}Running spread test...${NC}"
        python "$SCRIPT_DIR/test_spread.py" --api-url "$API_URL"
        echo ""
        echo -e "${GREEN}Running evaluation suite...${NC}"
        python "$SCRIPT_DIR/openai_evals/run_eval.py" --api-url "$API_URL" --mode both
        ;;
    *)
        echo -e "${RED}Unknown test type: $TEST_TYPE${NC}"
        echo ""
        echo "Usage: $0 [spread|continuation|timing|eval|all|both] [mode]"
        echo ""
        echo "Examples:"
        echo "  $0 spread              # Test spread parameter only"
        echo "  $0 continuation        # Test chord progression continuation"
        echo "  $0 timing              # Test timing preservation"
        echo "  $0 eval one_shot       # Run evals in one_shot mode"
        echo "  $0 eval both           # Run evals in both modes (default)"
        echo "  $0 all                 # Run all tests"
        echo "  $0 both                # Run spread test + full evals"
        exit 1
        ;;
esac
