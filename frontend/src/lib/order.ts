import type { Terminal } from './types';

// Moves `from` to where `to` is, but only inside the same state group: the server keeps the
// groups in state order, so a move across them would snap back.
export function moveWithinGroup(list: Terminal[], from: string, to: string): Terminal[] | null {
  const src = list.findIndex((t) => t.id === from);
  const dst = list.findIndex((t) => t.id === to);
  if (src < 0 || dst < 0 || src === dst || list[src].state !== list[dst].state) return null;
  const next = [...list];
  const [moved] = next.splice(src, 1);
  next.splice(dst, 0, moved);
  return next;
}
