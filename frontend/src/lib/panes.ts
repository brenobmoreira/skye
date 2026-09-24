// The main area shows one terminal, or two side by side; `focus` says which one gets the keys.
export type Panes = { left: string | null; right: string | null; focus: 'left' | 'right' };
export type Slot = 'full' | 'left' | 'right';

export const active = (p: Panes) => (p.focus === 'right' && p.right ? p.right : p.left);

const focusOn = (p: Panes, id: string): Panes | null => {
  if (id === p.left) return p.focus === 'left' ? p : { ...p, focus: 'left' };
  if (id === p.right) return p.focus === 'right' ? p : { ...p, focus: 'right' };
  return null;
};

// A plain click: the terminal takes the focused pane, or just gets the focus if it is on screen.
export function select(p: Panes, id: string): Panes {
  const on = focusOn(p, id);
  if (on) return on;
  return p.focus === 'right' && p.right ? { ...p, right: id } : { ...p, left: id, focus: 'left' };
}

// Ctrl+click: the terminal opens on the right, next to the one on the left.
export function selectSide(p: Panes, id: string): Panes {
  const on = focusOn(p, id);
  if (on) return on;
  if (!p.left) return { left: id, right: null, focus: 'left' };
  return { left: p.left, right: id, focus: 'right' };
}

export const closeSplit = (p: Panes): Panes => ({ left: active(p), right: null, focus: 'left' });

// Drops terminals that are gone; with nothing left on screen, shows the first terminal.
export function prune(p: Panes, ids: string[]): Panes {
  const has = (id: string | null) => id !== null && ids.includes(id);
  if ((p.left === null || has(p.left)) && (p.right === null || has(p.right)) && p.left !== null) return p;
  const kept = [p.left, p.right].filter(has) as string[];
  const left = kept[0] ?? ids[0] ?? null;
  const right = kept[1] ?? null;
  const focus = right && active(p) === right ? 'right' : 'left';
  const next: Panes = { left, right, focus };
  return next.left === p.left && next.right === p.right && next.focus === p.focus ? p : next;
}

export function slotOf(p: Panes, id: string, room: boolean): Slot | null {
  if (!p.right) return id === p.left ? 'full' : null;
  if (!room) return id === active(p) ? 'full' : null;
  if (id === p.left) return 'left';
  if (id === p.right) return 'right';
  return null;
}

export function visible(p: Panes, room: boolean): string[] {
  if (p.right && room) return [p.left, p.right].filter((id): id is string => id !== null);
  const id = active(p);
  return id ? [id] : [];
}
