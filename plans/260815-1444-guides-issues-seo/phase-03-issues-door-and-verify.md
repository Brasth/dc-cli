---
phase: 3
title: "Issues door and verify"
status: in-progress
priority: P1
dependencies: [2]
effort: "0.5d"
---

# Phase 3: Issues door and verify

## Overview

Ship `/issues/` as a door to GitHub. Add issue templates. Turn Wiki off. Prove live SEO after push.

## Context links

- Issues on, 0 templates: `https://github.com/Brasth/dc-cli/issues`
- New issue: `https://github.com/Brasth/dc-cli/issues/new`
- Wiki is on and empty — disable
- Phase 1 Layout + phase 2 guides must already exist

## Requirements

- Functional: `/issues/` explains what to file and jumps to templated new-issue URLs
- Functional: opening a new issue without a template is blocked or steered
- Non-functional: page is indexable, unique meta, no GitHub token, no form POST

## Architecture

```
/issues/  (static)
   ├─ bug     → /Brasth/dc-cli/issues/new?template=bug.yml
   └─ question → /Brasth/dc-cli/issues/new?template=question.yml

.github/ISSUE_TEMPLATE/
   config.yml     blank_issues_enabled: false
                  contact_links → https://dc.brasth.com/guide/
   bug.yml        command, OS, Docker backend, logs
   question.yml   what you tried, expected
```

No octokit. No issue list. Empty tracker is not a feature.

Wiki: `gh api -X PATCH repos/Brasth/dc-cli -f has_wiki=false`

## Related Code Files

- Create: `site/src/pages/issues.astro`
- Create: `.github/ISSUE_TEMPLATE/config.yml`
- Create: `.github/ISSUE_TEMPLATE/bug.yml`
- Create: `.github/ISSUE_TEMPLATE/question.yml`
- Modify: `site/src/components/Footer.astro` if issues link not already live
- Modify: `README.md` — one Issues line next to Site / Guides
- Modify: `site/public/robots.txt` only if sitemap filename changed

## Implementation Steps

1. Write `/issues/` in the same machine voice. H1: report a break, not “Contact us”.
2. Two buttons/links: bug template, question template. Third link: open issue list.
3. Tell people to paste `dc-df` / `dc-up` output. Do not ask for secrets.
4. Issue templates: required fields for command + OS + Colima vs Desktop. Labels `bug` / `question` only if those labels exist; otherwise omit labels.
5. `config.yml` `blank_issues_enabled: false` + contact_link to `/guide/`.
6. Disable Wiki.
7. Push. Wait for Pages. Verify live:

```bash
curl -sI https://dc.brasth.com/guide/install/ | head
curl -s https://dc.brasth.com/guide/install/ | grep -E 'canonical|og:url|<h1'
curl -s https://dc.brasth.com/sitemap-index.xml
# or sitemap-0.xml
curl -sI https://dc.brasth.com/issues/ | head
```

8. Optional user step: Search Console add `https://dc.brasth.com` + submit sitemap. Not a code gate.

## Success Criteria

- [ ] `https://dc.brasth.com/guide/` and four slugs 200, valid cert (after push)
- [ ] `https://dc.brasth.com/issues/` 200, canonical self, links hit `template=` (after push)
- [ ] New issue UI shows Bug + Question, not a blank box (templates ready; verify on GitHub after push)
- [x] Wiki tab gone or disabled (`hasWikiEnabled: false`)
- [ ] Live sitemap lists `/`, `/guide/`, four slugs, `/issues/` (local sitemap already does)
- [x] No page still canonicalizes to `/` except home (local dist)
- [x] Clipboard install on home still matches README

## Risk Assessment

- Template query param ignored if filename mismatches. Use exact `bug.yml` / `question.yml`.
- `blank_issues_enabled: false` without templates locks intake. Ship templates in the same commit.
- Wiki disable needs admin. If API 403, leave a note; do not block the site.

## Security Considerations

- Public issue templates only. No maintainer emails.
- Do not ask for `.env`, tokens, or full `docker inspect`.
- External GitHub links: same-tab is fine; if `target=_blank`, add `rel="noopener noreferrer"`.
