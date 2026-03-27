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

mkdir -p "$RESULTS_DIR"

echo "=== Benchmark: $SCENARIO / Tier $TIER ==="
echo "Work dir: $WORK_DIR"

# Copy test repo
cp -r "$SCRIPT_DIR/repos/$SCENARIO/"* "$WORK_DIR/"

# Copy prompt
cp "$SCRIPT_DIR/prompts/$SCENARIO.txt" "$WORK_DIR/prompt.txt"

# Copy verification script
cp "$SCRIPT_DIR/verify-$SCENARIO.sh" "$WORK_DIR/verify.sh"

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
    # TODO: install a single relevant skill
    echo "Tier C not yet configured for $SCENARIO"
    exit 1
    ;;
  D)
    TIER_NAME="superpowers"
    # TODO: install superpowers
    echo "Tier D not yet configured — clone superpowers into .claude/"
    exit 1
    ;;
  E)
    TIER_NAME="d-team"
    # TODO: install d-team
    echo "Tier E not yet configured — clone d-team into .claude/"
    exit 1
    ;;
  *)
    echo "Unknown tier: $TIER (use A/B/C/D/E)"
    exit 1
    ;;
esac

echo "Tier: $TIER ($TIER_NAME)"
echo "Running Claude Code..."

cd "$WORK_DIR"

# Run Claude Code and capture timing
START=$(date +%s)
PROMPT=$(cat prompt.txt)

claude -p "$PROMPT" --output-format json > /tmp/claude_output.json 2>&1 || true

END=$(date +%s)
WALL_TIME=$((END - START))

echo "Wall time: ${WALL_TIME}s"
echo "Running verification..."

# Run verification
VERIFY_RESULT=$(bash verify.sh 2>/dev/null || echo '{"error": "verification failed"}')

# Extract token usage from Claude output (if available)
TOKENS_IN=$(jq -r '.usage.input_tokens // 0' /tmp/claude_output.json 2>/dev/null || echo "0")
TOKENS_OUT=$(jq -r '.usage.output_tokens // 0' /tmp/claude_output.json 2>/dev/null || echo "0")

# Build final result
RESULT=$(echo "$VERIFY_RESULT" | jq \
  --arg scenario "$SCENARIO" \
  --arg tier "$TIER" \
  --arg tier_name "$TIER_NAME" \
  --argjson wt "$WALL_TIME" \
  --argjson ti "$TOKENS_IN" \
  --argjson to "$TOKENS_OUT" \
  '. + {
    scenario: $scenario,
    tier: $tier,
    tier_name: $tier_name,
    wall_time_seconds: $wt,
    tokens_in: $ti,
    tokens_out: $to
  }')

# Save result
RESULT_FILE="$RESULTS_DIR/${SCENARIO}_tier${TIER}_$(date +%Y%m%d_%H%M%S).json"
echo "$RESULT" | jq . > "$RESULT_FILE"

echo ""
echo "=== Result ==="
echo "$RESULT" | jq .
echo ""
echo "Saved to: $RESULT_FILE"

# Cleanup
rm -rf "$WORK_DIR"
