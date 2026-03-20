import { LitElement, html, css } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';
import type { Settings, UpdateInfo, RateLimits, RateWindow } from './types';

@customElement('status-bar')
export class StatusBar extends LitElement {
  @property({ type: Number }) count = 0;
  @state() private showSettings = false;
  @state() private settings: Settings = {
    notifyConfirm: true, notifyWaiting: false,
    badgeConfirm: true, badgeWaiting: true, badgeActive: false,
    showRateLimits: false,
  };
  @state() private updateInfo: UpdateInfo | null = null;
  @state() private rateLimits: RateLimits | null = null;

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
      color: var(--text-secondary);
      padding: 2px 4px;
      border-radius: 4px;
      transition: color 0.15s, background 0.15s;
      user-select: none;
    }
    .gear:hover { color: var(--text); background: var(--hover); }
    .settings-panel {
      border-top: 1px solid var(--border);
      padding: 10px 14px;
      display: flex;
      flex-direction: column;
      gap: 8px;
    }
    .section-label {
      font-size: 11px;
      color: var(--text-tertiary, #484f58);
    }
    .setting-row {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 12px;
      color: var(--text-secondary, #8b949e);
      cursor: pointer;
      padding-left: 2px;
    }
    .setting-row:hover { color: var(--text); }
    .toggle {
      position: relative;
      width: 20px;
      height: 12px;
      flex-shrink: 0;
      border-radius: 6px;
      background: #30363d;
      border: 1px solid rgba(255,255,255,0.1);
      transition: background 0.2s, border-color 0.2s;
    }
    .toggle.on {
      background: #58a6ff;
      border-color: #58a6ff;
    }
    .toggle::after {
      content: '';
      position: absolute;
      top: 1px;
      left: 1px;
      width: 8px;
      height: 8px;
      border-radius: 50%;
      background: #fff;
      transition: transform 0.2s;
    }
    .toggle.on::after {
      transform: translateX(8px);
    }
    .gear-wrap {
      position: relative;
      display: inline-block;
    }
    .update-dot {
      position: absolute;
      top: -1px;
      right: -1px;
      width: 6px;
      height: 6px;
      border-radius: 50%;
      background: #f08000;
    }
    .update-banner {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 8px 12px;
      background: rgba(240, 128, 0, 0.1);
      border-top: 1px solid rgba(240, 128, 0, 0.2);
      font-size: 12px;
      color: #f0a050;
    }
    .update-banner a {
      color: #f08000;
      text-decoration: none;
      font-weight: 500;
      cursor: pointer;
    }
    .update-banner a:hover {
      text-decoration: underline;
    }
    .rate-limits {
      padding: 6px 12px 4px;
      display: flex;
      flex-direction: column;
      gap: 4px;
      border-top: 1px solid var(--border, rgba(255,255,255,0.08));
    }
    .rate-row {
      display: flex;
      align-items: center;
      gap: 6px;
      font-size: 11px;
      color: var(--text-secondary, #8b949e);
    }
    .rate-label {
      width: 18px;
      flex-shrink: 0;
      font-weight: 500;
      font-size: 10px;
      color: var(--text-tertiary, #484f58);
    }
    .rate-track {
      flex: 1;
      height: 4px;
      background: var(--border, rgba(255,255,255,0.08));
      border-radius: 2px;
      overflow: hidden;
    }
    .rate-fill {
      height: 100%;
      border-radius: 2px;
      transition: width 0.3s ease, background 0.3s ease;
    }
    .rate-pct {
      width: 28px;
      text-align: right;
      font-size: 10px;
      font-variant-numeric: tabular-nums;
    }
  `;

  connectedCallback() {
    super.connectedCallback();
    // @ts-ignore
    window.go?.main?.App?.GetSettings().then((s: Settings) => {
      if (s) this.settings = s;
    });
    // @ts-ignore
    window.runtime?.EventsOn('update:available', (info: UpdateInfo) => {
      this.updateInfo = info;
    });
    // @ts-ignore
    window.runtime?.EventsOn('ratelimits:updated', (rl: RateLimits | null) => {
      this.rateLimits = rl;
    });
  }

  private _toggleSettings() {
    this.showSettings = !this.showSettings;
  }

  private _toggle(key: keyof Settings) {
    this.settings = { ...this.settings, [key]: !this.settings[key] };
    // @ts-ignore
    window.go?.main?.App?.UpdateSettings(this.settings);
  }

  private async _toggleRateLimits() {
    const enabling = !this.settings.showRateLimits;
    if (enabling) {
      // @ts-ignore
      const installed = await window.go?.main?.App?.IsBridgeInstalled();
      if (!installed) {
        // @ts-ignore
        await window.go?.main?.App?.EnableRateLimits();
      }
    } else {
      // @ts-ignore
      await window.go?.main?.App?.DisableRateLimits();
      this.rateLimits = null;
    }
    this._toggle('showRateLimits');
  }

  private _openUpdate() {
    if (this.updateInfo) {
      // @ts-ignore
      window.go?.main?.App?.OpenURL(this.updateInfo.downloadURL);
    }
  }

  private _barColor(pct: number): string {
    if (pct >= 90) return '#f85149';
    if (pct >= 75) return '#f08000';
    if (pct >= 50) return '#e3b341';
    return '#34d058';
  }

  private _renderBar(label: string, w: RateWindow) {
    const pct = Math.min(100, Math.max(0, w.used_percentage));
    const color = this._barColor(pct);
    return html`
      <div class="rate-row">
        <span class="rate-label">${label}</span>
        <div class="rate-track">
          <div class="rate-fill" style="width:${pct}%; background:${color}"></div>
        </div>
        <span class="rate-pct">${pct}%</span>
      </div>
    `;
  }

  render() {
    const label = this.count === 1 ? 'session' : 'sessions';
    const s = this.settings;
    const rl = this.rateLimits;
    return html`
      ${s.showRateLimits && rl?.dataAvailable ? html`
        <div class="rate-limits">
          ${rl.five_hour ? this._renderBar('5h', rl.five_hour) : ''}
          ${rl.seven_day ? this._renderBar('7d', rl.seven_day) : ''}
        </div>
      ` : ''}
      <div class="bar">
        <span class="bar-label">${this.count} active ${label}</span>
        <span class="gear-wrap">
          <span class="gear" @click=${this._toggleSettings}>&#x2699;</span>
          ${this.updateInfo ? html`<span class="update-dot"></span>` : ''}
        </span>
      </div>
      ${this.showSettings ? html`
        <div class="settings-panel">
          ${this.updateInfo ? html`
            <div class="update-banner">
              <span>Vigil ${this.updateInfo.version} available</span>
              <a @click=${() => this._openUpdate()}>Download</a>
            </div>
          ` : ''}
          <span class="section-label">Notifications</span>
          <label class="setting-row" @click=${() => this._toggle('notifyConfirm')}>
            <span class="toggle ${s.notifyConfirm ? 'on' : ''}"></span>
            Needs confirmation
          </label>
          <label class="setting-row" @click=${() => this._toggle('notifyWaiting')}>
            <span class="toggle ${s.notifyWaiting ? 'on' : ''}"></span>
            Waiting for input
          </label>
          <span class="section-label">Badge</span>
          <label class="setting-row" @click=${() => this._toggle('badgeConfirm')}>
            <span class="toggle ${s.badgeConfirm ? 'on' : ''}"></span>
            Needs confirmation
          </label>
          <label class="setting-row" @click=${() => this._toggle('badgeWaiting')}>
            <span class="toggle ${s.badgeWaiting ? 'on' : ''}"></span>
            Waiting for input
          </label>
          <label class="setting-row" @click=${() => this._toggle('badgeActive')}>
            <span class="toggle ${s.badgeActive ? 'on' : ''}"></span>
            Active sessions
          </label>
          <span class="section-label">Rate limits</span>
          <label class="setting-row" @click=${() => this._toggleRateLimits()}>
            <span class="toggle ${s.showRateLimits ? 'on' : ''}"></span>
            Show usage
          </label>
        </div>
      ` : ''}
    `;
  }
}
