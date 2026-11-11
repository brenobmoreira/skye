// flipDeltas gives, for each item that was there before and moved, the offset that draws it back
// where it was (old top minus new top). The list starts the item there and lets it slide to its
// new place: the push you see while dragging.
export function flipDeltas(before: Map<string, number>, after: Map<string, number>): Map<string, number> {
  const out = new Map<string, number>();
  for (const [id, top] of after) {
    const was = before.get(id);
    if (was !== undefined && was !== top) out.set(id, was - top);
  }
  return out;
}
