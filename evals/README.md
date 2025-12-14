# AIDEAS OpenAI Evals

Evaluation framework for one-shot generation mode using OpenAI's Evals framework.

## Architecture

This evaluation suite calls the deployed AIDEAS API and analyzes the musical output using custom scorers.

### Evaluation Criteria

1. **Chord Detection** - Identify chords at specific beats
2. **Content Type Classification** - Distinguish chords-only, chords+melody, chords+bass
3. **Key/Scale Validation** - Verify output matches requested key/scale
4. **Timing Quality** - Assess natural vs mechanical feel (advanced)

### Structure

```
evals/
├── openai_evals/
│   ├── test_cases.jsonl       # Test prompts and expected outputs
│   ├── scorers/               # Custom scoring functions
│   │   ├── chord_scorer.py
│   │   ├── key_scorer.py
│   │   └── content_scorer.py
│   └── run_eval.py            # Main evaluation runner
└── README.md
```

## Setup

```bash
cd evals

# Install dependencies (use uv for faster installation)
uv venv
source .venv/bin/activate
uv pip install -r requirements.txt

# Or use regular pip
pip install -r requirements.txt
```

## Authentication

The API requires authentication. Set your credentials:

```bash
# Option 1: Use .env file in project root (recommended)
# Add to magda-api/.env:
AIDEAS_EMAIL=your@email.com
AIDEAS_PASSWORD=yourpassword

# Option 2: Export environment variables
export AIDEAS_EMAIL=your@email.com
export AIDEAS_PASSWORD=yourpassword

# Option 3: Pass as command-line arguments
python test_spread.py --email your@email.com --password yourpassword
```

## Usage

### Quick Test Script (Recommended)

Use the convenience script that automatically loads credentials from `.env` or `.envrc`:

```bash
# Test spread parameter only
./run_tests.sh spread

# Run full evaluation suite
./run_tests.sh eval

# Run both spread test and full eval
./run_tests.sh both

# Test against different API URL
AIDEAS_API_URL=http://localhost:8080 ./run_tests.sh spread
```

### Manual Usage

```bash
# Test spread parameter
python test_spread.py --api-url https://api.musicalaideas.com

# Run full evals (both modes)
cd openai_evals
python run_eval.py --mode both --api-url https://api.musicalaideas.com

# Run only one-shot mode
python run_eval.py --mode one_shot --api-url https://api.musicalaideas.com
```

## How It Works

1. Read test cases from `test_cases.jsonl`
2. Call deployed API with test prompts (one-shot mode)
3. Score outputs using custom music theory scorers
4. Generate report
5. Upload results to OpenAI Evals platform (optional)
