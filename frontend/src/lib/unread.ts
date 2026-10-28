import type { State, Terminal } from './types';

const settled: State[] = ['waiting', 'idle'];

// Marks terminals that stopped running while nobody was looking at them; `viewing` holds the
// terminals on screen in a focused window.
export function nextUnread(prev: Record<string, State>, list: Terminal[], unread: Set<string>, viewing: string[]): Set<string> {
  const next = new Set<string>();
  for (const t of list) {
    if (viewing.includes(t.id)) continue;
    if (unread.has(t.id) || (prev[t.id] === 'running' && settled.includes(t.state))) next.add(t.id);
  }
  const same = next.size === unread.size && [...next].every((id) => unread.has(id));
  return same ? unread : next;
}

// Terminals marked unread by hand stay marked, on screen or not, until they are picked; marks of
// terminals that closed drop out.
export function withMarks(unread: Set<string>, marked: Set<string>, list: Terminal[]): Set<string> {
  if (marked.size === 0) return unread;
  const next = new Set(unread);
  for (const t of list) if (marked.has(t.id)) next.add(t.id);
  return next;
}
