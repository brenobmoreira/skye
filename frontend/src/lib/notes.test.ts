import { describe, expect, it } from 'vitest';
import { dueLabel, groupNotes, repeatLabel, type Row } from './notes';
import type { DoneRecord, NoteItem, Notes } from './types';

const now = new Date(2026, 10, 11, 10, 0);
const at = (d: number, h = 9, m = 0) => new Date(2026, 10, d, h, m).toISOString();
let seq = 0;
const item = (over: Partial<NoteItem>): NoteItem => ({ id: `n${++seq}`, text: `t${seq}`, created: at(1, 8, seq), ...over });
const record = (over: Partial<DoneRecord>): DoneRecord => ({ id: `d${++seq}`, itemId: 'x', text: 'r', doneAt: at(11), prevDate: '2026-11-11', ...over });
const ids = (rows: Row[]) => rows.map((r) => (r.kind === 'item' ? r.item.id : r.record.id));
const notes = (items: NoteItem[], done: DoneRecord[] = []): Notes => ({ items, done });

describe('groupNotes', () => {
  it('puts each open note in its group', () => {
    const undated1 = item({});
    const undated2 = item({});
    const late = item({ date: '2026-11-10' });
    const today = item({ date: '2026-11-11' });
    const passed = item({ date: '2026-11-11', time: '09:30' });
    const later = item({ date: '2026-11-11', time: '14:00' });
    const fri = item({ date: '2026-11-13' });
    const thuNoon = item({ date: '2026-11-12', time: '12:00' });
    const thuMorning = item({ date: '2026-11-12', time: '08:00' });
    const g = groupNotes(notes([undated1, late, today, passed, later, fri, thuNoon, thuMorning, undated2]), now);
    expect(ids(g.undated)).toEqual([undated1.id, undated2.id]);
    expect(ids(g.overdue)).toEqual([late.id, passed.id]);
    expect(ids(g.today)).toEqual([later.id, today.id]);
    expect(ids(g.upcoming)).toEqual([thuMorning.id, thuNoon.id, fri.id]);
    expect(g.history).toEqual([]);
  });

  it('keeps a note checked today crossed in its group and moves older ones to history', () => {
    const doneToday = item({ date: '2026-11-11', doneAt: at(11, 9) });
    const doneYesterday = item({ doneAt: at(10, 18) });
    const g = groupNotes(notes([doneToday, doneYesterday]), now);
    expect(g.today).toEqual([{ kind: 'item', item: doneToday, done: true }]);
    expect(g.undated).toEqual([]);
    expect(g.history.map((h) => h.day)).toEqual(['2026-11-10']);
    expect(ids(g.history[0].rows)).toEqual([doneYesterday.id]);
  });

  it('shows a repeating check from today where the note was, and older ones in history', () => {
    const today = record({ prevDate: '2026-11-11', doneAt: at(11, 9) });
    const old = record({ prevDate: '2026-11-09', doneAt: at(9, 9) });
    const g = groupNotes(notes([], [today, old]), now);
    expect(g.today).toEqual([{ kind: 'done', record: today }]);
    expect(g.history.map((h) => h.day)).toEqual(['2026-11-09']);
  });

  it('orders history days newest first and drops what is past the window', () => {
    const a = item({ doneAt: at(9) });
    const b = item({ doneAt: at(10) });
    const ancient = item({ doneAt: new Date(2026, 9, 1, 9).toISOString() });
    const g = groupNotes(notes([a, b, ancient]), now, 30);
    expect(g.history.map((h) => h.day)).toEqual(['2026-11-10', '2026-11-09']);
    expect(groupNotes(notes([ancient]), now, Infinity).history.map((h) => h.day)).toEqual(['2026-10-01']);
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
    expect(dueLabel(item({ date: '2026-11-11', time: '14:00' }), now)).toBe('hoje 14:00');
    expect(dueLabel(item({ date: '2026-11-11' }), now)).toBe('hoje');
    expect(dueLabel(item({ date: '2026-11-12' }), now)).toBe('amanhã');
    expect(dueLabel(item({ date: '2026-11-12', time: '09:00' }), now)).toBe('amanhã 09:00');
    expect(dueLabel(item({ date: '2026-11-13' }), now)).toBe('sex');
    expect(dueLabel(item({ date: '2026-12-11' }), now)).toBe('11/12');
    expect(dueLabel(item({ date: '2026-11-10' }), now)).toBe('ontem');
    expect(dueLabel(item({}), now)).toBe('');
  });
});
