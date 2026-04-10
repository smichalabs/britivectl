#!/usr/bin/env bash
set -euo pipefail

MSG=$(cat "$1")

if ! echo "$MSG" | grep -qE "^(feat|fix|chore|docs|refactor|test|ci|sec|perf)(\(.+\))?: .+"; then
  echo "ERROR -- commit message must match conventional format"
  echo "  Pattern: (feat|fix|chore|docs|refactor|test|ci|sec|perf)(scope)?: description"
  echo "  Got: $MSG"
  exit 1
fi
