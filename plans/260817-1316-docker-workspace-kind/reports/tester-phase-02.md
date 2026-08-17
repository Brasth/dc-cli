---
title: "Tester — Phase 2 compose-kind identity"
status: pass
created: 2026-08-17
updated: 2026-08-17
---

# Tester Phase 2 — compose-kind identity (re-run after claimants fix)

Scope: kind detect + `dc-ls` compose-kind (no fake labels) + stopped/`working_dir`/foreign/`--all` gates + `dc-up` refuse + doctor kind/start-disabled. **Claimants High fix:** labeled `local_folder` + own `.devcontainer` `working_dir` must not double-count (n=1); labeled + *foreign* compose-kind still n=2. Safety: labeled compose-in-`.devcontainer` still take-ports stop. Regression: unlabeled take-ports, doctor 18 ids / schemaVersion 1, Phase 1 exec restart.

Re-run after claimants skip-labeled-working_dir. Prior report: 8 kind / 19 safety (hole untested). Now 10 kind + 20 safety.

No product source changed by tester. No assertions weakened. TUI not in this re-run (user scope: four bash suites).

## Test Results Overview

| Suite | Command | Result | Pass | Fail | Skip | Time |
|---|---|---|---:|---:|---:|---|
| kind | `bash tests/kind/run.sh` | PASS | 10 | 0 | 0 | 3.892s |
| safety | `bash tests/safety/run.sh` | PASS | 20 | 0 | 0 | 12.355s |
| doctor | `bash tests/doctor/run.sh` | PASS | 9 | 0 | 0 | 13.480s |
| exec | `bash tests/exec/run.sh` | PASS | 8 | 0 | 0 | 1.381s |

**Total: 47 pass / 0 fail / 0 skip.** (this re-run; TUI 70 not re-run)

First pass + timed confirm both green. Same counts.

## Phase 2 gates

### `bash tests/kind/run.sh`

```
  ok  kind-detect
  ok  ls-compose-kind
  ok  ls-stopped-empty
  ok  ls-no-working-dir
  ok  ls-all-labeled-only
  ok  ls-foreign-working-dir
  ok  up-compose-kind-refuse
  ok  claimants-same-stack
  ok  claimants-labeled-plus-foreign
  ok  doctor-compose-kind

ok  10
```

| Required | Case | Assert | Result |
|---|---|---|---|
| detect compose / both=devcontainer / none | `kind-detect` | empty=`none`; compose.yaml=`compose`; +`.devcontainer`=`devcontainer` | PASS |
| `dc-ls` compose-kind + no fake labels | `ls-compose-kind` | JSON `"kind":"compose"`, id `c1`, no `devcontainer.local_folder`, `log_lacks commit` | PASS |
| stopped compose-only folder `[]` | `ls-stopped-empty` | `dc-ls --json` == `[]` | PASS |
| no `working_dir` → unknown / `[]` | `ls-no-working-dir` | same project name only → `[]` | PASS |
| `--all` labeled-only | `ls-all-labeled-only` | labeled `app1` + `kind=devcontainer`; compose-only `c1` absent | PASS |
| foreign `working_dir` not owned | `ls-foreign-working-dir` | same compose name, other folder's `working_dir` → `[]` | PASS |
| `dc-up` compose-kind refuse | `up-compose-kind-refuse` | rc=1, stdout has `compose-kind`, no `devc.log`, no `compose … up` | PASS |
| **claimants same-stack n=1** | `claimants-same-stack` | labeled `$ws` + `working_dir=$ws/.devcontainer` → `dc_compose_claimants` n=1 | PASS |
| **claimants labeled+foreign n=2** | `claimants-labeled-plus-foreign` | labeled `$a` + unlabeled `working_dir=$other` same project → n=2 | PASS |
| doctor kind=compose + start disabled | `doctor-compose-kind` | JSON `kind` compose + `start disabled` | PASS |

Name-alone never owns (no-`working_dir` + foreign `working_dir`). No invented DC label writes (`commit` absent from fake-docker log). `dc-up` refuses before official CLI (`devcontainer` stub never invoked).

**Claimants fix verified:** skip labeled container `working_dir` as a second claim. Same-stack official-CLI layout (`local_folder=$ws`, `working_dir=$ws/.devcontainer`) is one claimant. Foreign unlabeled compose-kind still a second claimant (fail-closed n>1 kept).

### `bash tests/safety/run.sh`

```
20/20 passed
```

| Case | Result |
|---|---|
| port takeover positive foreign | PASS |
| **port takeover labeled compose-in-devcontainer** | PASS (`log_has 'compose -p projA stop'`) |
| port takeover unlabeled | PASS |
| port takeover / down ambiguous compose | PASS |
| port takeover same workspace | PASS |
| port takeover forward sidecar | PASS |
| port takeover sidecar inspect-unknown | PASS |
| nontty without flag does not stop | PASS |
| compose revalidate missing id | PASS |
| prune scope honesty | PASS |
| single-volume protection | PASS |
| volume query error | PASS |
| orphan inspect-unknown no rm | PASS |
| orphan absent removed | PASS |
| prune image step failure | PASS |
| lock concurrent fail-closed | PASS |
| lock nested inherit | PASS |
| lock dead-owner reclaim | PASS |
| lock age-alone never reclaim | PASS |
| partial mixed holders | PASS |

