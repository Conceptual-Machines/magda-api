# MAGDA OpenAI Evals

Evaluation framework for one-shot generation mode using OpenAI's Evals framework.

## Architecture

This evaluation suite calls the deployed MAGDA API and analyzes the musical output using custom scorers.

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

For local development with `AUTH_MODE=none`, no credentials are needed.

For hosted environments, set your API URL:

```bash
export MAGDA_API_URL=http://localhost:8080
```

## Usage

```bash
# Test spread parameter
python test_spread.py --api-url http://localhost:8080

# Run full evals
cd openai_evals
python run_eval.py --mode both --api-url http://localhost:8080

# Run only one-shot mode
python run_eval.py --mode one_shot --api-url http://localhost:8080
```

## How It Works

1. Read test cases from `test_cases.jsonl`
2. Call deployed API with test prompts (one-shot mode)
3. Score outputs using custom music theory scorers
4. Generate report
5. Upload results to OpenAI Evals platform (optional)
