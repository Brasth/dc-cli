---
phase: 1
title: SEO shell
status: done
priority: P1
dependencies: []
effort: 0.5d
---

# Phase 1: SEO shell

## Overview

Make `Layout.astro` multi-page safe. Lock URL shape, per-route meta, sitemap. Nav/footer get `guide` + `issues` links so later pages are not orphaned.

## Context links

- `site/src/layouts/Layout.astro` — hardcoded `canonical = https://dc.brasth.com/`
- `site/public/sitemap.xml` — one URL, will fight `@astrojs/sitemap`
- `site/public/robots.txt` — already correct
- `site/astro.config.mjs` — `site` set, no `trailingSlash`
- Prior plan: `plans/260815-1041-github-pages-marketing-site`

## Requirements

- Functional: any page can pass title, description, path, noindex, extra JSON-LD
- Functional: home keeps current SoftwareApplication schema + OG image
- Non-functional: one URL shape; no relative canonicals; 404 stays noindex

## Architecture

```
Layout props
  title, description, path, noindex, schema[]
       │
       ├─ <title> / meta description / robots
       ├─ link rel=canonical  https://dc.brasth.com{path}
       ├─ og:url + og:title + og:description (same og.png)
       └─ JSON-LD: default WebSite + page schema[]
```

URL lock: `trailingSlash: 'always'` in `astro.config.mjs`. Home `/`. Nested `/guide/install/`.

Sitemap: add `@astrojs/sitemap`. Remove `site/public/sitemap.xml`. 404 excluded (Astro default).

Do **not** write guide copy in this phase. Stub routes optional; if added, they must not ship empty indexable shells. Prefer nav links that 404 until phase 2/3 **or** add `noindex` stubs. **Preferred:** link in nav now; pages land in later phases. Brief 404 on `/guide/` for one deploy is OK.

## Related Code Files

- Modify: `site/src/layouts/Layout.astro`
- Modify: `site/astro.config.mjs`
- Modify: `site/package.json` / lockfile (`@astrojs/sitemap`)
- Modify: `site/src/components/Nav.astro`
- Modify: `site/src/components/Footer.astro`
- Delete: `site/public/sitemap.xml`
- Keep: `site/public/robots.txt` (confirm Sitemap line still `https://dc.brasth.com/sitemap.xml`)

## Implementation Steps

1. Add `trailingSlash: 'always'` next to existing `site: 'https://dc.brasth.com'`.
2. `npm i @astrojs/sitemap` in `site/`. Register in `astro.config.mjs`.
3. Expand Layout props:

```ts
interface Props {
  title?: string;
  description?: string;
  path?: string;          // '/guide/install/'
  noindex?: boolean;
  schema?: unknown[];     // extra JSON-LD graphs
}
```

4. Canonical = `new URL(path ?? Astro.url.pathname, 'https://dc.brasth.com').href`. Never hardcode `/` except home default.
5. Home default title/description/schema stay as today. `og:url` follows canonical.
6. Nav: add `guide` → `/guide/`, keep github + `$ install`. Footer: `guides` + `issues` + existing repo/readme/upstream.
7. Delete hand sitemap. Build. Confirm `dist/sitemap-index.xml` or `dist/sitemap-0.xml` exists.

## Success Criteria

- [x] Home HTML still has unique title + SoftwareApplication JSON-LD
- [x] Home canonical is `https://dc.brasth.com/`
- [x] Layout no longer forces every page to `/`
- [x] `site/dist` sitemap is generated, not copied from `public/`
- [x] 404 remains `noindex`
- [x] `bin/` `lib/` `cmd/` `install.sh` untouched

## Risk Assessment

- Trailing-slash change can 404 old `/guide` without slash. Fine: those URLs do not exist yet.
- `@astrojs/sitemap` + leftover `public/sitemap.xml` = two sitemaps. Delete the public file.
- Nav pointing at missing `/guide/` for one commit: acceptable. Do not ship empty indexable stubs.

## Security Considerations

None. Static meta only. No tokens.
