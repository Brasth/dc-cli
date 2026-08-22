export type BoardView = 'board' | 'logs' | 'top' | 'nets' | 'more';

export type StackStatus = 'up' | 'exited';

export interface StackRow {
  name: string;
  svc: string;
  status: StackStatus;
  img: string;
}

export interface TopRow {
  svc: string;
  cpu: number;
  mem: string;
  net: string;
}

export interface NetRow {
  name: string;
  exists: boolean;
  external: boolean;
}

export interface BoardSnapshot {
  view: BoardView;
  workspace: string;
  folderStatus: 'stopped' | 'running' | 'starting' | 'stopping' | 'checking';
  editor: string;
  loadPulse: string;
  disk: string;
  netsLine: string;
  stack: StackRow[];
  cursor: number;
  confirm: '' | 'rm';
  leaving: '' | 'start' | 'shell' | 'logs' | 'files';
  status: string;
  err: string;
  refreshing: boolean;
  running: boolean;
  logName: string;
  logLines: string[];
  logFollow: boolean;
  topRows: TopRow[];
  topCursor: number;
  nets: NetRow[];
  urls: { key: string; label: string; url: string }[];
  hint: string;
}

type BtnSpec = {
  key: string;
  label: string;
  primary?: boolean;
  meta?: boolean;
  danger?: boolean;
  disabled?: boolean;
};

const INITIAL_STACK: StackRow[] = [
  { name: 'app-1', svc: 'app', status: 'up', img: 'node:22' },
  { name: 'db-1', svc: 'db', status: 'up', img: 'postgres:16' },
  { name: 'mitm-1', svc: 'mitm', status: 'exited', img: 'alpine/socat' },
];

const SAMPLE_LOGS = [
  '2026-08-22T05:00:01.123Z GET /health 200 12ms',
  '2026-08-22T05:00:04.881Z GET /api/projects 200 48ms',
  '2026-08-22T05:00:09.204Z POST /api/auth/login 200 112ms',
  '2026-08-22T05:00:11.557Z GET /api/workspaces 304 6ms',
  '2026-08-22T05:00:15.002Z GET /static/app.js 200 3ms',
  '2026-08-22T05:00:18.441Z GET /api/containers 200 31ms',
  '2026-08-22T05:00:22.019Z GET / 200 8ms',
  '2026-08-22T05:00:25.773Z GET /api/ports 200 19ms',
];

export class BoardSimulator {
  private view: BoardView = 'board';
  private workspace = '~/src/app';
  private folderStatus: BoardSnapshot['folderStatus'] = 'running';
  private editor = 'zed';
  private stack: StackRow[] = INITIAL_STACK.map((r) => ({ ...r }));
  private cursor = 0;
  private confirm: '' | 'rm' = '';
  private leaving: BoardSnapshot['leaving'] = '';
  private status = '';
  private err = '';
  private refreshing = false;
  private running = true;
  private logName = 'app-1';
  private logLines = [...SAMPLE_LOGS];
  private logFollow = true;
  private logOffset = 0;
  private topRows: TopRow[] = [
    { svc: 'app', cpu: 12.4, mem: '410M / 2G', net: '1.2M / 84K' },
    { svc: 'db', cpu: 3.1, mem: '128M / 1G', net: '420K / 210K' },
  ];
  private topCursor = 0;
  private nets: NetRow[] = [
    { name: 'shared-net', exists: false, external: true },
    { name: 'default', exists: true, external: false },
  ];
  private urls = [
    { key: '1', label: 'http://127.0.0.1:9001', url: 'http://127.0.0.1:9001' },
    { key: '2', label: 'http://127.0.0.1:5173', url: 'http://127.0.0.1:5173' },
  ];
  private hint =
    'u start · e shell · s stop · t top · n nets · b db · m files · 1-9 url · j/k enter · x rm asks y/n · q quit';
  private timers: number[] = [];
  private onChange: (snap: BoardSnapshot) => void;

  constructor(onChange: (snap: BoardSnapshot) => void) {
    this.onChange = onChange;
    // this.emit(); // moved to mountBoard
    this.schedule(() => this.tickPulse(), 2200);
  }

  destroy() {
    for (const id of this.timers) window.clearTimeout(id);
    this.timers = [];
  }

