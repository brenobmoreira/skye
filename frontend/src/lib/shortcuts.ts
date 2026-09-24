import type { Terminal } from './types';

type Keyish = Pick<KeyboardEvent, 'code' | 'ctrlKey' | 'shiftKey' | 'altKey' | 'metaKey'>;

// The terminal a key combination opens: Ctrl+1..9 the Nth in the list, Ctrl+Shift+Space the next
// one waiting for the user (or, with nobody waiting, the next unread). Null leaves the key to the
// terminal.
export function shortcut(e: Keyish, list: Terminal[], activeId: string | null, unread: Set<string>): string | null {
  if (!e.ctrlKey || e.altKey || e.metaKey) return null;
  const digit = /^Digit([1-9])$/.exec(e.code);
  if (digit && !e.shiftKey) return list[Number(digit[1]) - 1]?.id ?? null;
  if (e.code === 'Space' && e.shiftKey) {
    return nextAfter(list, activeId, (t) => t.state === 'waiting') ?? nextAfter(list, activeId, (t) => unread.has(t.id));
  }
  return null;
}

function nextAfter(list: Terminal[], activeId: string | null, match: (t: Terminal) => boolean): string | null {
  const start = list.findIndex((t) => t.id === activeId);
  for (let i = 1; i <= list.length; i++) {
    const t = list[(start + i) % list.length];
    if (match(t) && t.id !== activeId) return t.id;
  }
  return null;
}
