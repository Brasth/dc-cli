# First-run funnel audit

Date: 2026-08-28  
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

## Remaining gaps (this branch)

| Gap | Fix |
|-----|-----|
| Fleet empty state omits `dc try` | Update TUI copy in `view.go` / `model.go` |
| Web `/play/` demo always shows running stack | Add `kind=none` demo mode to board simulator |
| No site analytics | Add Plausible + install CTA events |
| No activation feedback template | Extend GitHub issue templates |
| Launch ops pending | `launch/execution.md` checklist + silent-user retry block |
| No funnel tracking process | `docs/growth/funnel-tracking.md` |

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

Core first-run product paths are implemented and tested. Growth work should focus on **conversion polish** (empty states, honest web demo), **measurement**, and **launch execution** — not new CLI verbs.
