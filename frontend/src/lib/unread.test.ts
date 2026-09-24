import { describe, expect, it } from 'vitest';
import { nextUnread } from './unread';
import type { State, Terminal } from './types';

const term = (id: string, state: State): Terminal => ({
  id,
  name: 'shell',
  preset: '',
  cwd: '',
  state,
  sessionId: '',
  title: '',
  createdAt: '2026-09-24T00:00:00Z',
});

describe('nextUnread', () => {
  it('marks a terminal that stopped running while nobody looks at it', () => {
    const next = nextUnread({ a: 'running', b: 'running' }, [term('a', 'waiting'), term('b', 'idle')], new Set(), null);
    expect([...next].sort()).toEqual(['a', 'b']);
  });
  it('does not mark the terminal being viewed', () => {
    expect([...nextUnread({ a: 'running' }, [term('a', 'idle')], new Set(), 'a')]).toEqual([]);
  });
  it('ignores transitions that do not come from running', () => {
    const next = nextUnread({ a: 'shell', b: 'idle' }, [term('a', 'idle'), term('b', 'running')], new Set(), null);
    expect([...next]).toEqual([]);
  });
  it('ignores terminals seen for the first time', () => {
    expect([...nextUnread({}, [term('a', 'waiting')], new Set(), null)]).toEqual([]);
  });
  it('clears the terminal being viewed and the ones that closed', () => {
    const next = nextUnread({ a: 'idle', b: 'idle' }, [term('a', 'idle'), term('b', 'idle')], new Set(['a', 'b', 'gone']), 'b');
    expect([...next]).toEqual(['a']);
  });
  it('returns the same set when nothing changes', () => {
    const unread = new Set(['a']);
    expect(nextUnread({ a: 'idle' }, [term('a', 'idle')], unread, null)).toBe(unread);
  });
});
