---
title: 'Guides, issues door, and per-page SEO'
description: >-
  Add four task guides and a GitHub issues door on dc.brasth.com, with real
  per-page SEO.
status: review-passed
priority: P2
branch: main
tags:
  - feature
  - frontend
  - docs
blockedBy: []
blocks: []
created: '2026-08-15T07:44:47.402Z'
createdBy: 'ck:plan'
source: skill
---

# Guides, issues door, and per-page SEO

## Overview

Landing is live at `https://dc.brasth.com`. README is already the command reference. Issues are on. There is no docs site, no issue form, no per-page SEO.

This plan adds a **thin guide layer** and a **GitHub issues door**. It does **not** add Mintlify, a ticket form, or a second README.

## Chosen approach (locked)

Picked over "one giant /guide TOC" and "full docs product":

| Decision | Value | Why |
|---|---|---|
| Routes | `/guide/`, 4 slugs, `/issues/` | Task pages rank. One blob does not. |
| Slugs | `install`, `tui`, `ports`, `disk` | The four things people fail at |
| README | Command reference stays | DRY. Guides do not copy flag tables |
| `/issues/` | Static door + templates | Pages is static. GitHub is the tracker |
| Wiki | Turn **off** | Split-brain with README |
| Docs platform | **No** | YAGNI. Astro pages are enough |
| Issue list on site | **No** | 0 issues. No API token. Link out |
| Unique OG art per page | **No** | Reuse `/og.png`. Titles/desc change |

Supersedes marketing lock **Pages: `/` only** in `plans/260815-1041-github-pages-marketing-site`. HTTPS is done. Extra routes are now allowed.

## SEO locked

Home already has OG + `SoftwareApplication` JSON-LD + a one-URL sitemap. `Layout.astro` hardcodes `canonical` to `/`. That is the bug.

| Rule | Value |
|---|---|
| URL shape | `trailingSlash: 'always'` |
| Canonical | Self, absolute, `https://dc.brasth.com{path}` |
| Title/desc/H1 | Unique per route. See phase 2 table |
| Sitemap | `@astrojs/sitemap`. Delete hand `public/sitemap.xml` |
| robots | Keep `Allow: /` + sitemap line |
| JSON-LD | Home: SoftwareApplication. Guides: HowTo + BreadcrumbList. `/guide/`: ItemList. `/issues/`: WebPage + BreadcrumbList |
| 404 | `noindex` stays |
| GSC / keyword APIs | Out of scope. Manual submit after ship |

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [SEO shell](./phase-01-seo-shell.md) | Done (local) |
| 2 | [Guide pages](./phase-02-guide-pages.md) | Done (local) |
| 3 | [Issues door and verify](./phase-03-issues-door-and-verify.md) | Code done; live HTTPS verify after push |

## Dependencies

- Live site + cert on `dc.brasth.com` (done)
- Isolated `site/` Astro 7 tree
- Public repo `Brasth/dc-cli` with Issues on
- Do **not** touch `bin/`, `lib/`, `cmd/`, `install.sh`

## Not in scope

- Mintlify / Starlight / Docusaurus / Wiki-as-docs
- Custom issue form or GitHub token
- Discussions, blog, changelog app
- Light mode, brasth.com apex, CLI behavior
- Search Console automation, pSEO, unique OG per guide
- `llms.txt`

## Success (whole plan)

- [ ] `/guide/` + 4 guides + `/issues/` return 200 over HTTPS (local `astro build` 200-equivalent; live after push)
- [x] Each indexable page has unique title, description, H1, self canonical (local dist)
- [x] Generated sitemap lists those URLs, not only `/`
- [x] New issue templates exist (`bug.yml` / `question.yml`); Wiki already off
- [x] README still owns the full command table; guides link to it (task tables still copied — see review)

## Review

- 2026-08-15: code review **Approved** 8/10. Report: [reports/260815-code-review.md](./reports/260815-code-review.md)
- No critical blockers. Warnings: README-slice guides, install title `Install dc-cli — dc-cli`, HowTo confirm step not on page.

## Next

1. Ship `site/` + `.github/ISSUE_TEMPLATE/` + README line together
2. After Pages: curl `/guide/install/` + `/issues/` + sitemap; confirm `template=` links
3. Optional: GSC add property + submit `https://dc.brasth.com/sitemap-index.xml`
4. Optional follow-up (not blocking): trim copied README tables; set issue labels `bug` / `question` (they exist)

## Cook

`/ck:cook /Users/huynguyen/src/dc-cli/plans/260815-1444-guides-issues-seo/plan.md`
