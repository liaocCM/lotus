#!/bin/bash
# Verification script for nextjs-app benchmark
set +e

RESULT="{}"
add_metric() { RESULT=$(echo "$RESULT" | jq --arg k "$1" --arg v "$2" '. + {($k): $v}'); }

# Check component exists
if [ -f src/components/DataTable.tsx ]; then
  add_metric "has_component" "true"
else
  add_metric "has_component" "false"
fi

# Check test exists
if [ -f src/components/DataTable.test.tsx ]; then
  add_metric "has_tests" "true"
else
  add_metric "has_tests" "false"
fi

# Install deps
npm install --silent > /dev/null 2>&1

# TypeScript check
if npx tsc --noEmit > /dev/null 2>&1; then
  add_metric "build_success" "true"
  add_metric "first_try_compile" "true"
else
  add_metric "build_success" "false"
  add_metric "first_try_compile" "false"
fi

# Run tests
if npx vitest run > /dev/null 2>&1; then
  add_metric "test_pass" "true"
else
  add_metric "test_pass" "false"
fi

# Count test cases
if [ -f src/components/DataTable.test.tsx ]; then
  TEST_COUNT=$(grep -c "it\|test(" src/components/DataTable.test.tsx 2>/dev/null || echo "0")
  add_metric "test_case_count" "$TEST_COUNT"
else
  add_metric "test_case_count" "0"
fi

# Check features in component
if [ -f src/components/DataTable.tsx ]; then
  COMP=$(cat src/components/DataTable.tsx)
  echo "$COMP" | grep -qi "search\|filter" && add_metric "has_search" "true" || add_metric "has_search" "false"
  echo "$COMP" | grep -qi "sort" && add_metric "has_sort" "true" || add_metric "has_sort" "false"
  echo "$COMP" | grep -qi "page\|pagination" && add_metric "has_pagination" "true" || add_metric "has_pagination" "false"
fi

echo "$RESULT" | jq .
