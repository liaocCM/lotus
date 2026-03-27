#!/bin/bash
# Run a single benchmark tier for a scenario
# Usage: ./run-benchmark.sh <scenario> <tier>
# Example: ./run-benchmark.sh go-slugify A
#
# Tiers:
#   A = bare (no config)
#   B = native subagents (CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1)
#   C = single skill
#   D = light bundle (superpowers)
#   E = heavy bundle (d-team)

set -e

SCENARIO=${1:?Usage: run-benchmark.sh <scenario> <tier>}
TIER=${2:?Usage: run-benchmark.sh <scenario> <tier>}
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RESULTS_DIR="$SCRIPT_DIR/results"
WORK_DIR=$(mktemp -d)
CLAUDE_OUTPUT="$WORK_DIR/claude_output.json"

mkdir -p "$RESULTS_DIR"

echo "=== Benchmark: $SCENARIO / Tier $TIER ==="
echo "Work dir: $WORK_DIR"

# Copy test repo
cp -r "$SCRIPT_DIR/repos/$SCENARIO/"* "$WORK_DIR/"

# Copy prompt
cp "$SCRIPT_DIR/prompts/$SCENARIO.txt" "$WORK_DIR/prompt.txt"

# Copy verification script
cp "$SCRIPT_DIR/verify-$SCENARIO.sh" "$WORK_DIR/verify.sh"

# Init git repo (many skills expect git context)
cd "$WORK_DIR"
git init -q
git add -A
git commit -q -m "initial" --allow-empty

# Set up .claude/ config based on tier
mkdir -p "$WORK_DIR/.claude"
case "$TIER" in
  A)
    TIER_NAME="bare"
    # no config
    ;;
  B)
    TIER_NAME="native-subagents"
    # enable agent teams via env
    export CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1
    ;;
  C)
    TIER_NAME="single-skill"
    # Install a minimal Go backend skill
    mkdir -p "$WORK_DIR/.claude/skills/go-backend"
    cat > "$WORK_DIR/.claude/skills/go-backend/SKILL.md" << 'SKILLEOF'
---
name: go-backend
description: Go backend development best practices
---
# Go Backend Development

## Rules
- Write idiomatic Go: short variable names, error returns, no panic
- Use table-driven tests with descriptive subtest names
- Handle edge cases: empty input, unicode, special characters
- Prefer stdlib over external dependencies when possible
- Run `go vet` and `go test -race` before finishing
- Aim for >90% test coverage
SKILLEOF
    ;;
  D)
    TIER_NAME="superpowers"
    echo "Cloning superpowers..."
    git clone --depth 1 https://github.com/obra/superpowers.git /tmp/superpowers-clone 2>/dev/null
    # install skills + hooks + agents into .claude/
    cp -r /tmp/superpowers-clone/skills "$WORK_DIR/.claude/skills" 2>/dev/null || true
    cp -r /tmp/superpowers-clone/hooks "$WORK_DIR/.claude/hooks" 2>/dev/null || true
    cp -r /tmp/superpowers-clone/agents "$WORK_DIR/.claude/agents" 2>/dev/null || true
    cp -r /tmp/superpowers-clone/commands "$WORK_DIR/.claude/commands" 2>/dev/null || true
    rm -rf /tmp/superpowers-clone
    ;;
  E)
    TIER_NAME="d-team"
    echo "Cloning D-Team..."
    git clone --depth 1 https://github.com/chemistrywow31/D-Team.git /tmp/dteam-clone 2>/dev/null
    # install .claude/ contents
    cp -r /tmp/dteam-clone/.claude/* "$WORK_DIR/.claude/" 2>/dev/null || true
    # also copy CLAUDE.md if exists
    cp /tmp/dteam-clone/CLAUDE.md "$WORK_DIR/CLAUDE.md" 2>/dev/null || true
    rm -rf /tmp/dteam-clone
    ;;
  *)
    echo "Unknown tier: $TIER (use A/B/C/D/E)"
    exit 1
    ;;
esac

echo "Tier: $TIER ($TIER_NAME)"
echo "Running Claude Code..."

# Run Claude Code and capture timing
START=$(date +%s)
PROMPT=$(cat prompt.txt)

claude -p "$PROMPT" --output-format json --dangerously-skip-permissions > $CLAUDE_OUTPUT 2>&1 || true

END=$(date +%s)
WALL_TIME=$((END - START))

echo "Wall time: ${WALL_TIME}s"
echo "Running verification..."

# Run verification
VERIFY_RESULT=$(bash verify.sh 2>/dev/null || echo '{"error": "verification failed"}')

# Extract token usage from Claude output
# input_tokens + cache_creation + cache_read = total input context
TOKENS_IN=$(jq -r '(.usage.input_tokens // 0) + (.usage.cache_creation_input_tokens // 0) + (.usage.cache_read_input_tokens // 0)' $CLAUDE_OUTPUT 2>/dev/null || echo "0")
TOKENS_OUT=$(jq -r '.usage.output_tokens // 0' $CLAUDE_OUTPUT 2>/dev/null || echo "0")
COST_USD=$(jq -r '.total_cost_usd // 0' $CLAUDE_OUTPUT 2>/dev/null || echo "0")
NUM_TURNS=$(jq -r '.num_turns // 0' $CLAUDE_OUTPUT 2>/dev/null || echo "0")

# Build final result
RESULT=$(echo "$VERIFY_RESULT" | jq \
  --arg scenario "$SCENARIO" \
  --arg tier "$TIER" \
  --arg tier_name "$TIER_NAME" \
  --argjson wt "$WALL_TIME" \
  --argjson ti "$TOKENS_IN" \
  --argjson to "$TOKENS_OUT" \
  --argjson cost "$COST_USD" \
  --argjson turns "$NUM_TURNS" \
  '. + {
    scenario: $scenario,
    tier: $tier,
    tier_name: $tier_name,
    wall_time_seconds: $wt,
    tokens_in: $ti,
    tokens_out: $to,
    cost_usd: $cost,
    num_turns: $turns
  }')

# Save result
RESULT_FILE="$RESULTS_DIR/${SCENARIO}_tier${TIER}_$(date +%Y%m%d_%H%M%S).json"
echo "$RESULT" | jq . > "$RESULT_FILE"

echo ""
echo "=== Result ==="
echo "$RESULT" | jq .
echo ""
echo "Saved to: $RESULT_FILE"

# Cleanup (skip if DEBUG=1)
if [ "${DEBUG:-0}" = "1" ]; then
  echo "DEBUG: work dir preserved at $WORK_DIR"
  # Also copy claude output to results for inspection
  cp "$CLAUDE_OUTPUT" "$RESULTS_DIR/${SCENARIO}_tier${TIER}_claude_output.json" 2>/dev/null || true
else
  rm -rf "$WORK_DIR"
fi
