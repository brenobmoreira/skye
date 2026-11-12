import { describe, expect, it } from 'vitest';
import { placeItem } from './order';
import type { State, Terminal } from './types';

const term = (id: string, state: State): Terminal => ({
  id, name: id, preset: '', cwd: '', state, sessionId: '', title: '', createdAt: '', order: 0,
});
const list = [term('w1', 'waiting'), term('i1', 'idle'), term('i2', 'idle'), term('i3', 'idle'), term('r1', 'running')];
const ids = (l: Terminal[] | null) => l?.map((t) => t.id).join(',');

describe('placeItem', () => {
  it('puts the item right before or after another one', () => {
    expect(ids(placeItem(list, 'i3', 'i1', 'before'))).toBe('w1,i3,i1,i2,r1');
    expect(ids(placeItem(list, 'i1', 'i3', 'after'))).toBe('w1,i2,i3,i1,r1');
    expect(ids(placeItem(list, 'i1', 'i2', 'after'))).toBe('w1,i2,i1,i3,r1');
  });
  it('gives back the same list when nothing moves', () => {
    expect(placeItem(list, 'i1', 'i2', 'before')).toBe(list);
    expect(placeItem(list, 'i2', 'i1', 'after')).toBe(list);
    expect(placeItem(list, 'i1', 'i1', 'after')).toBe(list);
    expect(placeItem(list, 'nope', 'i1', 'after')).toBe(list);
    expect(placeItem(list, 'i1', 'nope', 'after')).toBe(list);
  });
});