  snapshot(): BoardSnapshot {
    return {
      view: this.view,
      workspace: this.workspace,
      folderStatus: this.folderStatus,
      editor: this.editor,
      loadPulse: this.running ? 'cpu 12.4%  mem 410M / —' : 'cpu —  mem —',
      disk: 'docker 42% · colima 61%',
      netsLine: 'missing shared-net',
      stack: this.stack.map((r) => ({ ...r })),
      cursor: this.cursor,
      confirm: this.confirm,
      leaving: this.leaving,
      status: this.status,
      err: this.err,
      refreshing: this.refreshing,
      running: this.running,
      logName: this.logName,
      logLines: [...this.logLines],
      logFollow: this.logFollow,
      topRows: this.topRows.map((r) => ({ ...r })),
      topCursor: this.topCursor,
      nets: this.nets.map((n) => ({ ...n })),
      urls: this.urls.map((u) => ({ ...u })),
      hint: this.hint,
    };
  }

  handleKey(key: string): boolean {
    if (this.confirm === 'rm') {
      if (key === 'y') {
        this.confirm = '';
        this.runRm();
        return true;
      }
      if (key === 'n' || key === 'Escape') {
        this.confirm = '';
        this.setStatus('rm cancelled');
        return true;
      }
      return false;
    }

    if (this.view === 'logs') {
      if (key === 'q' || key === 'Escape' || key === 'l') {
        this.view = 'board';
        this.hint =
          'u start · e shell · s stop · t top · n nets · b db · m files · 1-9 url · j/k enter · x rm asks y/n · q quit';
        this.emit();
        return true;
      }
      if (key === 'j') {
        this.logOffset = Math.min(this.logOffset + 1, Math.max(0, this.logLines.length - 6));
        this.emit();
        return true;
      }
      if (key === 'k') {
        this.logOffset = Math.max(0, this.logOffset - 1);
        this.emit();
        return true;
      }
      if (key === 'f') {
        this.logFollow = !this.logFollow;
        this.emit();
        return true;
      }
      return false;
    }

    if (this.view === 'top') {
      if (key === 'q' || key === 'Escape' || key === 't') {
        this.view = 'board';
        this.emit();
        return true;
      }
      if (key === 'j') {
        this.topCursor = Math.min(this.topCursor + 1, this.topRows.length - 1);
        this.emit();
        return true;
      }
      if (key === 'k') {
        this.topCursor = Math.max(0, this.topCursor - 1);
        this.emit();
        return true;
      }
      return false;
    }

    if (this.view === 'nets') {
      if (key === 'q' || key === 'Escape' || key === 'n') {
        this.view = 'board';
        this.emit();
        return true;
      }
      if (key === 'y') {
        const missing = this.nets.find((n) => !n.exists && n.external);
        if (missing) {
          missing.exists = true;
          this.setStatus(`created bridge ${missing.name} — run start to attach`);
        }
        this.emit();
        return true;
      }
      return false;
    }

    if (this.view === 'more') {
      if (key === '?' || key === 'q' || key === 'Escape') {
        this.view = 'board';
        this.emit();
        return true;
      }
      return false;
    }

    if (key === 'j' || key === 'ArrowDown') {
      this.cursor = Math.min(this.cursor + 1, this.stack.length - 1);
      this.syncLogTarget();
      this.emit();
      return true;
    }
    if (key === 'k' || key === 'ArrowUp') {
      this.cursor = Math.max(0, this.cursor - 1);
      this.syncLogTarget();
      this.emit();
      return true;
    }
    if (key === 'Enter') {
      this.execStackRow(this.cursor);
      return true;
    }
    if (key === 'r') {
      this.reload();
      return true;
    }

    return this.runAction(key);
  }

  selectStack(index: number) {
    if (index < 0 || index >= this.stack.length) return;
    this.cursor = index;
    this.syncLogTarget();
    this.emit();
  }

  execSelected() {
    this.execStackRow(this.cursor);
  }

  handleButton(key: string) {
    if (key.startsWith('url:')) {
      const url = key.slice(4);
      this.setStatus(`open ${url} in host browser`);
      return;
    }
    this.handleKey(key);
  }


