import { describe, expect, it } from 'vitest';
import { META_ENTER, keyHandler, terminalKey } from './keys';

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

describe('keyHandler', () => {
  const setup = () => {
    const calls = { write: [] as string[], paste: 0 };
    const handle = keyHandler((data) => calls.write.push(data), () => { calls.paste++; });
    return { calls, handle };
  };
  const event = (over: Partial<KeyboardEvent>) => {
    const e = { ...key(over), prevented: false, preventDefault() { e.prevented = true; } };
    return e as unknown as KeyboardEvent & { prevented: boolean };
  };

  it('pastes once and blocks the browser paste on Ctrl+Shift+V', () => {
    const { calls, handle } = setup();
    const e = event({ key: 'V', ctrlKey: true, shiftKey: true });
    expect(handle(e)).toBe(false);
    expect(calls.paste).toBe(1);
    expect(e.prevented).toBe(true);
  });
  it('sends meta-enter and blocks the default', () => {
    const { calls, handle } = setup();
    const e = event({ key: 'Enter', altKey: true });
    expect(handle(e)).toBe(false);
    expect(calls.write).toEqual([META_ENTER]);
    expect(e.prevented).toBe(true);
  });
  it('lets other keys reach xterm untouched', () => {
    const { calls, handle } = setup();
    const e = event({ key: 'v', ctrlKey: true });
    expect(handle(e)).toBe(true);
    expect(calls).toEqual({ write: [], paste: 0 });
    expect(e.prevented).toBe(false);
  });
});
