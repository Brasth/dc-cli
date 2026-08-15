# QA: guides + issues + SEO

Date: 2026-08-15 14:53
Scope: local `site/` build + dist SEO + issue templates
Did not touch: `bin/`, `lib/`, `cmd/`, `install.sh`

## Test Results Overview

| Check | Result |
|---|---|
| `site && npm run build` | PASS (8 pages, 629ms) |
| Dist routes | PASS (8/8) |
| Unique title/desc/canonical/robots | PASS |
| 404 noindex | PASS |
| Sitemap URLs | PASS (7 loc, no 404) |
| robots.txt Sitemap | PASS |
| JSON-LD types | PASS |
| Install curl Brasth/dc-cli | PASS |
| ISSUE_TEMPLATE files | PASS |

Total gates: 9 run, 9 passed, 0 failed, 0 skipped.

## Coverage Metrics

N/A — static Astro pages, no unit suite in this scope.

Built routes: `/` `/guide/` `/guide/install/` `/guide/tui/` `/guide/ports/` `/guide/disk/` `/issues/` `/404.html`

## Failed Tests

None.

## Performance Metrics

`astro build`: 8 page(s) in 629ms. No slow-test concern.

## Build Status

PASS.

```
cd /Users/huynguyen/src/dc-cli/site && npm run build

14:52:35 [build] output: "static"
14:52:35   ├─ /404.html
14:52:35   ├─ /guide/disk/index.html
14:52:35   ├─ /guide/install/index.html
14:52:35   ├─ /guide/ports/index.html
14:52:35   ├─ /guide/tui/index.html
14:52:35   ├─ /guide/index.html
14:52:35   ├─ /issues/index.html
14:52:35   ├─ /index.html
14:52:35 [@astrojs/sitemap] `sitemap-index.xml` created at `dist`
14:52:35 [build] 8 page(s) built in 629ms
14:52:35 [build] Complete!
```

## Evidence

### 1. Dist routes

All present:

- `dist/index.html`
- `dist/guide/index.html`
- `dist/guide/install/index.html`
- `dist/guide/tui/index.html`
- `dist/guide/ports/index.html`
- `dist/guide/disk/index.html`
- `dist/issues/index.html`
- `dist/404.html`

### 2. Per-page SEO

| Route | Title | Robots | Canonical |
|---|---|---|---|
| `/` | dc-cli — host-global Dev Containers helpers | index, follow | https://dc.brasth.com/ |
| `/guide/` | Guides — dc-cli | index, follow | https://dc.brasth.com/guide/ |
| `/guide/install/` | Install dc-cli — dc-cli | index, follow | https://dc.brasth.com/guide/install/ |
| `/guide/tui/` | dc-tui board and keys — dc-cli | index, follow | https://dc.brasth.com/guide/tui/ |
| `/guide/ports/` | Colima ports and host clashes — dc-cli | index, follow | https://dc.brasth.com/guide/ports/ |
| `/guide/disk/` | Disk full: dc-df and dc-prune — dc-cli | index, follow | https://dc.brasth.com/guide/disk/ |
| `/issues/` | Report a dc-cli issue | index, follow | https://dc.brasth.com/issues/ |
| 404 | 404 — dc-cli | noindex, nofollow | https://dc.brasth.com/404/ |

Titles unique. Descriptions unique. Canonicals self + trailing slash + `https://dc.brasth.com`.

Descriptions:

- `/` Start, exec, stop, ports, and disk for official @devcontainers/cli. No VS Code required. Does not edit project .devcontainer.
- `/guide/` Task guides for dc-cli: install, dc-tui keys, Colima ports, and disk full. Command flags stay in the README.
- `/guide/install/` Install dc-cli with one curl. Tracks the latest GitHub release. Optional official CLI and agent skill.
- `/guide/tui/` dc-tui is the clickable board for this folder. Keys for start, shell, open, attach, ports, stop, logs, fleet.
- `/guide/ports/` dc-forward publishes app ports on Colima. dc-up shows the holder when a host port is already allocated.
- `/guide/disk/` When dc-up fails with ENOSPC, run dc-df then dc-prune --yes. Never docker system prune -af --volumes.
- `/issues/` File a dc-cli bug or question on GitHub. Include the command, OS, and Colima vs Docker Desktop. No secrets.

### 3. Sitemap

`dist/sitemap-index.xml` → `https://dc.brasth.com/sitemap-0.xml`

`dist/sitemap-0.xml` locs:

- https://dc.brasth.com/
- https://dc.brasth.com/guide/
- https://dc.brasth.com/guide/disk/
- https://dc.brasth.com/guide/install/
- https://dc.brasth.com/guide/ports/
- https://dc.brasth.com/guide/tui/
- https://dc.brasth.com/issues/

No 404 URL.

### 4. robots.txt

```
User-agent: *
Allow: /

Sitemap: https://dc.brasth.com/sitemap-index.xml
```

### 5. JSON-LD

| Route | Types |
|---|---|
| `/` | SoftwareApplication only (no BreadcrumbList; allowed) |
| `/guide/` | BreadcrumbList + ItemList |
| `/guide/install/` | BreadcrumbList + HowTo |
| `/guide/tui/` | BreadcrumbList + FAQPage |
| `/guide/ports/` | BreadcrumbList + HowTo |
| `/guide/disk/` | BreadcrumbList + HowTo |
| `/issues/` | BreadcrumbList + WebPage |
| 404 | none |

HowTo names: Install dc-cli; Colima ports and host clashes; Disk full: dc-df and dc-prune.
TUI FAQPage has 3 questions.

### 6. Install curl

Home + `/guide/install/`:

```
curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh
```

Not Canvilled.

Note: home JSON-LD `author.url` is `https://github.com/Canvilled` (person). Not install curl. Not a fail.

### 7. Issue templates

`.github/ISSUE_TEMPLATE/`:

- `config.yml` — `blank_issues_enabled: false`; contact_links → https://dc.brasth.com/guide/
- `bug.yml` — command, OS, Docker backend, logs required
- `question.yml` — asked / tried / expected required

`/issues/` links:

- `https://github.com/Brasth/dc-cli/issues/new?template=bug.yml`
- `https://github.com/Brasth/dc-cli/issues/new?template=question.yml`

## Critical Issues

None.

## Recommendations

- After push: live curl HTTPS 200 + cert for `/guide/` slugs + `/issues/` (phase 3 live gate; not this local check).
- Manual GSC sitemap submit still out of scope.

## Next Steps

1. Ship / wait Pages.
2. Live verify `curl -sI https://dc.brasth.com/guide/install/` and sitemap.
3. Confirm GitHub new-issue UI shows Bug + Question only.
4. Wiki disable if not already (`gh api -X PATCH repos/Brasth/dc-cli -f has_wiki=false`).

## Unresolved questions

None for the 9 assigned local checks.
