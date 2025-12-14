#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧪 Testing Streaming API Endpoint${NC}"
echo ""

# Test data
TEST_PROMPT="Generate a simple C major chord progression"

# API endpoint (use localhost for local testing, or production URL)
if [ -z "$API_URL" ]; then
    API_URL="https://api.musicalaideas.com"
fi

echo -e "${YELLOW}📡 Testing: ${API_URL}/api/generate/stream${NC}"
echo ""

# Create request payload
REQUEST_PAYLOAD=$(cat <<EOF
{
  "model": "gpt-4.1",
  "input_array": [
    {
      "type": "message",
      "role": "user",
      "content": "$TEST_PROMPT"
    }
  ]
}
EOF
)

echo -e "${BLUE}📤 Sending request...${NC}"
echo ""

# Make streaming request and parse SSE events
curl -N -X POST "${API_URL}/api/generate/stream" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d "$REQUEST_PAYLOAD" \
  2>/dev/null | while IFS= read -r line; do
    # Skip empty lines
    if [ -z "$line" ]; then
        continue
    fi

    # Parse SSE data lines
    if [[ $line == data:* ]]; then
        # Remove "data: " prefix
        json_data="${line#data: }"

        # Extract event type and message using jq if available
        if command -v jq &> /dev/null; then
            event_type=$(echo "$json_data" | jq -r '.type // "unknown"')
            message=$(echo "$json_data" | jq -r '.message // ""')

            case "$event_type" in
                "start")
                    echo -e "${GREEN}🚀 $message${NC}"
                    ;;
                "processing")
                    echo -e "${BLUE}⚙️  $message${NC}"
                    ;;
                "mcp_enabled")
                    echo -e "${YELLOW}🎵 $message${NC}"
                    ;;
                "heartbeat")
                    events=$(echo "$json_data" | jq -r '.data.events_received // 0')
                    elapsed=$(echo "$json_data" | jq -r '.data.elapsed_seconds // 0')
                    echo -e "${YELLOW}💓 Heartbeat - Events: $events, Elapsed: ${elapsed}s${NC}"
                    ;;
                "output_started")
                    echo -e "${GREEN}📝 $message${NC}"
                    ;;
                "output_progress")
                    event_count=$(echo "$json_data" | jq -r '.data.event_count // 0')
                    echo -e "${BLUE}📊 $message (event #$event_count)${NC}"
                    ;;
                "analyzing")
                    echo -e "${BLUE}🔍 $message${NC}"
                    ;;
                "mcp_used")
                    tools=$(echo "$json_data" | jq -r '.data.tools // [] | join(", ")')
                    calls=$(echo "$json_data" | jq -r '.data.calls // 0')
                    echo -e "${YELLOW}🎵 MCP Used - Tools: [$tools], Calls: $calls${NC}"
                    ;;
                "complete")
                    choices=$(echo "$json_data" | jq -r '.data.output_parsed.choices // [] | length')
                    echo -e "${GREEN}✅ Complete - Generated $choices choices${NC}"
                    ;;
                "done")
                    echo -e "${GREEN}🎉 Request completed!${NC}"
                    request_id=$(echo "$json_data" | jq -r '.data.request_id // "unknown"')
                    echo -e "${BLUE}📋 Request ID: $request_id${NC}"

                    # Show token usage
                    input_tokens=$(echo "$json_data" | jq -r '.data.usage.input_tokens // 0')
                    output_tokens=$(echo "$json_data" | jq -r '.data.usage.output_tokens // 0')
                    total_tokens=$(echo "$json_data" | jq -r '.data.usage.total_tokens // 0')
                    echo -e "${BLUE}🎫 Tokens - Input: $input_tokens, Output: $output_tokens, Total: $total_tokens${NC}"
                    ;;
                "error")
                    echo -e "${RED}❌ Error: $message${NC}"
                    ;;
                *)
                    echo -e "${YELLOW}📦 $event_type: $message${NC}"
                    ;;
            esac
        else
            # Fallback if jq is not available
            echo "$json_data"
        fi
    fi
done

echo ""
echo -e "${GREEN}✅ Streaming test completed${NC}"
