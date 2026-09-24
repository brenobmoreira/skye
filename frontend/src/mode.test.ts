import { expect, it } from 'vitest';
import { detectMode } from './mode';

it('picks the linux window when the wails bridge is bound', () => {
  expect(detectMode({ go: { main: { Bridge: {} } } })).toBe('linux-window');
});

it('picks the windows app when the wails shell is bound', () => {
  expect(detectMode({ go: { main: { Shell: {} } } })).toBe('windows-app');
});

it('falls back to the browser without wails bindings', () => {
  expect(detectMode({})).toBe('browser');
  expect(detectMode({ go: {} })).toBe('browser');
  expect(detectMode({ go: { main: {} } })).toBe('browser');
});