  private runAction(key: string): boolean {
    switch (key) {
      case 'u':
        if (!this.canStart()) return false;
        this.startUp();
        return true;
      case 'e':
        if (this.blocked()) return false;
        this.startLeave('shell', () => {
          this.setStatus('shell exited — board resumed');
        });
        return true;
      case 's':
        if (this.blocked()) return false;
        this.stopStack();
        return true;
      case 'x':
        if (this.blocked()) return false;
        this.confirm = 'rm';
        this.hint = 'remove stack containers? y/n';
        this.emit();
        return true;
      case 'l':
        if (this.blocked() || !this.canFollowLogs()) return false;
        this.openLogs();
        return true;
      case 't':
        if (this.blocked() || !this.running) return false;
        this.view = 'top';
        this.emit();
        return true;
      case 'n':
        if (this.blocked()) return false;
        this.view = 'nets';
        this.emit();
        return true;
      case 'o':
        if (this.blocked()) return false;
        this.setStatus('dc-open — host editor on bind-mount');
        return true;
      case 'a':
        if (this.blocked()) return false;
        this.setStatus('dc-open --attach — VS Code Remote URI printed');
        return true;
      case 'p':
        if (this.blocked()) return false;
        this.setStatus('dc-forward — reconciled owned sidecars');
        return true;
      case 'b':
        if (this.blocked()) return false;
        this.setStatus('dc-db — TablePlus on declared db port');
        return true;
      case 'm':
        if (this.blocked()) return false;
        this.startLeave('files', () => this.setStatus('dc-files exited — board resumed'));
        return true;
      case 'f':
        this.setStatus('fleet view needs install — try dc --all');
        return true;
      case '?':
        this.view = 'more';
        this.emit();
        return true;
      case 'q':
        this.setStatus('quit — install dc-cli to run the real board');
        return true;
      case '1':
      case '2':
        this.handleButton(`url:${this.urls.find((u) => u.key === key)?.url ?? ''}`);
        return true;
      default:
        return false;
    }
  }

  buttonGroups(): BtnSpec[][] {
    const blocked = this.blocked();
    return [
      [
        { key: 'u', label: 'start', primary: true, disabled: !this.canStart() },
        { key: 'e', label: 'shell', primary: true, disabled: blocked },
        { key: 's', label: 'stop', primary: true, disabled: blocked },
      ],
      [
        { key: 'o', label: 'open', meta: true, disabled: blocked },
        { key: 'a', label: 'attach', meta: true, disabled: blocked },
        { key: 'p', label: 'ports', meta: true, disabled: blocked },
        { key: 'l', label: 'logs', meta: true, disabled: blocked || !this.canFollowLogs() },
        { key: 't', label: 'top', meta: true, disabled: blocked || !this.running },
        { key: 'n', label: 'nets', meta: true, disabled: blocked },
      ],
      [
        { key: 'b', label: 'db', meta: true, disabled: blocked },
        { key: 'm', label: 'files', meta: true, disabled: blocked },
      ],
      [
        { key: 'f', label: 'fleet', meta: true },
        { key: '?', label: 'more', meta: true },
        { key: 'q', label: 'quit', meta: true },
        { key: 'x', label: 'rm', danger: true, disabled: blocked },
      ],
      this.urls.map((u) => ({
        key: `url:${u.url}`,
        label: `${u.key} ${u.label}`,
        primary: true,
        disabled: blocked || !this.running,
      })),
    ].filter((g) => g.length > 0);
  }

  private blocked() {
    return (
      this.folderStatus === 'checking' ||
      this.folderStatus === 'starting' ||
      this.folderStatus === 'stopping' ||
      this.leaving !== ''
    );
  }

  private canStart() {
    return !this.blocked() && (!this.running || this.stack.some((r) => r.status === 'exited'));
  }

  private canFollowLogs() {
    return this.stack.length > 0;
  }

