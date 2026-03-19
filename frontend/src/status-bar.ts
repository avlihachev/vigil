import { LitElement, html, css } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';
import type { Settings } from './types';

@customElement('status-bar')
export class StatusBar extends LitElement {
  @property({ type: Number }) count = 0;
  @state() private showSettings = false;
  @state() private settings: Settings = {
    notifyConfirm: true, notifyWaiting: false,
    badgeConfirm: true, badgeWaiting: true, badgeActive: false,
  };

  static styles = css`
    :host {
      display: block;
      border-top: 1px solid var(--border, rgba(255,255,255,0.08));
      flex-shrink: 0;
    }
    .bar {
      display: flex;
      align-items: center;
      padding: 8px 12px;
      font-size: 12px;
      color: var(--text-secondary, #8b949e);
    }
    .bar-label { flex: 1; }
    .gear {
      cursor: pointer;
      font-size: 13px;
      color: #6e7681;
      padding: 2px 4px;
      border-radius: 4px;
      transition: color 0.15s, background 0.15s;
      user-select: none;
    }
    .gear:hover { color: #c9d1d9; background: rgba(255,255,255,0.06); }
    .settings-panel {
      border-top: 1px solid rgba(255,255,255,0.06);
      padding: 10px 14px;
      display: flex;
      flex-direction: column;
      gap: 8px;
    }
    .section-label {
      font-size: 10px;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      color: #6e7681;
    }
    .setting-row {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 12px;
      color: #8b949e;
      cursor: pointer;
      padding-left: 2px;
    }
    .setting-row:hover { color: #c9d1d9; }
    input[type="checkbox"] { cursor: pointer; accent-color: #58a6ff; }
  `;

  connectedCallback() {
    super.connectedCallback();
    // @ts-ignore
    window.go?.main?.App?.GetSettings().then((s: Settings) => {
      if (s) this.settings = s;
    });
  }

  private _toggleSettings() {
    this.showSettings = !this.showSettings;
  }

  private _update(key: keyof Settings, e: Event) {
    const checked = (e.target as HTMLInputElement).checked;
    this.settings = { ...this.settings, [key]: checked };
    // @ts-ignore
    window.go?.main?.App?.UpdateSettings(this.settings);
  }

  render() {
    const label = this.count === 1 ? 'session' : 'sessions';
    const s = this.settings;
    return html`
      <div class="bar">
        <span class="bar-label">${this.count} active ${label}</span>
        <span class="gear" @click=${this._toggleSettings}>&#x2699;</span>
      </div>
      ${this.showSettings ? html`
        <div class="settings-panel">
          <span class="section-label">Notifications</span>
          <label class="setting-row">
            <input type="checkbox" .checked=${s.notifyConfirm}
                   @change=${(e: Event) => this._update('notifyConfirm', e)} />
            Needs confirmation
          </label>
          <label class="setting-row">
            <input type="checkbox" .checked=${s.notifyWaiting}
                   @change=${(e: Event) => this._update('notifyWaiting', e)} />
            Waiting for input
          </label>
          <span class="section-label">Badge</span>
          <label class="setting-row">
            <input type="checkbox" .checked=${s.badgeConfirm}
                   @change=${(e: Event) => this._update('badgeConfirm', e)} />
            Needs confirmation
          </label>
          <label class="setting-row">
            <input type="checkbox" .checked=${s.badgeWaiting}
                   @change=${(e: Event) => this._update('badgeWaiting', e)} />
            Waiting for input
          </label>
          <label class="setting-row">
            <input type="checkbox" .checked=${s.badgeActive}
                   @change=${(e: Event) => this._update('badgeActive', e)} />
            Active sessions
          </label>
        </div>
      ` : ''}
    `;
  }
}
