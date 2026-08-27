# Wedge First-Run Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A stranger with Docker already installed can copy the advertised curl, type `dc`, start this folder (or a `dc-try` sandbox), and get a shell — and the website sells that same win.

**Architecture:** Copy and first-minute path only. Helpers still wrap official `@devcontainers/cli`. Site layout and install curl stay. Recover stays the mutate door for an existing engine. Empty-machine install, desktop app, and spec rewrite stay out.

**Tech Stack:** bash helpers, Go `dc-tui` (bubbletea), Astro site, existing `tests/*/run.sh` harness.

**Spec:** `docs/superpowers/specs/2026-08-27-wedge-first-run-design.md`  
**Repo index:** `plans/260827-0927-wedge-first-run/plan.md`

## Global Constraints

- Advertised install stays exactly: `curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli`
- Never edit project `.devcontainer`
- `install.sh` with no flags stays wrappers-only (`WITH_CLI=0`)
- `dc-up --yes` on `kind=none` does **not** start a sandbox
- `dc-try` still writes override under XDG state only; no ports in generated config
- No new npm/Go modules
- Conventional commits (`feat`, `fix`, `docs`, `test`)
- Doctor stays read-only; mutate is `dc-recover` / confirmed TUI / TTY `dc-up` → try

---

## File map

| File | Role |
|---|---|
| `assets/branding/copy.md` | Locked marketing strings |
| `site/src/components/Hero.astro` | H1, sub, trust line |
| `site/src/layouts/Layout.astro` | default title + description |
| `site/src/components/PainPayoff.astro` | why-install copy |
| `site/src/components/Install.astro` | install H2 + note |
| `site/src/components/Bento.astro` | exec title |
| `site/src/components/ClipStrip.astro` | exec clip title |
| `site/src/pages/guide/index.astro` | guide card order |
| `site/src/content/guides/kind.md` | none path wording |
| `site/src/content/guides/install.md` | H1 align with “Then dc” |
| `README.md` | lead + none path |
| `launch/product-hunt.md` `launch/hacker-news.md` `launch/x-thread.md` | same H1 |
| `install.sh` | post-install Next block |
| `bin/dc-up` | TTY offer `dc-try` on `kind=none` |
| `cmd/dc-tui/view.go` | sandbox confirm string |
| `lib/dc-recover.sh` `bin/dc-recover` | apply `try_sandbox` |
| `skill/SKILL.md` | first-minute lead |
| `tests/copy/run.sh` | locked-string contract |
| `tests/try/run.sh` | existing try gates + new TTY/up cases |
| `tests/recover/run.sh` | `kind_none` applyAllowed |
| `.github/workflows/ci.yml` | run copy + try suites |

---

### Task 1: Locked copy contract test

**Files:**
- Create: `tests/copy/run.sh`
- Modify: `.github/workflows/ci.yml` (add the copy job next to other bash suites)
- Test: `tests/copy/run.sh`

**Interfaces:**
- Consumes: none
- Produces: grep contract for H1 `Dev containers from your terminal.` and advertised curl. First commit may assert **current** strings so the test is red after Task 2, or assert the **new** strings and stay red until Task 2. Prefer asserting the **new** locked strings from the spec (TDD).

- [ ] **Step 1: Write the failing copy contract**

Create `tests/copy/run.sh`:

```bash
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
absent site/src/components/Hero.astro 'Host-global helpers around the official Dev Containers CLI.'

echo
if [[ "$FAILED" -gt 0 ]]; then
  echo "FAILED $FAILED"
  exit 1
fi
echo "ok  copy"
```

`chmod +x tests/copy/run.sh`

- [ ] **Step 2: Run it to verify it fails**

Run: `bash tests/copy/run.sh`  
Expected: FAIL on Hero / Layout / branding (old H1 still present)

- [ ] **Step 3: Wire CI (suite may fail until Task 2)**

In `.github/workflows/ci.yml`, after `installer gates`, add:

```yaml
      - name: copy contract
        run: bash tests/copy/run.sh
      - name: try gates
        run: bash tests/try/run.sh
```

Also add to the `bash -n` list if missing: `lib/dc-try.sh`, `bin/dc-try`, `tests/try/run.sh`, `tests/copy/run.sh`.

- [ ] **Step 4: Commit**

```bash
git add tests/copy/run.sh .github/workflows/ci.yml
git commit -m "test: lock first-minute copy and wire dc-try into CI"
```

---

### Task 2: Website + branding + README + launch copy

