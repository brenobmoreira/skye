import { createRoot } from 'react-dom/client';
import '@fontsource/jetbrains-mono/400.css';
import '@fontsource/cascadia-mono/400.css';
import '@xterm/xterm/css/xterm.css';
import './styles.css';
import { App } from './App';
import { isWindow } from './bridge';

createRoot(document.getElementById('root')!).render(<App />);

if (!isWindow() && 'serviceWorker' in navigator) {
  navigator.serviceWorker.register('sw.js').catch(() => {});
}
