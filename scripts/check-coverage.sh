#!/usr/bin/env bash
# Fails when Go statement coverage drops below the committed baseline.
# Usage: check-coverage.sh <coverage-profile> <baseline-file>
set -euo pipefail

profile="${1:?usage: check-coverage.sh <coverage-profile> <baseline-file>}"
baseline_file="${2:?usage: check-coverage.sh <coverage-profile> <baseline-file>}"

current="$(go tool cover -func="$profile" | awk '/^total:/ { gsub(/%/, "", $3); print $3 }')"
baseline="$(tr -d '[:space:]' < "$baseline_file")"

if [[ -z "$current" ]]; then
  echo "::error::could not parse a total from $profile" >&2
  exit 1
fi

awk -v c="$current" -v b="$baseline" 'BEGIN {
  if (c + 0 < b + 0) {
    printf "::error::backend coverage %.1f%% is below the baseline %.1f%%\n", c, b
    exit 1
  }
  if (c + 0 > b + 0) {
    printf "::notice::backend coverage rose to %.1f%% (baseline %.1f%%) — bump coverage-baseline.txt\n", c, b
    exit 0
  }
  printf "backend coverage %.1f%% matches the baseline\n", c
  exit 0
}'
