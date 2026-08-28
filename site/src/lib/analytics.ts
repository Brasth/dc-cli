/**
 * Privacy-friendly analytics via Plausible (https://plausible.io).
 * Set PUBLIC_PLAUSIBLE_DOMAIN in site env to enable; defaults to dc.brasth.com in production builds.
 */

declare global {
  interface Window {
    plausible?: (event: string, options?: { props?: Record<string, string> }) => void;
  }
}

const domain =
  import.meta.env.PUBLIC_PLAUSIBLE_DOMAIN ??
  (import.meta.env.PROD ? 'dc.brasth.com' : '');

export function trackEvent(name: string, props?: Record<string, string>) {
  if (!domain || typeof window === 'undefined') return;
  window.plausible?.(name, props ? { props } : undefined);
}

export function bindAnalytics() {
  if (!domain || typeof document === 'undefined') return;
  document.addEventListener('click', (event) => {
    const target = event.target;
    if (!(target instanceof Element)) return;
    const installBtn = target.closest<HTMLElement>('[data-install-copy]');
    if (installBtn) {
      const source = installBtn.id || installBtn.dataset.trackSource || 'install';
      trackEvent('Install Copy', { source });
    }
    const playLink = target.closest<HTMLElement>('a[href="/play/"]');
    if (playLink) {
      trackEvent('Play Demo', { source: playLink.dataset.trackSource ?? 'link' });
    }
  });
}

export function plausibleDomain() {
  return domain;
}