**Files:**
- Modify: `assets/branding/copy.md`
- Modify: `site/src/components/Hero.astro`
- Modify: `site/src/layouts/Layout.astro`
- Modify: `site/src/components/PainPayoff.astro`
- Modify: `site/src/components/Install.astro`
- Modify: `site/src/components/Bento.astro`
- Modify: `site/src/components/ClipStrip.astro`
- Modify: `site/src/pages/guide/index.astro`
- Modify: `site/src/content/guides/install.md`
- Modify: `site/src/content/guides/kind.md`
- Modify: `README.md`
- Modify: `launch/product-hunt.md`
- Modify: `launch/hacker-news.md`
- Modify: `launch/x-thread.md`

**Interfaces:**
- Consumes: locked strings from Task 1 / spec
- Produces: homepage + README that pass `tests/copy/run.sh`

- [ ] **Step 1: Rewrite `assets/branding/copy.md`**

Replace H1/Sub with:

```markdown
# Locked copy

H1 (hero, brand-first with logo mark):
Dev containers from your terminal.

Sub:
`dc up` starts this folder. No config? `dc try`. No VS Code. Never edits `.devcontainer`.

Trust (smaller, under sub):
Wraps the official Dev Containers CLI.

Primary CTA:
Copy install

Secondary CTA:
Watch (product video) · GitHub

Install (must match README exactly):
```
curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli
# --with-cli = official standalone installer (not npm)
```

After:
`source ~/.zshrc` or `~/.bashrc`, then `dc`

Footer:
MIT. Canvilled / Brasth.
```

- [ ] **Step 2: Hero**

In `site/src/components/Hero.astro`, change the `h1` text to `Dev containers from your terminal.`

Replace the sub `<p>` with:

```astro
      <p class="mt-4 max-w-xl text-[15px] leading-relaxed text-mute md:text-base">
        <span class="font-mono text-ink/80">dc up</span> starts this folder. No config?
        <span class="font-mono text-ink/80">dc try</span>. No VS Code. Never edits
        <span class="font-mono text-ink/80">.devcontainer</span>.
      </p>
      <p class="mt-2 max-w-xl font-mono text-[12px] text-mute">
        Wraps the official Dev Containers CLI.
      </p>
```

- [ ] **Step 3: Layout SEO**

In `site/src/layouts/Layout.astro`:

```ts
const title = Astro.props.title ?? 'dc-cli — Dev containers from your terminal';
const description =
  Astro.props.description ??
  'dc up starts this folder. dc exec is the shell. No config? dc try. No VS Code required. Never edits project .devcontainer.';
```

- [ ] **Step 4: Pain, install, bento, clips**

`PainPayoff.astro` H2: `Same folder. A shell in one minute.`  
Keep the official-CLI / never-edits sentence under it.  
“with dc” list:

- `dc` — board for this folder
- `dc up` / `dc try` — start (sandbox if no config)
- `dc exec` — shell, never raw docker exec
- `dc recover` — one next step when Docker blocks you

`Install.astro` H2: `One curl. Then dc.`  
Note paragraph: `This line is helpers + official standalone CLI if missing (--with-cli). Then dc in this folder.`

`Bento.astro` exec tile `title`: `Shell in the app.` (body still says never raw docker exec)

`ClipStrip.astro` exec `title`: `Shell in the app`

- [ ] **Step 5: Guides + README**

`site/src/pages/guide/index.astro` order array:

```ts
const order = ['install', 'try', 'tui', 'stuck', 'kind', 'doctor', 'no-docker', 'ports', 'disk'];
```

`install.md` h1: `One curl. Then dc.`

`kind.md` FAQ “What if none?”: `dc try` is the start path; on a TTY `dc up` offers that sandbox. Agents use `dc-try --yes`.

`README.md` opening stays the H1. Change the detect line to:

```markdown
`.devcontainer` → official CLI. Else a root compose file → Compose. Else `dc try` (sandbox). `dc up` on a TTY offers that sandbox; non-TTY prints the `dc-try` hint.
```

- [ ] **Step 6: Launch kits**

`launch/product-hunt.md` tagline already matches. Maker comment first sentence:

```
I built dc-cli so I could start a folder from the terminal: dc up, or dc try when there is no config, then dc exec. No VS Code. Wraps the official Dev Containers CLI. Never edits .devcontainer.
```

`launch/hacker-news.md` body: lead with `dc up` / `dc try` / `dc exec`, then board / recover. Keep the honest limitation: folder-scoped wrappers, not a new engine.

`launch/x-thread.md` pin + 1/7 + 3/7: same H1. Tweet 3 includes `dc try`.

- [ ] **Step 7: Verify copy + site build**

Run:

```bash
bash tests/copy/run.sh
```

Expected: `ok  copy`

If `site/node_modules` exists:

```bash
cd site && npm run build
```

Expected: Astro build exit 0.

- [ ] **Step 8: Commit**

```bash
git add assets/branding/copy.md site README.md launch
git commit -m "docs: lead site and README with first-minute win"
```

---

### Task 3: Install Next block

**Files:**
- Modify: `install.sh` (the `echo "Next:"` block near the end)
- Test: `tests/install/run.sh` if it greps Next lines; otherwise add a small case or extend `tests/copy/run.sh`

**Interfaces:**
- Consumes: advertised curl unchanged
- Produces: post-install text that names `dc`, `dc try`, `dc recover` first

- [ ] **Step 1: Add a grep in `tests/copy/run.sh`**

```bash
check install.sh 'dc try'
check install.sh 'dc recover'
```

Run `bash tests/copy/run.sh` — should still pass if those strings exist (they do). Then tighten: assert the Next block does **not** list `dc --all` before `dc try` if you reorder. After the edit, grep:

```bash
# First "Next:" through "If Docker" must include cd + dc + dc try
awk '/^echo "Next:"/,/If Docker/' install.sh | grep -q 'dc try'
```

- [ ] **Step 2: Replace the Next echo list in `install.sh`**

```bash
echo
echo "Next:"
echo "  source ~/.zshrc   # or ~/.bashrc"
echo "  cd /path/to/your/project"
echo "  dc                # board — start this folder"
echo "  dc try            # no .devcontainer or compose"
echo "If Docker is quit or split: dc recover"
```

Keep the missing-`devcontainer` advertised-curl footer. Drop `dc --all`, `dc upgrade`, `dc engine --fix` from the default Next list (they stay in `--help`).

- [ ] **Step 3: Run install + copy suites**

```bash
bash tests/copy/run.sh
bash tests/install/run.sh
```

Expected: both `ok`

- [ ] **Step 4: Commit**

```bash
git add install.sh tests/copy/run.sh
git commit -m "fix: lead install Next with dc, dc try, dc recover"
```

---

### Task 4: `dc-up` TTY offers `dc-try`

**Files:**
- Modify: `bin/dc-up` (the `kind=none` branch)
- Modify: `tests/try/run.sh`
- Modify: `README.md` / `skill/SKILL.md` only if the none sentence is still “refuses” with no TTY offer

**Interfaces:**
- Consumes: `dc_workspace_kind`, `dc-try --yes`
- Produces: non-TTY exit 1 + `dc-try` hint (unchanged); TTY y/N prompt; `dc-up --yes` on none still refuse

- [ ] **Step 1: Extend `tests/try/run.sh` with two cases**

Keep `case_up_hint_on_none` (non-TTY, exit 1, mentions `dc-try`).

Add:

```bash
case_up_yes_does_not_try() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  set +e
  dc-up --yes "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  assert_eq "$rc" 1
}

case_up_tty_no() {
  local ws rc out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  set +e
  out="$(printf 'n\n' | dc-up "$ws" 2>&1)"
  rc=$?
  set -e
  assert_eq "$rc" 1
  printf '%s\n' "$out" | grep -qi 'sandbox'
}
```

Register both with `run_case`.  
`case_up_tty_no` only works if `dc-up` treats a piped stdin as the answer **and** still shows the prompt when stdin is a pipe. Lock: prompt when stdin **is** a TTY. Piped `printf n` is **not** a TTY, so it must take the non-TTY path (exit 1, hint). Do **not** add `case_up_tty_no` unless you also add a hatch.

Preferred hatch (testable, no fake TTY):

```bash
# bin/dc-up kind=none
if [[ "${DC_UP_TRY_PROMPT:-}" == "1" ]]; then
  # force the y/n path even without a TTY (tests only)
fi
```

Then `case_up_tty_no` is:

```bash
DC_UP_TRY_PROMPT=1
out="$(printf 'n\n' | dc-up "$ws" 2>&1)"
```

And `case_up_tty_yes` with a fake `devcontainer` (same stub as `case_try_up_argv`):

```bash
DC_UP_TRY_PROMPT=1
printf 'y\n' | dc-up "$ws"
# expect override-config in $STATE/devc.log
```

- [ ] **Step 2: Run new cases to verify they fail**

```bash
bash tests/try/run.sh
```

Expected: FAIL on `up-yes-does-not-try` (if `--yes` currently just refuses, this may already PASS) and FAIL on prompt cases until `bin/dc-up` changes.

- [ ] **Step 3: Implement the `kind=none` branch in `bin/dc-up`**