New case `port takeover labeled compose-in-devcontainer`: labeled `$a` + `working_dir=$a/.devcontainer` + port clash → `--take-ports` still `compose -p projA stop`. This is the High regression that 19/19 missed (fixtures omitted `working_dir`).

Unlabeled take-ports still report-only: rc≠0, stdout has `unlabeled`, `log_lacks 'stop stray'`, `log_lacks 'rm stray'`. Mixed-holder still stops labeled `projA` only (`log_has 'compose -p projA stop'`) and lacks `stop stray`.

No new stop path for unlabeled / compose-kind holders. Phase 2 `--take-ports` still report-only for unlabeled. Ambiguous compose (two labeled folders, same project) still refuse / `ambiguous-compose`.

### `bash tests/doctor/run.sh`

```
== doctor gates ==
  ok  help exit 0
  ok  unknown flag exit 2
  ok  non-dir exit 2
  ok  json schema + 18 ids
  ok  missing common valid json
  ok  fake-docker zero mutation
  ok  below-floor blocker
  ok  missing config reported
  ok  secrets not leaked

9/9 passed
```

`json schema + 18 ids` greps `schemaVersion` 1 and all 18 check ids:

`platform_bash` `common_library` `docker_cli` `docker_daemon` `docker_context` `colima` `devcontainer_cli` `devcontainer_read_configuration` `workspace_config` `duplicate_labels` `stack_identity` `desired_ports` `required_networks` `actual_ports` `stale_owned_sidecars` `disk` `dc_cli_version` `dc_cli_channel`

schemaVersion still 1 (additive `kind` did not bump). 18 ids still present. Zero-mutation / secrets cases still green.

### `bash tests/exec/run.sh`

```
  ok  term-env
  ok  color-rc-hl
restart  db  db-1  db1
  ok  restart-sibling
  ok  restart-unknown
  ok  restart-id-banned
  ok  restart-no-app
restart  db  db-1  ffff1111
  ok  restart-id-prefix-collision
  ok  restart-needs-service

ok  8
```

Phase 1 restart still green. Collision still `restart ffff1111` only. `--id --restart` still exit 2. No compose-kind start/exec leaked into this suite.

## Coverage Metrics

Bash suites: no coverage tooling. Kind cases exercise `dc_workspace_kind`, this-folder `dc-ls --json` compose tagging, `--all` labeled filter, classify fail-closed on missing/foreign `working_dir`, `dc-up` refuse-before-CLI, doctor `workspace.kind` + start-disabled warning, **and** `dc_compose_claimants` n=1 vs n=2.

Safety new case exercises take-ports stop on labeled compose-in-`.devcontainer` (the High hole). Unlabeled still report-only.

Doctor/ls `kind` is additive; schemaVersion stayed 1.

TUI cover not re-measured (not in this re-run). Prior: 57.3% statements / 58.9% func. Out of Phase 2 scope.

## Failed Tests

None.

## Performance Metrics

- `tests/kind/run.sh`: 3.892s, 10 fake-docker cases. +2 vs prior 8. Still fast. Claimants cases are inspect-only.
- `tests/safety/run.sh`: 12.355s, 20 harness cases. Slowest required suite. +1 case vs 19. Fake-docker + lock/prune. Not flaky (2/2 green).
- `tests/doctor/run.sh`: 13.480s, 9 cases. Includes live-ish doctor JSON path. No flake (2/2).
- `tests/exec/run.sh`: 1.381s, 8 cases.

No flaky pass/fail. Deterministic fake-docker harnesses. First run + timed confirm same pass/fail.

## Build Status

No production build run (not required). Bash harnesses sourced `dc-common` / `dc-net` / `dc-up` / `dc-ls` / `dc-doctor` / `dc-exec` without syntax fail. No deprecation warnings in suite output.

## Critical Issues

None.

## Recommendations

- High claimants hole is closed in tests: same-stack n=1; labeled+foreign n=2; labeled compose-in-`.devcontainer` still take-ports `compose -p projA stop`.
- Phase 2 success criteria still green: running compose-only `kind=compose` no DC labels; stopped folder `[]`; `--all` no daemon compose inventory; `dc-up` no `devcontainer` / `compose up`; same-name foreign `working_dir` not owned; unlabeled take-ports still report-only.
- Skill detect-only; manifesto not in this tester pass. Docs observational.
- Optional leftover (not this gate): `config_files`-only positive ls case; TTY `dc-ls` vs JSON id source (review Medium). Do not treat as claimants-fix blockers.
- Do not add TUI start / compose-kind stop until Phase 3.

## Next Steps

1. Claimants re-run green (10 kind / 20 safety / 9 doctor / 8 exec). High fix empirically held.
2. Do not enable `dc-up` / compose start / `compose exec` (Phase 3 only if user still wants start).
3. Keep `--take-ports` report-only for unlabeled / compose-kind holders until Phase 3.
4. User demo of `dc-ls --json` / `dc-doctor --json` on compose-only folder is still the safe surface. Labeled compose-in-`.devcontainer` `dc-down` / take-ports now has a passing harness case.

## Unresolved questions

None.
