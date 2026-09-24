import { describe, expect, it } from 'vitest';
import { moveWithinGroup } from './order';
import type { State, Terminal } from './types';

const term = (id: string, state: State): Terminal => ({
  id, name: id, preset: '', cwd: '', state, sessionId: '', title: '', createdAt: '', order: 0,
});
const list = [term('w1', 'waiting'), term('i1', 'idle'), term('i2', 'idle'), term('i3', 'idle'), term('r1', 'running')];
const ids = (l: Terminal[] | null) => l?.map((t) => t.id).join(',');

describe('moveWithinGroup', () => {
  it('moves a terminal down to the target place', () => {
    expect(ids(moveWithinGroup(list, 'i1', 'i3'))).toBe('w1,i2,i3,i1,r1');
  });
  it('moves a terminal up to the target place', () => {
    expect(ids(moveWithinGroup(list, 'i3', 'i1'))).toBe('w1,i3,i1,i2,r1');
  });
  it('refuses to cross state groups', () => {
    expect(moveWithinGroup(list, 'i1', 'w1')).toBeNull();
    expect(moveWithinGroup(list, 'r1', 'i2')).toBeNull();
  });
  it('ignores a drop on itself or on unknown ids', () => {
    expect(moveWithinGroup(list, 'i1', 'i1')).toBeNull();
    expect(moveWithinGroup(list, 'nope', 'i1')).toBeNull();
  });
});
