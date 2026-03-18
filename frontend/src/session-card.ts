import { LitElement, html, css } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import type { Session, Action } from './types';

@customElement('session-card')
export class SessionCard extends LitElement {
  @property({ type: Object }) session!: Session;

  static styles = css`
    :host {
      display: block;
      padding: 10px 12px;
      border-bottom: 1px solid var(--border, rgba(255,255,255,0.08));
      cursor: pointer;
      transition: background 0.15s;
    }
    :host(:hover) {
      background: var(--hover, rgba(255,255,255,0.05));
    }
    .header {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 13px;
    }
    .dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;
      flex-shrink: 0;
    }
    .dot.active { background: #34d058; }
    .dot.waiting { background: #f0c000; }
    .dot.idle { background: #6a737d; }
    .name {
      font-weight: 600;
      flex: 1;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .meta {
      color: var(--text-secondary, #8b949e);
      font-size: 12px;
      white-space: nowrap;
    }
    .path {
      color: var(--text-secondary, #8b949e);
      font-size: 11px;
      margin: 2px 0 0 16px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .actions {
      margin: 6px 0 0 16px;
      font-size: 11px;
      color: var(--text-secondary, #8b949e);
    }
    .action-line {
      display: flex;
      align-items: center;
      gap: 4px;
      padding: 1px 0;
    }
    .tree-char {
      color: var(--text-tertiary, #484f58);
      font-family: monospace;
      width: 12px;
      flex-shrink: 0;
    }
  `;

  render() {
    const s = this.session;
    if (!s) return html``;
    return html`
      <div class="header">
        <span class="dot ${s.status}"></span>
        <span class="name">${s.projectName}</span>
        <span class="meta">${s.source}</span>
        <span class="meta">${s.duration}</span>
      </div>
      <div class="path">${this._shortenPath(s.cwd)}</div>
      ${s.recentActions?.length ? html`
        <div class="actions">
          ${s.recentActions.map((a: Action, i: number) => html`
            <div class="action-line">
              <span class="tree-char">${i === s.recentActions.length - 1 ? '\u2514' : '\u251C'}</span>
              ${this._formatAction(a)}
            </div>
          `)}
        </div>
      ` : ''}
    `;
  }

  private _shortenPath(p: string): string {
    const home = '/Users/';
    const idx = p.indexOf(home);
    if (idx >= 0) {
      const afterHome = p.substring(idx + home.length);
      const slash = afterHome.indexOf('/');
      if (slash >= 0) {
        return '~' + afterHome.substring(slash);
      }
    }
    return p;
  }

  private _formatAction(a: Action): string {
    const labels: Record<string, string> = {
      edit: 'Edited',
      read: 'Read',
      bash: 'Ran',
      search: 'Searched',
      waiting: 'Waiting for input',
    };
    const label = labels[a.type] || a.type;
    if (a.type === 'waiting') return label;
    return `${label} ${a.target}${a.result ? ' (' + a.result + ')' : ''}`;
  }
}
