#!/usr/bin/env bash
# Locked first-minute copy. Must match assets/branding/copy.md.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
FAILED=0
check() {
  local file="$1" pat="$2"
  if grep -qF "$pat" "$ROOT/$file"; then
    echo "  ok  $file"
  else
    echo "FAIL $file missing: $pat" >&2
    FAILED=$((FAILED + 1))
  fi
}
absent() {
  local file="$1" pat="$2"
  if grep -qF "$pat" "$ROOT/$file"; then
    echo "FAIL $file still has: $pat" >&2
    FAILED=$((FAILED + 1))
  else
    echo "  ok  $file clean"
  fi
}

CURL='curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli'
H1='Dev containers from your terminal.'

check assets/branding/copy.md "$H1"
check assets/branding/copy.md "$CURL"
check site/src/components/Hero.astro "$H1"
check site/src/layouts/Layout.astro 'dc-cli — Dev containers from your terminal'
check README.md "$H1"
check site/src/lib/install.ts "$CURL"
check install.sh 'dc try'
check install.sh 'dc recover'
next_block="$(awk '/^echo "Next:"/,/If Docker/' "$ROOT/install.sh")"
if ! echo "$next_block" | grep -qF 'cd /path/to/your/project'; then
  echo "FAIL install.sh Next block missing: cd /path/to/your/project" >&2
  FAILED=$((FAILED + 1))
elif ! echo "$next_block" | grep -qF 'dc try'; then
  echo "FAIL install.sh Next block missing: dc try" >&2
  FAILED=$((FAILED + 1))
else
  echo "  ok  install.sh Next block"
fi
for pat in 'dc --all' 'dc upgrade' 'dc engine --fix'; do
  if echo "$next_block" | grep -qF "$pat"; then
    echo "FAIL install.sh Next block still has: $pat" >&2
    FAILED=$((FAILED + 1))
  else
    echo "  ok  install.sh Next clean: $pat"
  fi
done
absent site/src/components/Hero.astro 'Host-global helpers around the official Dev Containers CLI.'

echo
if [[ "$FAILED" -gt 0 ]]; then
  echo "FAILED $FAILED"
  exit 1
fi
echo "ok  copy"
