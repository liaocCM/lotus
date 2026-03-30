#!/bin/bash
# Verification script for go-slugify benchmark
# Outputs JSON metrics to stdout

set +e  # don't exit on error — capture all metrics

RESULT="{}"
add_metric() { RESULT=$(echo "$RESULT" | jq --arg k "$1" --arg v "$2" '. + {($k): $v}'); }

# Check if files exist
if [ ! -f stringutil/slugify.go ]; then
  echo '{"build_success": false, "error": "stringutil/slugify.go not found"}'
  exit 1
fi

# Build check
if go vet ./... 2>/dev/null; then
  add_metric "build_success" "true"
  add_metric "first_try_compile" "true"
else
  add_metric "build_success" "false"
  add_metric "first_try_compile" "false"
  echo "$RESULT" | jq .
  exit 0
fi

# Test check
if go test ./stringutil/... -v -count=1 > /tmp/test_output.txt 2>&1; then
  add_metric "test_pass" "true"
else
  add_metric "test_pass" "false"
fi

# Coverage
COVERAGE=$(go test ./stringutil/... -cover 2>/dev/null | grep -o '[0-9]*\.[0-9]*%' | head -1 | tr -d '%')
if [ -n "$COVERAGE" ]; then
  add_metric "coverage_pct" "$COVERAGE"
else
  add_metric "coverage_pct" "0"
fi

# Test case count
if [ -f stringutil/slugify_test.go ]; then
  TEST_COUNT=$(grep -c 'func Test' stringutil/slugify_test.go 2>/dev/null || echo "0")
  # Also count table-driven test cases
  TABLE_COUNT=$(grep -cE '^\s*\{' stringutil/slugify_test.go 2>/dev/null || echo "0")
  add_metric "test_func_count" "$TEST_COUNT"
  add_metric "table_case_count" "$TABLE_COUNT"
elif ls stringutil/*_test.go 1>/dev/null 2>&1; then
  TEST_FILE=$(ls stringutil/*_test.go | head -1)
  TEST_COUNT=$(grep -c 'func Test' "$TEST_FILE" 2>/dev/null || echo "0")
  TABLE_COUNT=$(grep -cE '^\s*\{' "$TEST_FILE" 2>/dev/null || echo "0")
  add_metric "test_func_count" "$TEST_COUNT"
  add_metric "table_case_count" "$TABLE_COUNT"
else
  add_metric "test_func_count" "0"
  add_metric "table_case_count" "0"
fi

echo "$RESULT" | jq .