Replace the current none block with:

```bash
if [[ "$_up_kind" == "none" ]]; then
  echo "dc-up: this folder is not a workspace (no .devcontainer, no compose file)." >&2
  echo "dc-up: sandbox start without project config: dc-try \"$_up_ws\"" >&2
  if [[ -n "${yes:-}" ]]; then
    echo "dc-up: --yes does not start a sandbox. Use: dc-try --yes \"$_up_ws\"" >&2
    exit 1
  fi
  if [[ "${DC_UP_TRY_PROMPT:-}" == "1" || -t 0 ]]; then
    printf 'Start a sandbox with dc-try? [y/N] ' >&2
    read -r _ans || _ans=
    if [[ "$_ans" == [yY] || "$_ans" == [yY][eE][sS] ]]; then
      exec dc-try --yes "$_up_ws"
    fi
    echo "dc-up: aborted" >&2
    exit 1
  fi
  echo "dc-up: non-interactive; start a sandbox with: dc-try --yes \"$_up_ws\"" >&2
  exit 1
fi
```

Confirm `yes` is the existing `--yes` flag variable in `bin/dc-up` (it is set for take-ports / create-nets). Use that same variable. If the flag is named differently, match the file.

- [ ] **Step 4: Run try + safety**

```bash
bash tests/try/run.sh
bash tests/safety/run.sh
```

Expected: both `ok`

- [ ] **Step 5: Commit**

```bash
git add bin/dc-up tests/try/run.sh
git commit -m "feat: offer dc-try from dc-up on a TTY when kind=none"
```

---

### Task 5: TUI sandbox confirm copy

**Files:**
- Modify: `cmd/dc-tui/view.go` (the `confirm == "try"` line)
- Test: `go test ./cmd/dc-tui`

**Interfaces:**
- Consumes: existing `confirm == "try"` flow
- Produces: friendlier string only

- [ ] **Step 1: Add or update a Go test that the try confirm line contains `sandbox`**

If no render test exists, add in `cmd/dc-tui/main_test.go`:

```go
func TestTryConfirmCopy(t *testing.T) {
	m := model{workspace: "/tmp/app", hoverStack: -1, confirm: "try", width: 80, height: 24}
	s := m.View()
	if !strings.Contains(s, "No config — start a sandbox?") {
		t.Fatalf("try confirm copy: %q", s)
	}
}
```

Import `strings` if needed.

- [ ] **Step 2: Run the test to see it fail**

```bash
go test ./cmd/dc-tui -run TestTryConfirmCopy
```

Expected: FAIL

- [ ] **Step 3: Change `cmd/dc-tui/view.go`**

```go
		b.WriteString("\n" + warnStyle.Render("No config — start a sandbox? y/n") + "\n")
```

- [ ] **Step 4: Run**

```bash
go test ./cmd/dc-tui
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/dc-tui/view.go cmd/dc-tui/main_test.go
git commit -m "fix: say sandbox, not dc-try jargon, on the empty-folder confirm"
```

---

### Task 6: Recover applies `dc-try` on `kind=none`

**Files:**
- Modify: `lib/dc-recover.sh` (`kind_none` row + apply function)
- Modify: `bin/dc-recover` if apply dispatch lives there
- Modify: `tests/recover/run.sh`
- Modify: `site/src/content/guides/stuck.md` one sentence
- Modify: `skill/SKILL.md` start/`kind=none` sentences

**Interfaces:**
- Consumes: `dc_recover_set`, `dc-try --yes`
- Produces: `next.id=try_sandbox`, `applyAllowed=1`, `--yes` runs `dc-try --yes "$ws"`

- [ ] **Step 1: Write the failing recover case**

In `tests/recover/run.sh`:

```bash
case_kind_none_allows_try() {
  plan_from ready unknown "ready" kind_none
  assert_eq "$DC_RECOVER_ID" try_sandbox
  assert_eq "$DC_RECOVER_APPLY_ALLOWED" 1
  printf '%s\n' "$DC_RECOVER_COMMAND" | grep -q 'dc-try'
}
```

Add `run_case kind-none-try case_kind_none_allows_try`

- [ ] **Step 2: Run**

```bash
bash tests/recover/run.sh
```

Expected: FAIL (`applyAllowed` is 0 today)

- [ ] **Step 3: Implement**

In `lib/dc-recover.sh` `kind_none` branch:

```bash
    kind_none)
      dc_recover_set try_sandbox \
        "This folder has no .devcontainer or compose file." \
        "dc-try --yes" \
        try_sandbox 1 \
        "dc-try"
      ;;
```

