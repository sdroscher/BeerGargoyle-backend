#!/bin/bash
TOTAL_COVERAGE=$(go tool cover -func=cover.out | grep total | grep -Eo '[0-9]+\.[0-9]+')
echo "Current test coverage : $TOTAL_COVERAGE%"
if [ "$(($(echo "$TOTAL_COVERAGE $TESTCOVERAGE_THRESHOLD" | awk '{print ($1 >= $2)}')))" -gt 0 ]; then
  echo "OK"
else
  echo "\033[0;31mFAIL: Current test coverage is below threshold of ${TESTCOVERAGE_THRESHOLD}%. Please add more unit tests or adjust threshold to a lower value.\033[0m"
  exit 1
fi
