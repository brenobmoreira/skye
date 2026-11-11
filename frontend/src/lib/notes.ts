import type { DoneRecord, NoteItem, Notes, Repeat } from './types';

export type Row = { kind: 'item'; item: NoteItem; done: boolean } | { kind: 'done'; record: DoneRecord };

export interface Groups {
  overdue: Row[];
  today: Row[];
  upcoming: Row[];
  undated: Row[];
  history: { day: string; rows: Row[] }[];
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

type Slot = 'overdue' | 'today' | 'upcoming' | 'undated';

function slotOf(date: string | undefined, time: string | undefined, today: string, hm: string): Slot {
  if (!date) return 'undated';
  if (date < today) return 'overdue';
  if (date > today) return 'upcoming';
  // A note for today whose time has come is late, right after its reminder.
  return time && time <= hm ? 'overdue' : 'today';
}

interface Keyed {
  row: Row;
  date: string;
  time: string;
  created: string;
}

// Timed notes come before untimed ones on the same day; ties keep creation order.
function compare(a: Keyed, b: Keyed): number {
  if (a.date !== b.date) return a.date < b.date ? -1 : 1;
  if (a.time !== b.time) {
    if (!a.time) return 1;
    if (!b.time) return -1;
    return a.time < b.time ? -1 : 1;
  }
  return a.created < b.created ? -1 : a.created > b.created ? 1 : 0;
}

export function groupNotes(notes: Notes, now: Date, historyDays = 30): Groups {
  const today = dayOf(now);
  const hm = hmOf(now);
  const oldest = Number.isFinite(historyDays) ? shiftDay(today, -historyDays) : '';
  const slots: Record<Slot, Keyed[]> = { overdue: [], today: [], upcoming: [], undated: [] };
  const past: { day: string; at: string; row: Row }[] = [];

  for (const item of notes.items) {
    const row: Row = { kind: 'item', item, done: !!item.doneAt };
    if (item.doneAt) {
      const day = dayOf(new Date(item.doneAt));
      if (day !== today) {
        if (day >= oldest) past.push({ day, at: item.doneAt, row });
        continue;
      }
    }
    slots[slotOf(item.date, item.time, today, hm)].push({ row, date: item.date ?? '', time: item.time ?? '', created: item.created });
  }
  for (const record of notes.done) {
    const row: Row = { kind: 'done', record };
    const day = dayOf(new Date(record.doneAt));
    if (day !== today) {
      if (day >= oldest) past.push({ day, at: record.doneAt, row });
      continue;
    }
    slots[slotOf(record.prevDate, undefined, today, hm)].push({ row, date: record.prevDate, time: '', created: record.doneAt });
  }

  const sorted = (s: Slot) => slots[s].sort(compare).map((k) => k.row);
  past.sort((a, b) => (a.at < b.at ? 1 : a.at > b.at ? -1 : 0));
  const history: Groups['history'] = [];
  for (const p of past) {
    const last = history[history.length - 1];
    if (last && last.day === p.day) last.rows.push(p.row);
    else history.push({ day: p.day, rows: [p.row] });
  }
  return { overdue: sorted('overdue'), today: sorted('today'), upcoming: sorted('upcoming'), undated: sorted('undated'), history };
}

const WEEKDAYS = ['domingo', 'segunda', 'terça', 'quarta', 'quinta', 'sexta', 'sábado'];
const SHORT = ['dom', 'seg', 'ter', 'qua', 'qui', 'sex', 'sáb'];

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
  if (item.date === today) return `hoje${time}`;
  if (item.date === shiftDay(today, 1)) return `amanhã${time}`;
  if (item.date === shiftDay(today, -1)) return `ontem${time}`;
  const [y, m, d] = item.date.split('-').map(Number);
  if (item.date > today && item.date <= shiftDay(today, 6)) return `${SHORT[new Date(y, m - 1, d).getDay()]}${time}`;
  return `${pad(d)}/${pad(m)}${time}`;
}
