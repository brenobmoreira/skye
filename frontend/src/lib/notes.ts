import type { DoneRecord, NoteItem, Notes, Repeat } from './types';

export type Row = { kind: 'item'; item: NoteItem; done: boolean } | { kind: 'done'; record: DoneRecord };

export interface Groups {
  pinned: Row[];
  overdue: Row[];
  today: Row[];
  upcoming: Row[];
  undated: Row[];
  // The Completas tab: checked notes by the day they were checked, newest first.
  done: { day: string; rows: Row[] }[];
}

const pad = (n: number) => String(n).padStart(2, '0');

// dayOf is the local calendar day, the same way the Go side writes dates.
export function dayOf(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

const hmOf = (d: Date) => `${pad(d.getHours())}:${pad(d.getMinutes())}`;

function shiftDay(day: string, by: number): string {
  const [y, m, d] = day.split('-').map(Number);
  return dayOf(new Date(y, m - 1, d + by));
}

type Slot = 'pinned' | 'overdue' | 'today' | 'upcoming' | 'undated';

function slotOf(item: NoteItem, today: string, hm: string): Slot {
  if (item.pinned) return 'pinned';
  if (!item.date) return 'undated';
  if (item.date < today) return 'overdue';
  if (item.date > today) return 'upcoming';
  // A note for today whose time has come is late, right after its reminder.
  return item.time && item.time <= hm ? 'overdue' : 'today';
}

// Inside a group the date leads and the list's own order, which dragging sets, breaks the ties.
const dated = (s: Slot) => s === 'overdue' || s === 'today' || s === 'upcoming';

export function groupNotes(notes: Notes, now: Date, historyDays = 30): Groups {
  const today = dayOf(now);
  const hm = hmOf(now);
  const oldest = Number.isFinite(historyDays) ? shiftDay(today, -historyDays) : '';
  const slots: Record<Slot, { row: Row; date: string; index: number }[]> = { pinned: [], overdue: [], today: [], upcoming: [], undated: [] };
  const past: { day: string; at: string; index: number; row: Row }[] = [];

  notes.items.forEach((item, index) => {
    if (item.doneAt) {
      const day = dayOf(new Date(item.doneAt));
      if (day >= oldest) past.push({ day, at: item.doneAt, index, row: { kind: 'item', item, done: true } });
      return;
    }
    slots[slotOf(item, today, hm)].push({ row: { kind: 'item', item, done: false }, date: item.date ?? '', index });
  });
  notes.done.forEach((record, i) => {
    const day = dayOf(new Date(record.doneAt));
    if (day >= oldest) past.push({ day, at: record.doneAt, index: notes.items.length + i, row: { kind: 'done', record } });
  });

  const sorted = (s: Slot) =>
    slots[s].sort((a, b) => (dated(s) && a.date !== b.date ? (a.date < b.date ? -1 : 1) : a.index - b.index)).map((k) => k.row);
  // Newest day first; inside a day, the list's order.
  past.sort((a, b) => (a.day !== b.day ? (a.day < b.day ? 1 : -1) : a.index - b.index));
  const done: Groups['done'] = [];
  for (const p of past) {
    const last = done[done.length - 1];
    if (last && last.day === p.day) last.rows.push(p.row);
    else done.push({ day: p.day, rows: [p.row] });
  }
  return { pinned: sorted('pinned'), overdue: sorted('overdue'), today: sorted('today'), upcoming: sorted('upcoming'), undated: sorted('undated'), done };
}

// canMove: a note can only be dragged among the open notes of its own group, and in a dated group
// only among the same day, where the list's order is what decides.
export function canMove(notes: Notes, from: string, to: string, now: Date): boolean {
  if (from === to) return false;
  const a = notes.items.find((i) => i.id === from);
  const b = notes.items.find((i) => i.id === to);
  if (!a || !b || a.doneAt || b.doneAt) return false;
  const today = dayOf(now);
  const hm = hmOf(now);
  const slot = slotOf(a, today, hm);
  if (slot !== slotOf(b, today, hm)) return false;
  return !dated(slot) || a.date === b.date;
}


const WEEKDAYS = ['domingo', 'segunda', 'terça', 'quarta', 'quinta', 'sexta', 'sábado'];
const SHORT = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb'];

export function repeatLabel(r: Repeat): string {
  switch (r.kind) {
    case 'daily':
      return 'todo dia';
    case 'weekdays':
      return 'dias úteis';
    case 'weekly': {
      const w = r.weekday ?? 0;
      return `${w === 0 || w === 6 ? 'todo' : 'toda'} ${WEEKDAYS[w]}`;
    }
    case 'monthly':
      return `todo dia ${r.day ?? 1}`;
  }
}

export function dueLabel(item: NoteItem, now: Date): string {
  if (!item.date) return '';
  const today = dayOf(now);
  const time = item.time ? ` ${item.time}` : '';
  if (item.date === today) return `Hoje${time}`;
  if (item.date === shiftDay(today, 1)) return `Amanhã${time}`;
  if (item.date === shiftDay(today, -1)) return `Ontem${time}`;
  const [y, m, d] = item.date.split('-').map(Number);
  if (item.date > today && item.date <= shiftDay(today, 6)) return `${SHORT[new Date(y, m - 1, d).getDay()]}${time}`;
  return `${pad(d)}/${pad(m)}${time}`;
}
