import { BoardSimulator, visibleLogLines, type BoardSnapshot } from './board-simulator';

function esc(s: string) {
  return s
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;');
}

function folderStatusLabel(s: BoardSnapshot) {
  if (s.refreshing) return 'refreshing…';
  if (s.folderStatus === 'checking') return 'checking…';
  if (s.folderStatus === 'starting') return 'starting…';
  if (s.folderStatus === 'stopping') return 'stopping…';
  if (s.running) return 'running  ready';
  return 'stopped';
}

function renderButtons(sim: BoardSimulator, hoverKey: string) {
  return sim
    .buttonGroups()
    .map((group) => {
      const btns = group
        .map((b) => {
          const cls = [
            'board-btn',
            b.meta && 'is-meta',
            b.danger && 'is-danger',
            b.disabled && 'is-disabled',
            hoverKey === b.key && 'is-on',
          ]
            .filter(Boolean)
            .join(' ');
          return `<button type="button" class="${cls}" data-key="${esc(b.key)}" ${
            b.disabled ? 'disabled' : ''
          }>${esc(b.label)}</button>`;
        })
        .join('');
      return `<div class="flex flex-wrap gap-1.5">${btns}</div>`;
    })
    .join('');
}

function renderStack(s: BoardSnapshot) {
  return s.stack
    .map((row, i) => {
      const selected = i === s.cursor ? ' is-selected' : '';
      return `<li class="board-row${selected} grid grid-cols-[1fr_4rem_3.5rem] items-baseline gap-3 px-4 py-2 font-mono text-[12px] md:grid-cols-[7rem_5rem_5rem_1fr]" data-stack="${i}">
      <span class="text-fg">${esc(row.name)}</span>
      <span class="text-muted">${esc(row.svc)}</span>
      <span class="${row.status === 'up' ? 'text-accent' : 'text-danger'}">${esc(row.status)}</span>
      <span class="hidden text-muted md:inline">${esc(row.img)}</span>
    </li>`;
    })
    .join('');
}

function leaveLine(kind: string) {
  switch (kind) {
    case 'start':
      return 'leaving board — dc up…';
    case 'shell':
      return 'leaving board — dc exec shell…';
    case 'logs':
      return 'leaving board — opening logs…';
    case 'files':
      return 'leaving board — dc files…';
    default:
      return `leaving board — ${kind}…`;
  }
}

function renderBoardMain(root: HTMLElement, s: BoardSnapshot, sim: BoardSimulator, hoverKey: string) {
  root.innerHTML = `
    <header class="flex flex-wrap items-start gap-3 px-4 pt-4">
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" width="36" height="36" fill="none" aria-hidden="true" class="text-muted">
        <rect x="5" y="5" width="54" height="54" rx="14" stroke="currentColor" stroke-width="2"></rect>
        <rect x="16" y="16" width="32" height="32" rx="8" stroke="currentColor" stroke-width="1.75"></rect>
        <rect x="40" y="12" width="7" height="7" rx="1.5" fill="var(--accent)"></rect>
      </svg>
      <div class="min-w-0 flex-1">
        <p class="font-mono text-sm font-semibold text-fg">dc-cli <span class="font-normal text-accent">app</span></p>
        <p class="font-mono text-[11px] text-muted">${esc(s.workspace)}</p>
        <p class="font-mono text-[11px] text-muted">load  ${esc(s.loadPulse)}  t=top</p>
        <p class="font-mono text-[11px] text-muted">disk  ${esc(s.disk)}  d=df</p>
        <p class="font-mono text-[11px] text-muted">nets  ${esc(s.netsLine)}  n=nets</p>
      </div>
    </header>

    <dl class="mt-3 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 px-4 font-mono text-[12px] leading-5">
      <dt class="text-muted">this folder</dt><dd class="text-soft">${esc(s.workspace)}</dd>
      <dt class="text-muted">status</dt><dd class="text-accent">${esc(folderStatusLabel(s))}</dd>
      <dt class="text-muted">editor</dt><dd class="text-soft">${esc(s.editor)}</dd>
    </dl>

    <div class="mt-4 flex flex-col gap-1.5 px-4" role="toolbar" aria-label="dc-tui actions">
      ${renderButtons(sim, hoverKey)}
    </div>
    <p class="px-4 pt-2 font-mono text-[11px] text-muted" id="board-hint">${esc(s.hint)}</p>

    ${s.leaving ? `<p class="px-4 font-mono text-[11px] text-warn">${esc(leaveLine(s.leaving))}</p>` : ''}
    ${s.confirm === 'rm' ? `<p class="px-4 font-mono text-[11px] text-warn">remove stack containers? y/n</p>` : ''}
    ${s.confirm === 'try' ? `<p class="px-4 font-mono text-[11px] text-warn">No config — start a sandbox? y/n</p>` : ''}
    ${s.status ? `<p class="px-4 font-mono text-[11px] text-accent">${esc(s.status)}</p>` : ''}
    ${s.err ? `<p class="px-4 font-mono text-[11px] text-danger">${esc(s.err)}</p>` : ''}

    <ul class="mt-4 border-t border-line" id="board-stack">${s.stack.length ? renderStack(s) : '<li class="px-4 py-3 font-mono text-[11px] text-muted">(no containers — press u to start)</li>'}</ul>
    <p class="board-foot px-4 py-3 font-mono text-[11px] text-muted">${s.hasConfig ? 'Sandbox demo — install <span class="text-fg">dc-cli</span> to run against your folder.' : 'First-run demo — no config folder. Press <span class="text-fg">u</span> then <span class="text-fg">y</span> to try a sandbox, then <span class="text-fg">e</span> for shell.'}</p>
  `;
}

