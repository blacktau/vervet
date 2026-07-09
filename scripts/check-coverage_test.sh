#!/usr/bin/env bash
# Self-check for check-coverage.sh. Run: bash scripts/check-coverage_test.sh
set -uo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
script="$here/check-coverage.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# A minimal coverage profile whose `go tool cover -func` total is 66.1%.
# We stub `go` on PATH instead of generating a real profile, so this test
# stays fast and needs no compilation.
mkdir -p "$tmp/bin"
cat > "$tmp/bin/go" <<'STUB'
#!/usr/bin/env bash
# Emulates: go tool cover -func=<profile>
# Echoes the total line, reading the percentage from the profile file itself.
pct="$(cat "${3#-func=}")"
printf 'total:\t\t\t\t(statements)\t\t%s%%\n' "$pct"
STUB
chmod +x "$tmp/bin/go"
export PATH="$tmp/bin:$PATH"

fail=0
check() {
  local name="$1" pct="$2" baseline="$3" want_rc="$4"
  printf '%s' "$pct" > "$tmp/profile"
  printf '%s' "$baseline" > "$tmp/baseline"
  "$script" "$tmp/profile" "$tmp/baseline" >/dev/null 2>&1
  local got_rc=$?
  if [[ "$got_rc" != "$want_rc" ]]; then
    echo "FAIL: $name — coverage=$pct baseline=$baseline: want rc=$want_rc, got rc=$got_rc"
    fail=1
  else
    echo "ok: $name"
  fi
}

check "below baseline fails"      "65.0" "66.1" 1
check "equal to baseline passes"  "66.1" "66.1" 0
check "above baseline passes"     "70.2" "66.1" 0
check "just below fails"          "66.0" "66.1" 1
check "integer baseline passes"   "80.0" "80"   0

exit "$fail"