  private startUp() {
    this.folderStatus = 'starting';
    this.leaving = 'start';
    this.setStatus('');
    this.emit();
    this.schedule(() => {
      this.leaving = '';
      this.running = true;
      this.folderStatus = 'running';
      for (const row of this.stack) row.status = 'up';
      const mitm = this.stack.find((r) => r.svc === 'mitm');
      if (mitm) mitm.status = 'up';
      this.setStatus('dc up — kind=devcontainer · started · ports forwarded');
      this.topRows = [
        { svc: 'app', cpu: 12.4, mem: '410M / 2G', net: '1.2M / 84K' },
        { svc: 'db', cpu: 3.1, mem: '128M / 1G', net: '420K / 210K' },
        { svc: 'mitm', cpu: 0.4, mem: '12M / 256M', net: '8K / 8K' },
      ];
      this.emit();
    }, 900);
  }

  private stopStack() {
    this.folderStatus = 'stopping';
    this.setStatus('dc down — stopping compose stack…');
    this.emit();
    this.schedule(() => {
      for (const row of this.stack) row.status = 'exited';
      this.running = false;
      this.folderStatus = 'stopped';
      this.setStatus('dc down — stack stopped');
      this.topRows = [];
      this.emit();
    }, 700);
  }

  private runRm() {
    this.setStatus('dc down --rm — removing stack containers…');
    this.emit();
    this.schedule(() => {
      this.stack = this.stack.filter((r) => r.svc !== 'mitm');
      this.setStatus('dc down --rm — mitm removed');
      this.emit();
    }, 600);
  }

  private execStackRow(i: number) {
    if (i < 0 || i >= this.stack.length) return;
    const row = this.stack[i];
    if (row.status === 'exited') {
      row.status = 'up';
      this.setStatus(`started ${row.svc} (was exited)`);
      this.emit();
      this.schedule(() => this.startLeave('shell', () => this.setStatus(`dc exec --service ${row.svc} — shell exited`)), 500);
      return;
    }
    this.startLeave('shell', () => this.setStatus(`dc exec --service ${row.svc} — shell exited`));
  }

  private openLogs() {
    this.view = 'logs';
    this.logOffset = Math.max(0, this.logLines.length - 6);
    this.hint = 'q back · j/k scroll · f follow';
    this.emit();
    this.schedule(() => this.appendLog(), 1800);
  }

  private appendLog() {
    if (this.view !== 'logs' || !this.logFollow) return;
    const verbs = ['GET', 'POST', 'GET', 'GET'];
    const paths = ['/api/health', '/api/exec', '/static/chunk.js', '/'];
    const i = this.logLines.length % paths.length;
    const ts = new Date().toISOString();
    this.logLines.push(`${ts} ${verbs[i]} ${paths[i]} 200 ${8 + (i * 7)}ms`);
    if (this.logFollow) this.logOffset = Math.max(0, this.logLines.length - 6);
    this.emit();
    this.schedule(() => this.appendLog(), 2200);
  }

  private reload() {
    this.refreshing = true;
    this.folderStatus = 'checking';
    this.emit();
    this.schedule(() => {
      this.refreshing = false;
      this.folderStatus = this.running ? 'running' : 'stopped';
      this.setStatus('reload — snapshot refreshed');
      this.emit();
    }, 550);
  }

  private startLeave(kind: BoardSnapshot['leaving'], done: () => void) {
    this.leaving = kind;
    this.emit();
    this.schedule(() => {
      this.leaving = '';
      done();
      this.emit();
    }, 650);
  }

  private syncLogTarget() {
    const row = this.stack[this.cursor];
    if (!row) return;
    this.logName = row.name;
  }

  private tickPulse() {
    if (!this.running || this.view !== 'board') {
      this.schedule(() => this.tickPulse(), 2200);
      return;
    }
    const cpu = 10 + Math.random() * 8;
    this.topRows[0] = { ...this.topRows[0], cpu: Math.round(cpu * 10) / 10 };
    this.emit();
    this.schedule(() => this.tickPulse(), 2200);
  }

  private setStatus(msg: string) {
    this.status = msg;
    this.emit();
    if (!msg) return;
    this.schedule(() => {
      if (this.status === msg) {
        this.status = '';
        this.emit();
      }
    }, 3200);
  }

  private schedule(fn: () => void, ms: number) {
    const id = window.setTimeout(() => {
      this.timers = this.timers.filter((t) => t !== id);
      fn();
    }, ms);
    this.timers.push(id);
  }

  private emit() {
    this.onChange(this.snapshot());
  }
}
