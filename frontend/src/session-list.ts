import { LitElement, html, css } from 'lit';
import { customElement, state } from 'lit/decorators.js';
import type { Session } from './types';
import './session-card';
import './status-bar';
import './history-list';

const STATUS_ORDER: Record<string, number> = { confirm: 0, active: 1, waiting: 2, idle: 3 };

type Tab = 'active' | 'history';

@customElement('session-list')
export class SessionList extends LitElement {
  @state() private sessions: Session[] = [];
  @state() private tab: Tab = 'active';

  static styles = css`
    :host {
      display: flex;
      flex-direction: column;
      height: 100vh;
      overflow: hidden;
    }
    .tabs {
      display: flex;
      border-bottom: 1px solid rgba(255,255,255,0.08);
      flex-shrink: 0;
    }
    .tab {
      flex: 1;
      padding: 7px 0;
      text-align: center;
      font-size: 12px;
      color: #8b949e;
      cursor: pointer;
      transition: color 0.15s, border-bottom 0.15s;
      border-bottom: 2px solid transparent;
      user-select: none;
    }
    .tab.active-tab {
      color: #e6edf3;
      border-bottom: 2px solid #58a6ff;
    }
    .tab:hover:not(.active-tab) { color: #c9d1d9; }
    .list {
      flex: 1;
      overflow-y: auto;
    }
    .list::-webkit-scrollbar { width: 6px; }
    .list::-webkit-scrollbar-thumb {
      background: rgba(128,128,128,0.3);
      border-radius: 3px;
    }
    .empty {
      display: flex;
      align-items: center;
      justify-content: center;
      height: 100%;
      color: var(--text-secondary, #8b949e);
      font-size: 13px;
    }
    history-list {
      flex: 1;
      overflow: hidden;
    }
  `;

  connectedCallback() {
    super.connectedCallback();
    // @ts-ignore
    if (window.go?.main?.App?.GetSessions) {
      // @ts-ignore
      window.go.main.App.GetSessions().then((s: Session[]) => {
        this.sessions = s || [];
      });
    }
    // @ts-ignore
    if (window.runtime?.EventsOn) {
      // @ts-ignore
      window.runtime.EventsOn('sessions:updated', (sessions: Session[]) => {
        this.sessions = sessions || [];
      });
    }
  }

  private _activeLabel() {
    const count = this.sessions.filter(s => s.status !== 'idle').length;
    return count > 0 ? `Active (${count})` : 'Active';
  }

  render() {
    const sorted = [...this.sessions].sort((a, b) => {
      const d = (STATUS_ORDER[a.status] ?? 9) - (STATUS_ORDER[b.status] ?? 9);
      if (d !== 0) return d;
      return b.startedAt - a.startedAt;
    });

    return html`
      <div class="tabs">
        <div class="tab ${this.tab === 'active' ? 'active-tab' : ''}"
             @click=${() => { this.tab = 'active'; }}>
          ${this._activeLabel()}
        </div>
        <div class="tab ${this.tab === 'history' ? 'active-tab' : ''}"
             @click=${() => { this.tab = 'history'; }}>
          History
        </div>
      </div>
      ${this.tab === 'active' ? html`
        <div class="list">
          ${sorted.length === 0
            ? html`<div class="empty">No active sessions</div>`
            : sorted.map(s => html`<session-card .session=${s}></session-card>`)}
        </div>
        <status-bar .count=${sorted.length}></status-bar>
      ` : html`
        <history-list></history-list>
      `}
    `;
  }
}
