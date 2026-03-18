import { LitElement, html, css } from 'lit';
import { customElement, state } from 'lit/decorators.js';
import type { Session } from './types';
import './session-card';
import './status-bar';

const STATUS_ORDER: Record<string, number> = { confirm: 0, active: 1, waiting: 2, idle: 3 };

@customElement('session-list')
export class SessionList extends LitElement {
  @state() private sessions: Session[] = [];

  static styles = css`
    :host {
      display: flex;
      flex-direction: column;
      height: 100vh;
      overflow: hidden;
    }
    .list {
      flex: 1;
      overflow-y: auto;
    }
    .list::-webkit-scrollbar {
      width: 6px;
    }
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
  `;

  connectedCallback() {
    super.connectedCallback();
    // @ts-ignore — Wails runtime injected at runtime
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

  render() {
    const sorted = [...this.sessions].sort((a, b) => {
      const d = (STATUS_ORDER[a.status] ?? 9) - (STATUS_ORDER[b.status] ?? 9);
      if (d !== 0) return d;
      return b.startedAt - a.startedAt;
    });

    return html`
      <div class="list">
        ${sorted.length === 0
          ? html`<div class="empty">No active sessions</div>`
          : sorted.map(s => html`<session-card .session=${s}></session-card>`)}
      </div>
      <status-bar .count=${sorted.length}></status-bar>
    `;
  }
}
