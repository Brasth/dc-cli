/**
 * Cloudflare Web Analytics (https://developers.cloudflare.com/web-analytics/get-started/).
 *
 * Production default is the JS-snippet token for dc.brasth.com (public; it ships in HTML).
 * Override with PUBLIC_CF_BEACON_TOKEN. Empty in dev unless that env is set.
 * Disable Cloudflare "automatic setup" if this snippet is on, or visits double-count.
 * No custom events — Play is `/play/` pageviews; install intent is GitHub downloads.
 */

const DEFAULT_PROD_TOKEN = '5e3af3d1b9fb4f46bca44129ff1ec913';

export function cloudflareBeaconToken(): string {
  const token = import.meta.env.PUBLIC_CF_BEACON_TOKEN ?? (import.meta.env.PROD ? DEFAULT_PROD_TOKEN : '');
  return /^[A-Za-z0-9_-]{16,128}$/.test(token) ? token : '';
}
