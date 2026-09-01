#!/usr/bin/env bash
# Public GitHub funnel snapshot for Brasth/dc-cli.
# No tokens, no PII — counts only (release downloads, open issues).
# Cloudflare UV / /play/ pageviews are dashboard-only; this script cannot print them.
set -euo pipefail

REPO="Brasth/dc-cli"
API="https://api.github.com"
UA="dc-cli-funnel-snapshot"

die() {
  echo "funnel-snapshot: $*" >&2
  exit 1
}

fetch() {
  local url="$1"
  local body
  if ! body="$(curl -fsSL -H "Accept: application/vnd.github+json" -H "User-Agent: ${UA}" "$url")"; then
    die "curl failed for ${url}"
  fi
  printf '%s\n' "$body"
}

json() {
  python3 -c "$1"
}

command -v curl >/dev/null 2>&1 || die "curl is required"
command -v python3 >/dev/null 2>&1 || die "python3 is required"

release_json="$(fetch "${API}/repos/${REPO}/releases/latest")"
open_json="$(fetch "${API}/search/issues?q=repo:${REPO}+is:issue+state:open")"
first_run_json="$(fetch "${API}/search/issues?q=repo:${REPO}+is:issue+state:open+label:first-run")"

tag="$(printf '%s\n' "$release_json" | json 'import json,sys; d=json.load(sys.stdin); print(d.get("tag_name") or "")')"
[[ -n "$tag" ]] || die "latest release JSON had no tag_name"

echo "date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "tag: ${tag}"
echo "assets:"
printf '%s\n' "$release_json" | json '
import json, sys
d = json.load(sys.stdin)
assets = d.get("assets") or []
if not assets:
    print("  (none)")
    raise SystemExit(0)
for a in assets:
    name = a.get("name") or "?"
    count = a.get("download_count", 0)
    print(f"  {name}  {count}")
'

open_count="$(printf '%s\n' "$open_json" | json 'import json,sys; d=json.load(sys.stdin); print(d["total_count"] if "total_count" in d else "")')"
echo "open_issues: ${open_count:-n/a}"

# first-run label count only when the search payload includes total_count
first_run_count="$(printf '%s\n' "$first_run_json" | json 'import json,sys; d=json.load(sys.stdin); print(d["total_count"] if "total_count" in d else "")')"
if [[ -n "$first_run_count" ]]; then
  echo "first_run_open: ${first_run_count}"
fi
