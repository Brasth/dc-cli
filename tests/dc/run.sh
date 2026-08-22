#!/usr/bin/env bash
# dc meta-binary: verb dispatch + help. No Docker.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
FAILED=0
ran=0
[[ -x "$ROOT/bin/dc" ]] || { echo "missing $ROOT/bin/dc" >&2; exit 1; }

pass() { echo "  ok  $*"; }
fail() { echo "FAIL $*" >&2; FAILED=$((FAILED + 1)); }

run_case() {
  local name="$1"
  shift
  ran=$((ran + 1))
  if ( set -euo pipefail; "$@" ); then
    pass "$name"
  else
    fail "$name"
  fi
}

stub_kit() {
  local dest="$1"
  mkdir -p "$dest"
  cp "$ROOT/bin/dc" "$dest/dc"
  chmod +x "$dest/dc"
  local name
  for name in tui up exec down doctor recover engine try ls open db files forward ps df stats net prune; do
    cat >"$dest/dc-$name" <<EOF
#!/usr/bin/env bash
printf '%s' "STUB-${name}"
if [[ \$# -gt 0 ]]; then
  printf ' %s' "\$@"
fi
printf '\n'
EOF
    chmod +x "$dest/dc-$name"
  done
}

case_help_lists_verbs() {
  local out
  out="$("$ROOT/bin/dc" --help)"
  printf '%s\n' "$out" | grep -q 'dc up'
  printf '%s\n' "$out" | grep -q 'dc exec'
  printf '%s\n' "$out" | grep -q 'dc down'
  printf '%s\n' "$out" | grep -q 'dc-up'
}

case_help_aliases() {
  [[ "$("$ROOT/bin/dc" -h)" == "$("$ROOT/bin/dc" --help)" ]]
  [[ "$("$ROOT/bin/dc" help)" == "$("$ROOT/bin/dc" --help)" ]]
}

case_dispatch_verbs() {
  local kit out
  kit="$(mktemp -d "${TMPDIR:-/tmp}/dc-meta.XXXX")"
  stub_kit "$kit"
  out="$("$kit/dc" up --take-ports ./proj)"
  [[ "$out" == "STUB-up --take-ports ./proj" ]]
  out="$("$kit/dc" exec --service db -- true)"
  [[ "$out" == "STUB-exec --service db -- true" ]]
  out="$("$kit/dc" recover --yes)"
  [[ "$out" == "STUB-recover --yes" ]]
  rm -rf "$kit"
}

case_no_args_opens_board() {
  local kit out
  kit="$(mktemp -d "${TMPDIR:-/tmp}/dc-meta.XXXX")"
  stub_kit "$kit"
  out="$("$kit/dc")"
  [[ "$out" == "STUB-tui" ]]
  rm -rf "$kit"
}

case_flags_and_dir_go_to_tui() {
  local kit out dir
  kit="$(mktemp -d "${TMPDIR:-/tmp}/dc-meta.XXXX")"
  dir="$(mktemp -d "${TMPDIR:-/tmp}/dc-ws.XXXX")"
  stub_kit "$kit"
  out="$("$kit/dc" --all)"
  [[ "$out" == "STUB-tui --all" ]]
  out="$("$kit/dc" "$dir")"
  [[ "$out" == "STUB-tui $dir" ]]
  rm -rf "$kit" "$dir"
}

case_tui_verb() {
  local kit out
  kit="$(mktemp -d "${TMPDIR:-/tmp}/dc-meta.XXXX")"
  stub_kit "$kit"
  out="$("$kit/dc" tui --all)"
  [[ "$out" == "STUB-tui --all" ]]
  rm -rf "$kit"
}

case_unknown_verb() {
  local kit out rc
  kit="$(mktemp -d "${TMPDIR:-/tmp}/dc-meta.XXXX")"
  stub_kit "$kit"
  set +e
  out="$("$kit/dc" frobnicate 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -eq 2 ]]
  printf '%s\n' "$out" | grep -qi 'unknown'
  rm -rf "$kit"
}

case_real_up_help() {
  "$ROOT/bin/dc" up --help | grep -q 'dc-up'
}

case_both_spellings_same_help() {
  local v spaced dashed
  # dc-ps has no --help and talks to Docker; skip it here.
  for v in tui up exec down doctor recover engine try ls open db files forward df stats net prune; do
    spaced="$("$ROOT/bin/dc" "$v" --help)"
    dashed="$("$ROOT/bin/dc-$v" --help)"
    [[ -n "$spaced" && "$spaced" == "$dashed" ]]
  done
}

case_both_spellings_same_argv() {
  local kit spaced dashed
  kit="$(mktemp -d "${TMPDIR:-/tmp}/dc-meta.XXXX")"
  stub_kit "$kit"
  spaced="$("$kit/dc" exec --service db -- true)"
  dashed="$("$kit/dc-exec" --service db -- true)"
  [[ "$spaced" == "$dashed" ]]
  spaced="$("$kit/dc" up --take-ports ./proj)"
  dashed="$("$kit/dc-up" --take-ports ./proj)"
  [[ "$spaced" == "$dashed" ]]
  rm -rf "$kit"
}

echo "== dc meta-binary =="
run_case "help lists verbs and hyphenated alias" case_help_lists_verbs
run_case "--help / -h / help match" case_help_aliases
run_case "verbs dispatch to dc-<verb>" case_dispatch_verbs
run_case "no args opens the board" case_no_args_opens_board
run_case "flags and dirs go to tui" case_flags_and_dir_go_to_tui
run_case "tui verb" case_tui_verb
run_case "unknown verb exits 2" case_unknown_verb
run_case "dc up --help reaches dc-up" case_real_up_help
run_case "dc <verb> --help matches dc-<verb> --help" case_both_spellings_same_help
run_case "dc <verb> argv matches dc-<verb>" case_both_spellings_same_argv

echo
if [[ "$FAILED" -ne 0 ]]; then
  echo "$FAILED/$ran failed"
  exit 1
fi
echo "$ran/$ran passed"
