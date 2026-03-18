import { LitElement, html, css } from 'lit';
import { customElement, property } from 'lit/decorators.js';

@customElement('status-bar')
export class StatusBar extends LitElement {
  @property({ type: Number }) count = 0;

  static styles = css`
    :host {
      display: block;
      padding: 8px 12px;
      border-top: 1px solid var(--border, rgba(255,255,255,0.08));
      font-size: 12px;
      color: var(--text-secondary, #8b949e);
    }
  `;

  render() {
    const label = this.count === 1 ? 'session' : 'sessions';
    return html`${this.count} active ${label}`;
  }
}
