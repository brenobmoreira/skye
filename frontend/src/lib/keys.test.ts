import { describe, expect, it } from 'vitest';
import { META_ENTER, composerSubmits, terminalKey } from './keys';

const key = (over: Partial<KeyboardEvent>) =>
  ({ type: 'keydown', key: 'a', altKey: false, ctrlKey: false, shiftKey: false, ...over }) as KeyboardEvent;

describe('terminalKey', () => {
  it('turns Alt+Enter and Ctrl+Enter into meta-enter', () => {
    expect(terminalKey(key({ key: 'Enter', altKey: true }))).toEqual({ kind: 'send', data: META_ENTER });
    expect(terminalKey(key({ key: 'Enter', ctrlKey: true }))).toEqual({ kind: 'send', data: META_ENTER });
  });
  it('asks for paste on Ctrl+Shift+V', () => {
    expect(terminalKey(key({ key: 'V', ctrlKey: true, shiftKey: true }))).toEqual({ kind: 'paste' });
  });
  it('passes plain keys and keyup through', () => {
    expect(terminalKey(key({ key: 'Enter' }))).toEqual({ kind: 'pass' });
    expect(terminalKey(key({ type: 'keyup', key: 'Enter', altKey: true }))).toEqual({ kind: 'pass' });
  });
});

describe('composerSubmits', () => {
  it('submits only on Ctrl+Enter', () => {
    expect(composerSubmits(key({ key: 'Enter', ctrlKey: true }))).toBe(true);
    expect(composerSubmits(key({ key: 'Enter' }))).toBe(false);
  });
});
