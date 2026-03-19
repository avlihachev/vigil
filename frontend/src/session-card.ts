import { LitElement, html, css } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import type { Session, Action } from './types';

const STATUS_COLOR: Record<string, string> = {
  active:  '#34d058',
  confirm: '#f08000',
  waiting: '#e3b341',
  idle:    '#30363d',
};

const ACTION_COLOR: Record<string, string> = {
  edit:    '#58a6ff',
  read:    '#79c0ff',
  bash:    '#e3b341',
  search:  '#bc8cff',
  waiting: '#8b949e',
  confirm: '#f08000',
};

const ACTION_ICON: Record<string, string> = {
  edit:    '✎',
  read:    '◎',
  bash:    '$',
  search:  '⌕',
  waiting: '…',
  confirm: '⚠',
};

@customElement('session-card')
export class SessionCard extends LitElement {
  @property({ type: Object }) session!: Session;

  static styles = css`
    :host {
      display: block;
      border-bottom: 1px solid rgba(255,255,255,0.06);
      cursor: pointer;
      transition: background 0.15s;
      position: relative;
    }
    :host(:hover) {
      background: rgba(255,255,255,0.04);
    }
    .accent {
      position: absolute;
      left: 0; top: 0; bottom: 0;
      width: 3px;
      border-radius: 0 2px 2px 0;
      transition: background 0.3s;
    }
    .body {
      padding: 9px 12px 9px 14px;
    }
    .header {
      display: flex;
      align-items: center;
      gap: 7px;
      font-size: 13px;
      min-width: 0;
    }
    .dot {
      width: 7px;
      height: 7px;
      border-radius: 50%;
      flex-shrink: 0;
      transition: box-shadow 0.3s;
    }
    .dot.active {
      background: #34d058;
      box-shadow: 0 0 6px 1px rgba(52,208,88,0.6);
      animation: pulse-green 2s ease-in-out infinite;
    }
    .dot.confirm {
      background: #f08000;
      box-shadow: 0 0 6px 1px rgba(240,128,0,0.7);
      animation: pulse-orange 1.2s ease-in-out infinite;
    }
    .dot.waiting {
      background: #e3b341;
      box-shadow: 0 0 4px rgba(227,179,65,0.4);
    }
    .dot.idle {
      background: #30363d;
    }
    @keyframes pulse-green {
      0%, 100% { box-shadow: 0 0 4px 1px rgba(52,208,88,0.4); }
      50%       { box-shadow: 0 0 9px 2px rgba(52,208,88,0.8); }
    }
    @keyframes pulse-orange {
      0%, 100% { box-shadow: 0 0 5px 1px rgba(240,128,0,0.5); }
      50%       { box-shadow: 0 0 10px 3px rgba(240,128,0,0.9); }
    }
    .name {
      font-weight: 600;
      font-size: 13px;
      flex: 1;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      color: #e6edf3;
    }
    .source {
      font-size: 10px;
      padding: 1px 6px;
      border-radius: 10px;
      background: rgba(255,255,255,0.08);
      color: #8b949e;
      white-space: nowrap;
      flex-shrink: 0;
    }
    .tokens {
      display: flex;
      gap: 4px;
      flex-shrink: 0;
    }
    .tok-in {
      font-size: 11px;
      color: #6e8cff;
      white-space: nowrap;
    }
    .tok-out {
      font-size: 11px;
      color: #3fb950;
      white-space: nowrap;
    }
    .duration {
      font-size: 11px;
      color: #6e7681;
      white-space: nowrap;
      flex-shrink: 0;
    }
    .slug {
      font-size: 10px;
      margin: 2px 0 0 15px;
      color: #6e7681;
      font-style: italic;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .path {
      font-size: 11px;
      margin: 1px 0 0 15px;
      color: #8b949e;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .actions {
      margin: 5px 0 0 15px;
    }
    .action-line {
      display: flex;
      align-items: center;
      gap: 5px;
      padding: 1px 0;
      font-size: 11px;
    }
    .tree-char {
      color: #30363d;
      font-family: monospace;
      width: 10px;
      flex-shrink: 0;
    }
    .action-badge {
      width: 14px;
      height: 14px;
      border-radius: 3px;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 9px;
      flex-shrink: 0;
    }
    .action-label {
      color: #8b949e;
    }
    .action-line.is-confirm .action-label {
      color: #f08000;
      font-weight: 500;
    }
    .action-line.is-waiting .action-label {
      color: #e3b341;
    }
  `;

  connectedCallback() {
    super.connectedCallback();
    this.addEventListener('click', () => {
      if (this.session) {
        // @ts-ignore
        window.go?.main?.App?.OpenSession(this.session.source, this.session.cwd, this.session.pid);
      }
    });
  }

  render() {
    const s = this.session;
    if (!s) return html``;
    const accentColor = STATUS_COLOR[s.status] ?? '#30363d';
    return html`
      <div class="accent" style="background:${accentColor}"></div>
      <div class="body">
        <div class="header">
          <span class="dot ${s.status}"></span>
          <span class="name">${s.projectName}</span>
          <span class="source">${s.source}</span>
          ${s.sibling ? html`<span class="source">${s.sibling}</span>` : ''}
          ${s.tokensIn || s.tokensOut ? html`
            <div class="tokens">
              ${s.tokensIn  ? html`<span class="tok-in">↑${s.tokensIn}</span>`  : ''}
              ${s.tokensOut ? html`<span class="tok-out">↓${s.tokensOut}</span>` : ''}
            </div>
          ` : ''}
          <span class="duration">${s.duration}</span>
        </div>
        ${s.name ? html`<div class="slug">${s.name}</div>` : ''}
        <div class="path">${this._shortenPath(s.cwd)}</div>
        ${s.recentActions?.length ? html`
          <div class="actions">
            ${s.recentActions.map((a: Action, i: number) => {
              const isLast    = i === s.recentActions.length - 1;
              const isConfirm = a.type === 'confirm';
              const isWaiting = a.type === 'waiting';
              const color     = ACTION_COLOR[a.type] ?? '#8b949e';
              const icon      = ACTION_ICON[a.type]  ?? '·';
              return html`
                <div class="action-line ${isConfirm ? 'is-confirm' : ''} ${isWaiting ? 'is-waiting' : ''}">
                  <span class="tree-char">${isLast ? '└' : '├'}</span>
                  <span class="action-badge" style="background:${color}22; color:${color}">${icon}</span>
                  <span class="action-label">${this._formatLabel(a)}</span>
                </div>
              `;
            })}
          </div>
        ` : ''}
      </div>
    `;
  }

  private _shortenPath(p: string): string {
    const m = p.match(/^\/Users\/[^/]+(\/.*)?$/);
    if (m) return '~' + (m[1] ?? '');
    return p;
  }

  private _formatLabel(a: Action): string {
    if (a.type === 'waiting') return 'Waiting for input';
    if (a.type === 'confirm') return 'Needs confirmation';
    const t = a.target;
    if (t.startsWith('mcp__')) {
      const parts = t.slice(5).split('__');
      if (parts.length >= 2) return parts[0] + ':' + parts.slice(1).join('_');
    }
    return t;
  }
}
