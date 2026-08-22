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
      return `<li class="board-row${selected} grid grid-cols-[1fr_auto] items-baseline gap-3 px-4 py-2 font-mono text-[12px] md:grid-cols-[7rem_5rem_5rem_1fr]" data-stack="${i}">
      <span class="text-ink">${esc(row.name)}</span>
      <span class="text-mute">${esc(row.svc)}</span>
      <span class="${row.status === 'up' ? 'text-[#8ecf7a]' : 'text-[#d36b6b]'}">${esc(row.status)}</span>
      <span class="hidden text-mute md:inline">${esc(row.img)}</span>
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
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" width="36" height="36" fill="none" aria-hidden="true">
        <rect x="5" y="5" width="54" height="54" rx="14" stroke="#8A8680" stroke-width="2"></rect>
        <rect x="16" y="16" width="32" height="32" rx="8" stroke="#8A8680" stroke-width="1.75"></rect>
        <rect x="40" y="12" width="7" height="7" rx="1.5" fill="#6FCF7B"></rect>
      </svg>
      <div class="min-w-0 flex-1">
        <p class="font-mono text-sm font-semibold text-[#F4F1EA]">dc-cli <span class="font-normal text-[#8ecf7a]">app</span></p>
        <p class="font-mono text-[11px] text-mute">${esc(s.workspace)}</p>
        <p class="font-mono text-[11px] text-mute">load  ${esc(s.loadPulse)}  t=top</p>
        <p class="font-mono text-[11px] text-mute">disk  ${esc(s.disk)}  d=df</p>
        <p class="font-mono text-[11px] text-mute">nets  ${esc(s.netsLine)}  n=nets</p>
      </div>
    </header>

    <dl class="mt-3 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 px-4 font-mono text-[12px] leading-5">
      <dt class="text-mute">this folder</dt><dd class="text-ink/85">${esc(s.workspace)}</dd>
      <dt class="text-mute">status</dt><dd class="text-[#8ecf7a]">${esc(folderStatusLabel(s))}</dd>
      <dt class="text-mute">editor</dt><dd class="text-ink/85">${esc(s.editor)}</dd>
    </dl>

    <div class="mt-4 flex flex-col gap-1.5 px-4" role="toolbar" aria-label="dc-tui actions">
      ${renderButtons(sim, hoverKey)}
    </div>
    <p class="px-4 pt-2 font-mono text-[11px] text-mute" id="board-hint">${esc(s.hint)}</p>

    ${s.leaving ? `<p class="px-4 font-mono text-[11px] text-amber">${esc(leaveLine(s.leaving))}</p>` : ''}
    ${s.confirm === 'rm' ? `<p class="px-4 font-mono text-[11px] text-amber">remove stack containers? y/n</p>` : ''}
    ${s.status ? `<p class="px-4 font-mono text-[11px] text-[#8ecf7a]">${esc(s.status)}</p>` : ''}
    ${s.err ? `<p class="px-4 font-mono text-[11px] text-[#d36b6b]">${esc(s.err)}</p>` : ''}

    <ul class="mt-4 border-t border-white/5" id="board-stack">${renderStack(s)}</ul>
    <p class="board-foot px-4 py-3 font-mono text-[11px] text-mute">Sandbox demo — install <span class="text-ink/80">dc-cli</span> to run against your folder.</p>
  `;
}

function renderLogs(root: HTMLElement, s: BoardSnapshot) {
  const lines = visibleLogLines(s.logLines, s.logOffset);
  root.innerHTML = `
    <div class="board-overlay px-4 py-4 font-mono text-[12px]">
      <p class="text-sm font-semibold text-ink">dc-cli <span class="text-mute">logs</span> <span class="text-[#8ecf7a]">${esc(s.logName)}</span></p>
      <p class="mt-1 text-[11px] text-mute">q back · j/k scroll · f follow (${s.logFollow ? 'on' : 'off'})</p>
      <pre class="mt-4 max-h-48 overflow-y-auto text-[11px] leading-5 text-ink/85">${lines.map((l) => esc(l)).join('\n')}</pre>
    </div>
  `;
}

function renderTop(root: HTMLElement, s: BoardSnapshot) {
  const rows = s.topRows
    .map((r, i) => {
      const sel = i === s.topCursor ? ' bg-white/5' : '';
      return `<div class="grid grid-cols-[5rem_4rem_1fr_1fr] gap-2 px-2 py-1${sel}">
        <span>${esc(r.svc)}</span><span>${r.cpu.toFixed(1)}%</span><span>${esc(r.mem)}</span><span class="text-mute">${esc(r.net)}</span>
      </div>`;
    })
    .join('');
  root.innerHTML = `
    <div class="board-overlay px-4 py-4 font-mono text-[12px]">
      <p class="text-sm font-semibold text-ink">dc-cli <span class="text-mute">top</span></p>
      <p class="mt-1 text-[11px] text-mute">q back · j/k select</p>
      <p class="mt-4 text-[11px] text-mute">SERVICE · CPU · MEM · NET</p>
      <div class="mt-2 border-t border-white/5 pt-2">${rows || '<p class="text-mute">(no running boxes)</p>'}</div>
    </div>
  `;
}

function renderNets(root: HTMLElement, s: BoardSnapshot) {
  const rows = s.nets
    .map((n) => {
      const state = n.exists ? 'exists' : 'missing';
      const kind = n.external ? 'external bridge' : 'compose';
      return `<li class="py-1"><span class="text-ink">${esc(n.name)}</span> <span class="text-mute">${state} · ${kind}</span></li>`;
    })
    .join('');
  root.innerHTML = `
    <div class="board-overlay px-4 py-4 font-mono text-[12px]">
      <p class="text-sm font-semibold text-ink">dc-cli <span class="text-mute">nets</span></p>
      <p class="mt-1 text-[11px] text-mute">q back · y create missing external bridge</p>
      <ul class="mt-4 space-y-1">${rows}</ul>
    </div>
  `;
}

function renderMore(root: HTMLElement) {
  root.innerHTML = `
    <div class="board-overlay px-4 py-4 font-mono text-[12px] leading-6 text-ink/85">
      <p class="text-sm font-semibold text-ink">dc-cli <span class="text-mute">more</span></p>
      <p class="mt-3 text-mute">Primary: u start · e shell · s stop</p>
      <p class="text-mute">Meta: o open · a attach · p ports · l logs · t top · n nets · b db · m files</p>
      <p class="text-mute">Stack: j/k move · Enter exec row · R restart sibling</p>
      <p class="text-mute">? back · q quit · r reload · x rm (y/n)</p>
      <p class="mt-4 text-[11px] text-mute">Press ? or q to return.</p>
    </div>
  `;
}

export function mountBoard(root: HTMLElement) {
  let hoverKey = '';
  const sim = new BoardSimulator((snap) => paint(snap));

  function paint(s: BoardSnapshot) {
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
    hoverKey = btn.dataset.key ?? '';
    if (sim.snapshot().view === 'board') paint(sim.snapshot());
  });

  root.addEventListener('click', (event) => {
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
    if (sim.handleKey(event.key)) event.preventDefault();
  };
  window.addEventListener('keydown', onKey);

  return () => {
    window.removeEventListener('keydown', onKey);
    sim.destroy();
  };
}
