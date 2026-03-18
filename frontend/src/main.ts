import './styles.css';
import './session-list';

// emit blur event to Go backend so it can auto-hide the window
window.addEventListener('blur', () => {
  // @ts-ignore
  window.runtime?.EventsEmit('window:blur');
});
