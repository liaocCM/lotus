#!/bin/bash
# Verification script for go-greenfield benchmark
set +e

RESULT="{}"
add_metric() { RESULT=$(echo "$RESULT" | jq --arg k "$1" --arg v "$2" '. + {($k): $v}'); }

# Check CLAUDE.md exists
if [ -f CLAUDE.md ]; then
  add_metric "has_claude_md" "true"
  CLAUDE_MD_LINES=$(wc -l < CLAUDE.md | tr -d ' ')
  add_metric "claude_md_lines" "$CLAUDE_MD_LINES"
else
  add_metric "has_claude_md" "false"
  add_metric "claude_md_lines" "0"
fi

# Check agents
AGENT_COUNT=0
if [ -d .claude/agents ]; then
  AGENT_COUNT=$(find .claude/agents -name '*.md' | wc -l | tr -d ' ')
fi
add_metric "agent_count" "$AGENT_COUNT"

# Check skills
SKILL_COUNT=0
if [ -d .claude/skills ]; then
  SKILL_COUNT=$(find .claude/skills -type d -mindepth 1 | wc -l | tr -d ' ')
  if [ "$SKILL_COUNT" = "0" ]; then
    SKILL_COUNT=$(find .claude/skills -name '*.md' | wc -l | tr -d ' ')
  fi
fi
add_metric "skill_count" "$SKILL_COUNT"

# Check rules
RULE_COUNT=0
if [ -d .claude/rules ]; then
  RULE_COUNT=$(find .claude/rules -name '*.md' | wc -l | tr -d ' ')
fi
add_metric "rule_count" "$RULE_COUNT"

# Overall success: has CLAUDE.md + at least 3 agents + at least 2 skills
if [ "$AGENT_COUNT" -ge 3 ] 2>/dev/null && [ "$SKILL_COUNT" -ge 2 ] 2>/dev/null && [ -f CLAUDE.md ]; then
  add_metric "build_success" "true"
  add_metric "test_pass" "true"
else
  add_metric "build_success" "true"
  add_metric "test_pass" "false"
fi

echo "$RESULT" | jq .
