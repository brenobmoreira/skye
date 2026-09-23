export const META_ENTER = '\x1b\r';

export type KeyAction = { kind: 'send'; data: string } | { kind: 'paste' } | { kind: 'pass' };

type Keyish = Pick<KeyboardEvent, 'type' | 'key' | 'altKey' | 'ctrlKey' | 'shiftKey'>;

export function terminalKey(e: Keyish): KeyAction {
  if (e.type !== 'keydown') return { kind: 'pass' };
  if (e.key === 'Enter' && (e.altKey || e.ctrlKey)) return { kind: 'send', data: META_ENTER };
  if (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === 'v') return { kind: 'paste' };
  return { kind: 'pass' };
}

export function composerSubmits(e: Pick<KeyboardEvent, 'key' | 'ctrlKey'>): boolean {
  return e.key === 'Enter' && e.ctrlKey;
}