Add apply helper:

```bash
dc_recover_apply_try_sandbox() {
  local ws="${1:-.}"
  echo "dc-recover: dc-try --yes ${ws}"
  dc-try --yes "$ws"
}
```

Wire it in the existing `--yes` dispatch (same `case` as `colima_start` / `prune_safe`). Pass the recover workspace argument, not `.` if the user passed a path.

- [ ] **Step 4: Run**

```bash
bash tests/recover/run.sh
bash tests/try/run.sh
```

Expected: both `ok`

- [ ] **Step 5: Skill + stuck guide**

`skill/SKILL.md` — in the start bullet, replace “`dc-up` on none still refuses” with: TTY `dc-up` offers `dc-try`; agents use `dc-try --yes`; `dc-recover --yes` may apply `try_sandbox`.

`stuck.md` — add: if the host is ready and the folder has no config, recover’s next step is `dc-try`.

- [ ] **Step 6: Commit**

```bash
git add lib/dc-recover.sh bin/dc-recover tests/recover/run.sh skill/SKILL.md site/src/content/guides/stuck.md
git commit -m "feat: recover applies dc-try when the folder has no config"
```

---

### Task 7: Skill first-minute lead

**Files:**
- Modify: `skill/SKILL.md` (When to Use + Procedure step 1 opening)

**Interfaces:**
- Consumes: Task 4–6 behavior
- Produces: agents start with install → recover → up/try → exec

- [ ] **Step 1: Rewrite the first 4 sentences of When to Use**

```markdown
## When to Use
First minute: `dc doctor` (read-only). Host wall on an existing engine → `dc-recover` then `dc-recover --yes`. This folder: `.devcontainer` or compose → `dc-up`; else `dc-try --yes` (never edit the project). Then `dc-exec`. Never raw `docker exec NAME`.
```

Keep the rest of the skill (ports, prune, engine) after that lead. Do not delete safety rules.

- [ ] **Step 2: Commit**

```bash
git add skill/SKILL.md
git commit -m "docs: lead the agent skill with the first-minute path"
```

---

### Task 8: Verify the whole slice

**Files:** none new

- [ ] **Step 1: Run the suites this plan touches**

```bash
bash tests/copy/run.sh
bash tests/try/run.sh
bash tests/recover/run.sh
bash tests/install/run.sh
bash tests/safety/run.sh
go test ./cmd/dc-tui
```

Expected: all `ok` / PASS

- [ ] **Step 2: Site build**

```bash
cd site && npm run build
```

Expected: exit 0. Spot-check `dist/index.html` contains `Dev containers from your terminal.` and does not contain `Host-global helpers around the official Dev Containers CLI.`

- [ ] **Step 3: Manual first-minute (when a Docker engine is available)**

```bash
# advertised install into a temp prefix if you do not want to touch $HOME
cd /tmp
mkdir -p /tmp/dc-plain && cd /tmp/dc-plain
dc-up .          # non-TTY: hint; TTY: n aborts, y starts sandbox
dc-try --print .
```

Expected: no project files created; override under XDG state.

---

### Task 9: Listen + launch (ops, no code)

Not implemented by CI. Do after Tasks 1–7 are on `main` and Pages has rebuilt.

- [ ] **Step 1: Text the four silent users** (still pending in `plans/260815-first-run-curl-cli`)

```text
command -v devcontainer || echo MISSING
curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli
source ~/.zshrc
cd <their project>
dc
```

Ask: did you get a shell? If not, `dc doctor` then `dc recover`.

- [ ] **Step 2: Show HN** using `launch/hacker-news.md` (weekday 8–10 AM ET). First comment: first-minute path + honest limitation (wrappers, not a new engine).

- [ ] **Step 3: Product Hunt** using `launch/product-hunt.md`. Maker comment matches the new first sentence.

- [ ] **Step 4: Do not ship a new verb or a desktop app in the same week.**

---

## Self-review

1. **Spec coverage:** Site copy → Task 2. Install Next → Task 3. `dc-up` TTY → Task 4. TUI string → Task 5. Recover try apply → Task 6. Skill → Task 7. Copy/CI contract → Task 1. Launch → Task 9. Empty-machine install / desktop / spec rewrite / try ports — explicitly out.
2. **Placeholders:** none.
3. **Contracts:** advertised curl, `--yes` does not try, `dc-try` still external override.

## Out of this plan

Continue recover host lifecycle (start stopped engine, pick-one) under `plans/260822-0140-docker-self-serve` if any apply path is still incomplete. Do not expand that plan into this one.
