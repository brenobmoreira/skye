// placeItem puts item `id` right before or after `target`; the same list back when that changes
// nothing, so a drag held over the same spot does not render again.
export function placeItem<T extends { id: string }>(list: T[], id: string, target: string, side: 'before' | 'after'): T[] {
  const src = list.findIndex((i) => i.id === id);
  if (src < 0 || id === target || !list.some((i) => i.id === target)) return list;
  const next = list.filter((i) => i.id !== id);
  const at = next.findIndex((i) => i.id === target) + (side === 'after' ? 1 : 0);
  next.splice(at, 0, list[src]);
  return next.every((i, k) => i === list[k]) ? list : next;
}
