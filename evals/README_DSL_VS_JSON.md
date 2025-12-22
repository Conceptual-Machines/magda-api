# DSL vs JSON Schema Comparison Test

## Overview

This test compares the performance of DSL (CFG grammar) vs JSON Schema output formats for musical composition generation.

## What It Tests

1. **Output Token Count**: Compares how many tokens each format uses
2. **Latency**: Measures total response time for both formats
3. **First Token Time**: For streaming, measures time to first token
4. **Output Quality**: Verifies both formats produce valid, similar results

## Usage

```bash
# Set environment variables
export MAGDA_API_URL="http://localhost:8080"  # or your API URL

# Run the test
python3 evals/test_dsl_vs_json.py
```

## Expected Results

Based on our analysis, you should see:

- **Token Savings**: DSL should use ~60-70% fewer output tokens
- **Latency Improvement**: DSL should be faster, especially for streaming
- **Output Quality**: Both should produce equivalent musical output

## Output

The test will show:
- Side-by-side comparison of metrics
- Token savings percentage
- Latency improvement percentage
- First token time comparison (for streaming)
- Output quality verification

## Notes

- Tests run sequentially with a 2-second delay between formats
- Uses the same input for both formats to ensure fair comparison
- Non-streaming and streaming tests are run separately