function renderLogs(root: HTMLElement, s: BoardSnapshot) {
  const lines = visibleLogLines(s.logLines, s.logOffset);
  root.innerHTML = `
    <div class="board-overlay px-4 py-4 font-mono text-[12px]">
      <p class="text-sm font-semibold text-fg">dc-cli <span class="text-muted">logs</span> <span class="text-accent">${esc(s.logName)}</span></p>
      <p class="mt-1 text-[11px] text-muted">q back · j/k scroll · f follow (${s.logFollow ? 'on' : 'off'})</p>
      <pre class="mt-4 max-h-48 overflow-y-auto text-[11px] leading-5 text-soft">${lines.map((l) => esc(l)).join('\n')}</pre>
    </div>
  `;
}

function renderTop(root: HTMLElement, s: BoardSnapshot) {
  const rows = s.topRows
    .map((r, i) => {
      const sel = i === s.topCursor ? ' bg-overlay' : '';
      return `<div class="grid grid-cols-[5rem_4rem_1fr_1fr] gap-2 px-2 py-1${sel}">
        <span>${esc(r.svc)}</span><span>${r.cpu.toFixed(1)}%</span><span>${esc(r.mem)}</span><span class="text-muted">${esc(r.net)}</span>
      </div>`;
    })
    .join('');
  root.innerHTML = `
    <div class="board-overlay px-4 py-4 font-mono text-[12px]">
      <p class="text-sm font-semibold text-fg">dc-cli <span class="text-muted">top</span></p>
      <p class="mt-1 text-[11px] text-muted">q back · j/k select</p>
      <p class="mt-4 text-[11px] text-muted">SERVICE · CPU · MEM · NET</p>
      <div class="mt-2 border-t border-line pt-2">${rows || '<p class="text-muted">(no running boxes)</p>'}</div>
    </div>
  `;
}

function renderNets(root: HTMLElement, s: BoardSnapshot) {
  const rows = s.nets
    .map((n) => {
      const state = n.exists ? 'exists' : 'missing';
      const kind = n.external ? 'external bridge' : 'compose';
      return `<li class="py-1"><span class="text-fg">${esc(n.name)}</span> <span class="text-muted">${state} · ${kind}</span></li>`;
    })
    .join('');
  root.innerHTML = `
    <div class="board-overlay px-4 py-4 font-mono text-[12px]">
      <p class="text-sm font-semibold text-fg">dc-cli <span class="text-muted">nets</span></p>
      <p class="mt-1 text-[11px] text-muted">q back · y create missing external bridge</p>
      <ul class="mt-4 space-y-1">${rows}</ul>
    </div>
  `;
}

