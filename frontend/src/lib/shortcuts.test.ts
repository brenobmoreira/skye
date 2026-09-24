import { describe, expect, it } from 'vitest';
import { shortcut } from './shortcuts';
import type { State, Terminal } from './types';

const t = (id: string, state: State): Terminal => ({
  id, name: id, preset: '', cwd: '', state, sessionId: '', title: '', createdAt: '', order: 0,
});
const list = [t('a', 'waiting'), t('b', 'waiting'), t('c', 'idle'), t('d', 'running')];
const key = (over: Partial<KeyboardEvent>) =>
  ({ key: 'x', code: '', ctrlKey: true, shiftKey: false, altKey: false, metaKey: false, ...over }) as KeyboardEvent;

describe('shortcut', () => {
  it('opens the Nth terminal with Ctrl and a digit', () => {
    expect(shortcut(key({ key: '1', code: 'Digit1' }), list, 'c', new Set())).toBe('a');
    expect(shortcut(key({ key: '4', code: 'Digit4' }), list, 'a', new Set())).toBe('d');
  });
  it('reads the digit from the key position, whatever the keyboard layout', () => {
    expect(shortcut(key({ key: 'é', code: 'Digit2' }), list, 'a', new Set())).toBe('b');
  });
  it('leaves Ctrl+Shift with a digit to the terminal', () => {
    expect(shortcut(key({ key: '@', code: 'Digit2', shiftKey: true }), list, 'a', new Set())).toBeNull();
  });
  it('ignores digits past the end of the list and Ctrl+0', () => {
    expect(shortcut(key({ key: '5', code: 'Digit5' }), list, 'a', new Set())).toBeNull();
    expect(shortcut(key({ key: '0', code: 'Digit0' }), list, 'a', new Set())).toBeNull();
  });
  it('ignores digits without Ctrl or with Alt', () => {
    expect(shortcut(key({ key: '1', code: 'Digit1', ctrlKey: false }), list, 'a', new Set())).toBeNull();
    expect(shortcut(key({ key: '1', code: 'Digit1', altKey: true }), list, 'a', new Set())).toBeNull();
  });

  const next = key({ key: ' ', code: 'Space', shiftKey: true });
  it('jumps to the next waiting terminal after the open one, wrapping around', () => {
    expect(shortcut(next, list, 'c', new Set())).toBe('a');
    expect(shortcut(next, list, 'a', new Set())).toBe('b');
    expect(shortcut(next, list, 'b', new Set())).toBe('a');
  });
  it('falls back to the next unread terminal when nobody waits', () => {
    const calm = [t('a', 'idle'), t('b', 'running'), t('c', 'idle')];
    expect(shortcut(next, calm, 'a', new Set(['c']))).toBe('c');
    expect(shortcut(next, calm, 'a', new Set())).toBeNull();
  });
  it('does nothing when the only waiting terminal is already open', () => {
    expect(shortcut(next, [t('a', 'waiting'), t('b', 'idle')], 'a', new Set())).toBeNull();
  });
});
