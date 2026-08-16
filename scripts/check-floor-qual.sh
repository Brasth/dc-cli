#!/usr/bin/env bash
# Parse DC_CLI_FLOOR_QUAL and compare to floor constants. Fail-closed when constants are set.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ART="$ROOT/docs/qualification/devcontainer-cli-floor.md"
# shellcheck source=/dev/null
source "$ROOT/lib/dc-floor.sh"

min="${DC_DEVCONTAINER_MIN_VERSION:-}"
pin="${DC_DEVCONTAINER_NPM_VERSION:-}"

if [[ -z "$min" && -z "$pin" ]]; then
  echo "floor unpublished (constants empty) — ok"
  exit 0
fi

if [[ ! -f "$ART" ]]; then
  echo "floor constants set but $ART missing" >&2
  exit 1
fi

block="$(awk '/^```yaml/{p=1;next} /^```/{p=0} p' "$ART")"
[[ -n "$block" ]] || { echo "DC_CLI_FLOOR_QUAL yaml block missing" >&2; exit 1; }

val() { printf '%s\n' "$block" | awk -F': *' -v k="$1" '$1==k{sub(/^"/,"",$2);sub(/"$/,"",$2);print $2; exit}'; }

id="$(val id)"
sel="$(val selected_version)"
npin="$(val npm_pin)"
role="$(val owner_role)"

[[ "$id" == "DC_CLI_FLOOR_QUAL" ]] || { echo "id mismatch: $id" >&2; exit 1; }
[[ "$role" == "release_engineer" ]] || { echo "owner_role must be release_engineer" >&2; exit 1; }
[[ "$sel" == "$min" ]] || { echo "selected_version '$sel' != DC_DEVCONTAINER_MIN_VERSION '$min'" >&2; exit 1; }
if [[ -n "$pin" && "$npin" != "$pin" ]]; then
  echo "npm_pin '$npin' != DC_DEVCONTAINER_NPM_VERSION '$pin'" >&2
  exit 1
fi

need=(darwin/arm64 darwin/amd64 linux/arm64 linux/amd64)
for key in "${need[@]}"; do
  os="${key%/*}"
  arch="${key#*/}"
  if ! printf '%s\n' "$block" | grep -q "os: ${os}" || ! printf '%s\n' "$block" | grep -q "arch: ${arch}"; then
    echo "missing platform row $key" >&2
    exit 1
  fi
done
if printf '%s\n' "$block" | grep -q 'result: fail'; then
  echo "non-pass platform row" >&2
  exit 1
fi
echo "DC_CLI_FLOOR_QUAL ok (selected=$sel npm_pin=$npin)"
