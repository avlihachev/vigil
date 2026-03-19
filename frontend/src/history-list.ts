import { LitElement, html, css } from 'lit';
import { customElement, state } from 'lit/decorators.js';
import type { ProjectHistory, HistoricalSession } from './types';

@customElement('history-list')
export class HistoryList extends LitElement {
  @state() private groups: ProjectHistory[] = [];
  @state() private loading = true;
  @state() private expanded = new Set<string>();

  static styles = css`
    :host {
      display: flex;
      flex-direction: column;
      flex: 1;
      overflow-y: auto;
    }
    :host::-webkit-scrollbar { width: 6px; }
    :host::-webkit-scrollbar-thumb {
      background: rgba(128,128,128,0.3);
      border-radius: 3px;
    }
    .empty {
      display: flex;
      align-items: center;
      justify-content: center;
      height: 100%;
      color: var(--text-secondary);
      font-size: 13px;
    }
    .group {
      border-bottom: 1px solid var(--border);
    }
    .group-header {
      display: flex;
      align-items: center;
      gap: 6px;
      padding: 8px 12px;
      cursor: pointer;
      transition: background 0.15s;
    }
    .group-header:hover { background: var(--hover); }
    .chevron {
      font-size: 10px;
      color: var(--text-tertiary);
      width: 10px;
      flex-shrink: 0;
    }
    .group-name {
      font-size: 12px;
      font-weight: 600;
      color: var(--text);
      flex: 1;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .group-path {
      font-size: 10px;
      color: var(--text-secondary);
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      max-width: 140px;
    }
    .session-row {
      display: flex;
      align-items: center;
      gap: 6px;
      padding: 5px 12px 5px 28px;
      cursor: pointer;
      transition: background 0.15s;
    }
    .session-row:hover { background: var(--hover); }
    .tree-char {
      color: var(--text-tertiary);
      font-family: monospace;
      font-size: 11px;
      width: 10px;
      flex-shrink: 0;
    }
    .session-name {
      font-size: 11px;
      color: var(--text-secondary);
      flex: 1;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      font-style: italic;
    }
    .session-age {
      font-size: 10px;
      color: var(--text-tertiary);
      white-space: nowrap;
      flex-shrink: 0;
    }
    .session-tokens {
      display: flex;
      gap: 3px;
      flex-shrink: 0;
    }
    .tok-in  { font-size: 10px; color: #6e8cff; }
    .tok-out { font-size: 10px; color: #3fb950; }
  `;

  connectedCallback() {
    super.connectedCallback();
    // @ts-ignore
    window.go?.main?.App?.GetHistory().then((groups: ProjectHistory[]) => {
      this.groups = groups || [];
      this.groups.forEach(g => {
        if (g.sessions.length === 1) this.expanded.add(g.cwd);
      });
      this.loading = false;
    });
  }

  private _toggle(cwd: string) {
    const next = new Set(this.expanded);
    if (next.has(cwd)) next.delete(cwd);
    else next.add(cwd);
    this.expanded = next;
  }

  private _resume(cwd: string, sessionId: string) {
    // @ts-ignore
    window.go?.main?.App?.ResumeSession(cwd, sessionId);
  }

  private _age(ms: number): string {
    const diff = Date.now() - ms;
    const mins = Math.floor(diff / 60_000);
    if (mins < 60) return `${mins}m`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs}h`;
    return `${Math.floor(hrs / 24)}d`;
  }

  private _shortPath(p: string): string {
    return p.replace(/^\/Users\/[^/]+/, '~');
  }

  render() {
    if (this.loading) return html`<div class="empty">Loading...</div>`;
    if (!this.groups.length) return html`<div class="empty">No history</div>`;

    return html`${this.groups.map(g => {
      const open = this.expanded.has(g.cwd);
      return html`
        <div class="group">
          <div class="group-header" @click=${() => this._toggle(g.cwd)}>
            <span class="chevron">${open ? '\u25BC' : '\u25B6'}</span>
            <span class="group-name">${g.projectName}</span>
            <span class="group-path">${this._shortPath(g.cwd)}</span>
          </div>
          ${open ? html`
            <div class="sessions">
              ${g.sessions.map((s: HistoricalSession, i: number) => {
                const isLast = i === g.sessions.length - 1;
                return html`
                  <div class="session-row" @click=${() => this._resume(g.cwd, s.sessionId)}>
                    <span class="tree-char">${isLast ? '\u2514' : '\u251C'}</span>
                    <span class="session-name">${s.name || s.sessionId}</span>
                    <span class="session-age">${this._age(s.lastActiveAt)}</span>
                    ${s.tokensIn || s.tokensOut ? html`
                      <div class="session-tokens">
                        ${s.tokensIn  ? html`<span class="tok-in">\u2191${s.tokensIn}</span>`  : ''}
                        ${s.tokensOut ? html`<span class="tok-out">\u2193${s.tokensOut}</span>` : ''}
                      </div>
                    ` : ''}
                  </div>
                `;
              })}
            </div>
          ` : ''}
        </div>
      `;
    })}`;
  }
}
