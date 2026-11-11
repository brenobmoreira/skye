import { describe, expect, it } from 'vitest';
import { canMove, dueLabel, groupNotes, moveItems, repeatLabel, type Row } from './notes';
import type { DoneRecord, NoteItem, Notes } from './types';

const now = new Date(2026, 10, 11, 10, 0);
const at = (d: number, h = 9, m = 0) => new Date(2026, 10, d, h, m).toISOString();
let seq = 0;
const item = (over: Partial<NoteItem>): NoteItem => ({ id: `n${++seq}`, text: `t${seq}`, created: at(1, 8, seq), ...over });
const record = (over: Partial<DoneRecord>): DoneRecord => ({ id: `d${++seq}`, itemId: 'x', text: 'r', doneAt: at(11), prevDate: '2026-11-11', ...over });
const ids = (rows: Row[]) => rows.map((r) => (r.kind === 'item' ? r.item.id : r.record.id));
const notes = (items: NoteItem[], done: DoneRecord[] = []): Notes => ({ items, done });

describe('groupNotes', () => {
  it('puts each open note in its group, by date and then in the list order', () => {
    const undated1 = item({});
    const late = item({ date: '2026-11-10' });
    const today = item({ date: '2026-11-11' });
    const passed = item({ date: '2026-11-11', time: '09:30' });
    const later = item({ date: '2026-11-11', time: '14:00' });
    const fri = item({ date: '2026-11-13' });
    const thuNoon = item({ date: '2026-11-12', time: '12:00' });
    const thuMorning = item({ date: '2026-11-12', time: '08:00' });
    const undated2 = item({});
    const g = groupNotes(notes([undated1, late, today, passed, later, fri, thuNoon, thuMorning, undated2]), now);
    expect(ids(g.pinned)).toEqual([]);
    expect(ids(g.undated)).toEqual([undated1.id, undated2.id]);
    expect(ids(g.overdue)).toEqual([late.id, passed.id]);
    expect(ids(g.today)).toEqual([today.id, later.id]);
    expect(ids(g.upcoming)).toEqual([thuNoon.id, thuMorning.id, fri.id]);
    expect(g.done).toEqual([]);
  });

  it('keeps pinned notes in their own group, in the list order', () => {
    const a = item({ date: '2026-11-10', pinned: true });
    const b = item({});
    const c = item({ pinned: true });
    const g = groupNotes(notes([a, b, c]), now);
    expect(ids(g.pinned)).toEqual([a.id, c.id]);
    expect(ids(g.overdue)).toEqual([]);
    expect(ids(g.undated)).toEqual([b.id]);
  });

  it('moves checked notes out of the open groups into the done days, today first', () => {
    const doneToday = item({ date: '2026-11-11', doneAt: at(11, 9) });
    const pinnedDone = item({ pinned: true, doneAt: at(11, 8) });
    const doneYesterday = item({ doneAt: at(10, 18) });
    const repeatToday = record({ prevDate: '2026-11-11', doneAt: at(11, 7) });
    const g = groupNotes(notes([doneToday, pinnedDone, doneYesterday], [repeatToday]), now);
    expect(ids(g.today)).toEqual([]);
    expect(ids(g.pinned)).toEqual([]);
    expect(g.done.map((d) => d.day)).toEqual(['2026-11-11', '2026-11-10']);
    expect(ids(g.done[0].rows)).toEqual([doneToday.id, pinnedDone.id, repeatToday.id]);
    expect(g.done[0].rows[0]).toEqual({ kind: 'item', item: doneToday, done: true });
  });

  it('orders done days newest first and drops what is past the window', () => {
    const a = item({ doneAt: at(9) });
    const b = item({ doneAt: at(10) });
    const ancient = item({ doneAt: new Date(2026, 9, 1, 9).toISOString() });
    const g = groupNotes(notes([a, b, ancient]), now, 30);
    expect(g.done.map((h) => h.day)).toEqual(['2026-11-10', '2026-11-09']);
    expect(groupNotes(notes([ancient]), now, Infinity).done.map((h) => h.day)).toEqual(['2026-10-01']);
  });
});

describe('canMove', () => {
  it('lets a note move only among the notes it shares a place with', () => {
    const p1 = item({ pinned: true, date: '2026-11-12' });
    const p2 = item({ pinned: true });
    const u1 = item({});
    const u2 = item({});
    const thu1 = item({ date: '2026-11-12' });
    const thu2 = item({ date: '2026-11-12', time: '08:00' });
    const fri = item({ date: '2026-11-13' });
    const list = notes([p1, p2, u1, u2, thu1, thu2, fri]);
    expect(canMove(list, p1.id, p2.id, now)).toBe(true);
    expect(canMove(list, u2.id, u1.id, now)).toBe(true);
    expect(canMove(list, thu2.id, thu1.id, now)).toBe(true);
    expect(canMove(list, thu1.id, fri.id, now)).toBe(false);
    expect(canMove(list, u1.id, p2.id, now)).toBe(false);
    expect(canMove(list, u1.id, u1.id, now)).toBe(false);
  });
});

describe('moveItems', () => {
  it('puts the note where the other one was, like the Go store', () => {
    const [a, b, c, d] = [item({}), item({}), item({}), item({})];
    expect(moveItems([a, b, c, d], d.id, b.id).map((x) => x.id)).toEqual([a.id, d.id, b.id, c.id]);
    expect(moveItems([a, d, b, c], a.id, c.id).map((x) => x.id)).toEqual([d.id, b.id, c.id, a.id]);
  });
});

describe('repeatLabel', () => {
  it('names each kind', () => {
    expect(repeatLabel({ kind: 'daily' })).toBe('todo dia');
    expect(repeatLabel({ kind: 'weekdays' })).toBe('dias úteis');
    expect(repeatLabel({ kind: 'weekly', weekday: 5 })).toBe('toda sexta');
    expect(repeatLabel({ kind: 'weekly', weekday: 0 })).toBe('todo domingo');
    expect(repeatLabel({ kind: 'monthly', day: 10 })).toBe('todo dia 10');
  });
});

describe('dueLabel', () => {
  it('says when, the short way', () => {
    expect(dueLabel(item({ date: '2026-11-11', time: '14:00' }), now)).toBe('Hoje 14:00');
    expect(dueLabel(item({ date: '2026-11-11' }), now)).toBe('Hoje');
    expect(dueLabel(item({ date: '2026-11-12' }), now)).toBe('Amanhã');
    expect(dueLabel(item({ date: '2026-11-12', time: '09:00' }), now)).toBe('Amanhã 09:00');
    expect(dueLabel(item({ date: '2026-11-13' }), now)).toBe('Sex');
    expect(dueLabel(item({ date: '2026-12-11' }), now)).toBe('11/12');
    expect(dueLabel(item({ date: '2026-11-10' }), now)).toBe('Ontem');
    expect(dueLabel(item({}), now)).toBe('');
  });
});
