export const META_ENTER = '\x1b\r';

export type KeyAction = { kind: 'send'; data: string } | { kind: 'paste' } | { kind: 'pass' };

type Keyish = Pick<KeyboardEvent, 'type' | 'key' | 'altKey' | 'ctrlKey' | 'shiftKey'>;

export function terminalKey(e: Keyish): KeyAction {
  if (e.type !== 'keydown') return { kind: 'pass' };
  if (e.key === 'Enter' && (e.altKey || e.ctrlKey)) return { kind: 'send', data: META_ENTER };
  if (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === 'v') return { kind: 'paste' };
  return { kind: 'pass' };
}

// Keys we handle ourselves must also skip the browser default: xterm stops at a false return
// without cancelling, and in WebView2 Ctrl+Shift+V is a native paste that xterm would paste again.
export function keyHandler(write: (data: string) => void, paste: () => void) {
  return (e: KeyboardEvent): boolean => {
    const action = terminalKey(e);
    if (action.kind === 'pass') return true;
    e.preventDefault();
    if (action.kind === 'send') write(action.data);
    else paste();
    return false;
  };
}
