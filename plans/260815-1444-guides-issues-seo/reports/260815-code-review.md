# Code Review — guides + issues + SEO

Date: 2026-08-15
Scope: uncommitted site/docs (no CLI)
Verdict: **Approved** — 8/10
Plan: `plans/260815-1444-guides-issues-seo/plan.md`

## Code Review Summary

### Scope
- Files reviewed: Layout/Page, schema helper, guide routes + 4 content files, issues page, issue templates, Nav/Footer/Commands, 404, robots, astro.config, package.json, README site line
- Lines of code analyzed: ~890
- Review focus: recent uncommitted site/docs for guides + issues + SEO
- Updated plans: `plans/260815-1444-guides-issues-seo/plan.md` + 3 phase files

### Overall Assessment
SEO shell is correct. Multi-page Layout no longer hardcodes `/`. Sitemap is generated. Guides exist as four task URLs. Issues door has no token/form. Templates do not ask for secrets. Wiki already off. `astro build` 8 pages, exit 0. Landing H1 / SoftwareApplication / install curl unchanged. Not a second full README, but tui/ports/disk are near-verbatim README slices.

### Critical Issues
None.

### High Priority Findings
None.

### Medium Priority Improvements
1. **README-slice guides (DRY).** `tui.md` / `ports.md` / `disk.md` copy README sections almost line-for-line (keys table, ports prose, prune risk table). `install.md` copies the install flags table. Drift risk when README changes. Not a second full README (no Commands-all table / editors / skill). Trim tables to the failure path + link README.
2. **Install HowTo step 3 not visible.** Schema says confirm via `dc-tui --help` / `dc-up --help`. Body has curl + source + flags, no confirm step. Add one line or drop the step.
3. **Title `Install dc-cli — dc-cli`.** `[slug].astro` always appends ` — dc-cli`. Unique, but awkward vs locked table title `Install dc-cli`.

### Low Priority Suggestions
1. Issue labels `bug` and `question` exist on Brasth/dc-cli; templates use `labels: []`. Plan said use them if they exist.
2. `question.yml` has no “do not paste secrets” line. Bug template does.
3. Unused frontmatter `updated`. Either show it or drop it.
4. Nav logo `#top` → `/` is correct for inner pages; home logo now reloads instead of scrolling. `/#top` would keep both.
5. After ship, submit `https://dc.brasth.com/sitemap-index.xml` (old `sitemap.xml` is deleted).
6. 404 canonical is `https://dc.brasth.com/404/` (noindex, not in sitemap). Fine.

### Positive Observations
- Self canonical + og:url match on every indexable route; trailing slash consistent (`trailingSlash: 'always'`).
- Schema types match lock: home SoftwareApplication; `/guide/` ItemList+BreadcrumbList; install/ports/disk HowTo; tui FAQPage; issues WebPage.
- 404 stays `noindex, nofollow`; no JSON-LD; not in sitemap.
- robots → `sitemap-index.xml`. Generated sitemap lists `/`, `/guide/`, 4 slugs, `/issues/`.
- Issue templates: no emails, no token, no `.env` / `docker inspect`. `blank_issues_enabled: false` + templates ship together.
- CLI tree (`bin/` `lib/` `cmd/` `install.sh`) untouched.
- Shared `Page.astro` instead of a one-off Guide layout is the right DRY cut.
- Internal links use trailing slashes. GitHub README anchors (`#commands-all-of-them`, `#tui`, `#ports-colima`, `#disk-full`, `#host-port-conflict`) match live headings.

### Recommended Actions
1. Ship as-is. Live curl after Pages.
2. Optional follow-up: trim copied tables; fix install title; add visible confirm step; set issue labels.

### Metrics
- Type Coverage: Astro strict tsconfig; `npx astro check` not run (would install `@astrojs/check`). `npm run build` exit 0.
- Test Coverage: no site tests. Manual dist grep of title/desc/h1/canonical/schema.
- Linting Issues: 0 from build.

### Spec check (local)

| Requirement | Status |
|---|---|
| Layout per-route title/desc/path/canonical | PASS |
| Home SoftwareApplication + unique title | PASS |
| trailingSlash always | PASS |
| Generated sitemap, hand sitemap deleted | PASS |
| `/guide/` + 4 slugs + `/issues/` | PASS |
| Unique title/desc/H1/canonical | PASS (install title redundant) |
| HowTo/ItemList/FAQ/WebPage as locked | PASS |
| Guides not a second full README | PASS with warning (section copies) |
| Install curl = README | PASS |
| Templates, no secrets, no token | PASS |
| Wiki off | PASS (`hasWikiEnabled: false`) |
| Live HTTPS 200 | PENDING push |

### Unresolved
- Live Pages 200 / cert / new-issue UI not verified (uncommitted).
- GSC submit is optional and out of code scope.