function renderMore(root: HTMLElement) {
  root.innerHTML = `
    <div class="board-overlay px-4 py-4 font-mono text-[12px] leading-6 text-soft">
      <p class="text-sm font-semibold text-fg">dc-cli <span class="text-muted">more</span></p>
      <p class="mt-3 text-muted">Primary: u start · e shell · s stop</p>
      <p class="text-muted">Meta: o open · a attach · p ports · l logs · t top · n nets · b db · m files</p>
      <p class="text-muted">Stack: j/k move · Enter exec row · R restart sibling</p>
      <p class="text-muted">? back · q quit · r reload · x rm (y/n)</p>
      <p class="mt-4 text-[11px] text-muted">Press ? or q to return.</p>
    </div>
  `;
}

export interface MountBoardOptions {
  demoMode?: 'configured' | 'none';
  /**
   * window: keys work anywhere on the page (the /play page).
   * focus: keys only while the board has focus, so an embedded board never
   * hijacks typing or page shortcuts (the homepage).
   */
  keyScope?: 'window' | 'focus';
}

export function mountBoard(root: HTMLElement, options: MountBoardOptions = {}) {
  let hoverKey = '';
  const keyScope = options.keyScope ?? (root.dataset.keys === 'focus' ? 'focus' : 'window');
  const fromDom = root.dataset.demo === 'none' ? 'none' : root.dataset.demo === 'configured' ? 'configured' : undefined;
  const demoMode = options.demoMode ?? fromDom ?? 'configured';
  const sim = new BoardSimulator((snap) => paint(snap), { demoMode });

  function paint(s: BoardSnapshot) {
    // Re-rendering replaces the focused button; keep focus inside the board.
    const hadFocus = root.contains(document.activeElement);
    render(s);
    if (hadFocus && !root.contains(document.activeElement)) root.focus({ preventScroll: true });
  }

  function render(s: BoardSnapshot) {
    switch (s.view) {
      case 'logs':
        renderLogs(root, s);
        break;
      case 'top':
        renderTop(root, s);
        break;
      case 'nets':
        renderNets(root, s);
        break;
      case 'more':
        renderMore(root);
        break;
      default:
        renderBoardMain(root, s, sim, hoverKey);
    }
  }

  root.addEventListener('pointerover', (event) => {
    const btn = (event.target as HTMLElement).closest<HTMLButtonElement>('.board-btn');
    if (!btn) return;
    const key = btn.dataset.key ?? '';
    if (key === hoverKey) return;
    hoverKey = key;
    // Toggle the hover state in place. Re-rendering here would put a new
    // element under the pointer, fire pointerover again, and loop forever.
    root.querySelectorAll<HTMLElement>('.board-btn').forEach((el) => {
      el.classList.toggle('is-on', el.dataset.key === hoverKey);
    });
  });

  root.addEventListener('click', (event) => {
    // Clicking a board button re-renders it; keep keyboard focus on the board.
    if (keyScope === 'focus') {
      queueMicrotask(() => {
        if (!root.contains(document.activeElement)) root.focus({ preventScroll: true });
      });
    }
    const btn = (event.target as HTMLElement).closest<HTMLButtonElement>('.board-btn');
    if (btn) {
      const key = btn.dataset.key ?? '';
      hoverKey = key;
      sim.handleButton(key);
      return;
    }
    const row = (event.target as HTMLElement).closest<HTMLElement>('[data-stack]');
    if (row) {
      const i = Number(row.dataset.stack);
      sim.selectStack(i);
      sim.execSelected();
    }
  });

  paint(sim.snapshot());

  const onKey = (event: KeyboardEvent) => {
    const target = event.target as HTMLElement | null;
    if (target && /^(INPUT|TEXTAREA)$/.test(target.tagName)) return;
    if (event.metaKey || event.ctrlKey || event.altKey) return;
    if (sim.handleKey(event.key)) event.preventDefault();
  };
  const keyTarget: HTMLElement | Window = keyScope === 'focus' ? root : window;
  keyTarget.addEventListener('keydown', onKey as EventListener);

  return () => {
    keyTarget.removeEventListener('keydown', onKey as EventListener);
    sim.destroy();
  };
}
