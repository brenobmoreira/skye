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

export interface RowBox { id: string; top: number; height: number }
export interface Spot { id: string; side: 'before' | 'after' }

// dropSpot says where a dragged item lands for a pointer at y: before the first of the others
// whose middle is below the pointer, or after the last. rows are the others, without the dragged
// one, in their order and where layout puts them (not where an animation draws them). Halves
// decide, not the row under the pointer, so a tall item passing a short one does not swing back.
export function dropSpot(rows: RowBox[], y: number): Spot | null {
  if (rows.length === 0) return null;
  for (const r of rows) {
    if (y < r.top + r.height / 2) return { id: r.id, side: 'before' };
  }
  return { id: rows[rows.length - 1].id, side: 'after' };
}

// translateYOf reads the vertical offset of a computed transform: where an animation draws an
// item away from its place in the layout.
export function translateYOf(transform: string): number {
  const m = /^matrix(3d)?\(([^)]*)\)$/.exec(transform.trim());
  if (!m) return 0;
  const v = m[2].split(',').map(Number);
  const y = m[1] ? v[13] : v[5];
  return Number.isFinite(y) ? y : 0;
}
