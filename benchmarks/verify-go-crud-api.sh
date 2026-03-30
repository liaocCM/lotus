#!/bin/bash
# Verification script for go-crud-api benchmark

set +e  # don't exit on error — capture all metrics even if some fail

RESULT="{}"
add_metric() { RESULT=$(echo "$RESULT" | jq --arg k "$1" --arg v "$2" '. + {($k): $v}'); }

# Check handler files exist
if [ ! -f internal/handler/routes.go ]; then
  echo '{"build_success": false, "error": "internal/handler/routes.go not found"}'
  exit 1
fi

# go mod tidy first (Claude may have added deps)
go mod tidy 2>/dev/null || true

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
if go test ./internal/handler/... -v -count=1 > /tmp/test_output_$$.txt 2>&1; then
  add_metric "test_pass" "true"
else
  add_metric "test_pass" "false"
fi

# Also run all tests
if go test ./... -count=1 > /dev/null 2>&1; then
  add_metric "all_tests_pass" "true"
else
  add_metric "all_tests_pass" "false"
fi

# Coverage
COVERAGE=$(go test ./internal/handler/... -cover 2>/dev/null | grep -o '[0-9]*\.[0-9]*%' | head -1 | tr -d '%')
if [ -n "$COVERAGE" ]; then
  add_metric "coverage_pct" "$COVERAGE"
else
  add_metric "coverage_pct" "0"
fi

# Count test functions
TEST_COUNT=0
TABLE_COUNT=0
for f in $(find internal/handler -name '*_test.go' 2>/dev/null); do
  TC=$(grep -c 'func Test' "$f" 2>/dev/null || echo "0")
  TBL=$(grep -cE '^\s*\{' "$f" 2>/dev/null || echo "0")
  TEST_COUNT=$((TEST_COUNT + TC))
  TABLE_COUNT=$((TABLE_COUNT + TBL))
done
add_metric "test_func_count" "$TEST_COUNT"
add_metric "table_case_count" "$TABLE_COUNT"

# Check endpoint coverage
HANDLER_FILES=$(find internal/handler -name '*.go' ! -name '*_test.go' | xargs cat 2>/dev/null)
for endpoint in "GET" "POST" "PUT" "DELETE"; do
  if echo "$HANDLER_FILES" | grep -qi "$endpoint"; then
    add_metric "has_${endpoint,,}" "true"
  else
    add_metric "has_${endpoint,,}" "false"
  fi
done

echo "$RESULT" | jq .
