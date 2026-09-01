# First-run funnel audit

Date: 2026-08-28
Reconciled: 2026-08-31
Branch: feature/growth-strategy-e1fa

## Success state

A stranger with Docker installed reaches **`dc exec`** (shell in the app container) within one minute of copying the install command.

## Audit results

| Step | Expected | Status | Evidence |
|------|----------|--------|----------|
| Install advertises `--with-cli` | Official CLI on PATH | **PASS** | `tests/copy/run.sh` greps README, Hero, install.sh |
| Install "Next" block | `source` → `cd` → `dc` → `dc try` | **PASS** | `install.sh` Next block test in copy suite |
| `dc-up` on `kind=none` (non-TTY) | Refuse + `dc-try` hint | **PASS** | `tests/try/run.sh` up-hint-none |
| `dc-up` on `kind=none` (TTY) | Offer sandbox confirm | **PASS** | `tests/try/run.sh` up-tty-yes / up-tty-no |
| `dc-up --yes` on `kind=none` | Does not auto-try | **PASS** | `tests/try/run.sh` up-yes-does-not-try |
| Recover `kind_none` | Apply `dc-try --yes` | **PASS** | `tests/recover/run.sh` apply-try-sandbox |
| TUI start failure | Shows error, not "back from start" | **PASS** | `cmd/dc-tui/main_test.go` TestExecDoneStartFailure |
| TUI sandbox confirm copy | "No config — start a sandbox? y/n" | **PASS** | `cmd/dc-tui/view.go` |

## Remaining gaps

| Gap | Status |
|-----|--------|
| Fleet empty state omits `dc try` | **Shipped** — `cmd/dc-tui/view.go` (empty fleet copy includes `dc try`) |
| Web `/play/` demo always shows running stack | **Shipped** — `site/src/components/Board.astro` (`data-demo="none"` on compact `/play/`) |
| No site analytics | **Shipped (wiring only)** — Cloudflare Web Analytics via optional `PUBLIC_CF_BEACON_TOKEN` JS snippet; proxied hostnames can use dashboard auto-inject instead. No custom click events. Do **not** claim live receipt. |
| No activation feedback template | **Shipped** — `.github/ISSUE_TEMPLATE/feedback.yml` + `/issues/` Activation feedback card |
| No funnel tracking process | **Shipped** — `docs/growth/funnel-tracking.md` + `scripts/funnel-snapshot.sh` (public GitHub API). |
| Launch ops pending | **Still pending** — `launch/execution.md` checklist + silent-user retry. Unverified; do not check off. |
| Live analytics receipt | **Still pending** — enable dc.brasth.com in Cloudflare Web Analytics (auto if orange-cloud, else beacon token + Pages secret). Dashboard unmeasured. |

## Manual smoke (Docker required)

Run on a machine with Docker when validating releases:

```bash
# kind=none
mkdir /tmp/dc-audit-none && cd /tmp/dc-audit-none
dc-up          # expect TTY sandbox prompt
dc-try --yes   # expect sandbox shell path
dc exec        # SUCCESS

# kind=devcontainer — use any repo with .devcontainer/
cd /path/to/devcontainer-project
dc up && dc exec   # SUCCESS
```

## Conclusion

Core first-run product paths are implemented and tested. `/play/` no-config, Cloudflare Web Analytics wiring, activation feedback, and funnel-tracking docs are in-repo. Growth work should focus on **measurement** (activation baseline still unmeasured) and **launch execution** — not new CLI verbs.
