import {css, html, LitElement} from 'lit'

export class MyElement extends LitElement {
  static get styles() {
    return css`
      :host {
        display: block;
        padding: 16px;
        font-family: -apple-system, BlinkMacSystemFont, sans-serif;
        color: #ccc;
      }
      h3 { margin: 0; }
    `
  }

  render() {
    return html`
      <h3>Claude Sessions Monitor</h3>
      <p>Loading...</p>
    `
  }
}

window.customElements.define('my-element', MyElement)
