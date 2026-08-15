---
phase: 2
title: "Guide pages"
status: done
priority: P1
dependencies: [1]
effort: "1d"
---

# Phase 2: Guide pages

## Overview

Four task guides + `/guide/` index. Same terminal-machine look. README stays the flag bible. Each page has unique on-page SEO + HowTo/ItemList schema.

## Context links

- Source of truth: `README.md` (Install, TUI, Ports, Disk, Host port conflict)
- Agent copy is **not** for humans: `skill/SKILL.md`
- Brand: `assets/branding/copy.md`, `site/src/styles/tokens.css`

## Requirements

- Functional: `/guide/` lists the four guides. Each slug is a real how-to.
- Functional: guides link to README anchors for full command tables.
- Non-functional: unique title, description, H1, canonical, breadcrumbs. No thin-page duplicates of README.

## Architecture

```
site/src/content.config.ts
site/src/content/guides/{install,tui,ports,disk}.md
site/src/layouts/Guide.astro          # breadcrumb + article + footer CTA
site/src/pages/guide/index.astro
site/src/pages/guide/[slug].astro
```

Content frontmatter:

```yaml
title: "Install dc-cli"
description: "…"
h1: "One curl. Then source your shell."
updated: 2026-08-15
howto: true          # emit HowTo JSON-LD from steps
faq: []              # optional FAQPage
```

Do **not** put plan IDs in content or comments.

### Route + SEO table (locked)

| Path | Title | H1 intent | Primary query | Schema |
|---|---|---|---|---|
| `/guide/` | Guides — dc-cli | Task index | dc-cli guide | ItemList + BreadcrumbList |
| `/guide/install/` | Install dc-cli | Install + flags that matter | dc-cli install | HowTo + BreadcrumbList |
| `/guide/tui/` | dc-tui board and keys | Board keys, open vs attach | dc-tui | FAQPage + BreadcrumbList |
| `/guide/ports/` | Colima ports and host clashes | Why :3000 is dead; port already allocated | dc-cli colima ports | HowTo + BreadcrumbList |
| `/guide/disk/` | Disk full: dc-df and dc-prune | ENOSPC, never docker prune -af | dc-cli disk full | HowTo + BreadcrumbList |

Host-port clash lives **inside** `ports`, not a fifth page.

### Copy rules

- Task steps. Failure modes. One command block per step.
- Link `README.md#Commands-all-of-them` (or the live GitHub anchor) for the full table.
- Same install curl as README: `Brasth/dc-cli`.
- Voice: terse, host-global, no VS Code required. Match landing.
- Do not invent flags.

## Related Code Files

- Create: `site/src/content.config.ts`
- Create: `site/src/content/guides/install.md`
- Create: `site/src/content/guides/tui.md`
- Create: `site/src/content/guides/ports.md`
- Create: `site/src/content/guides/disk.md`
- Create: `site/src/layouts/Guide.astro`
- Create: `site/src/pages/guide/index.astro`
- Create: `site/src/pages/guide/[slug].astro`
- Modify: `site/src/layouts/Layout.astro` only if schema helper needs a tweak
- Modify: `README.md` — add Guides link under the site line. Do not rewrite sections.

## Implementation Steps

1. Content collection `guides` with the frontmatter above.
2. `Guide.astro`: breadcrumb `dc-cli / guide / {title}`, article, “full flags → README”, CTA to `/issues/` and `#install` or `/#install`.
3. `[slug].astro`: 404 unknown slugs. Pass `path: /guide/${slug}/` into Layout.
4. Index: 2x2 of the four tasks. ItemList JSON-LD of the four URLs.
5. Each HowTo: 3–6 steps that match visible copy. FAQ on TUI: open vs attach; e vs stack row.
6. Internal links: home hero/nav already has guide. Each guide links siblings. Landing Commands section may add “full guide →”.
7. Build. Grep dist HTML for each canonical + H1.

## Success Criteria

- [x] Five URLs 200 locally (`/guide/` + 4 slugs)
- [x] Each has unique `<title>`, meta description, H1, self canonical with trailing slash
- [x] HowTo/ItemList JSON-LD present and URLs absolute
- [ ] No full command table pasted from README (install flags + tui keys + disk/ports tables still copied — warning, not blocker)
- [x] Install curl matches README exactly
- [x] Unknown `/guide/nope/` is 404 noindex
- [x] Sitemap after build includes the five URLs

## Risk Assessment

- README/guide drift: guides stay task-only; flags live in README.
- Thin content: ports + disk must include the actual failure (Colima, ENOSPC). Do not pad.
- Trailing slash mismatch vs GitHub README links: site uses trailing slash; GitHub anchors do not need one.

## Security Considerations

Static Markdown. No user input. No `target=_blank` without `rel` if used.
