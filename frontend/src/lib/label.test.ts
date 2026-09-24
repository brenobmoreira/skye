import { describe, expect, it } from 'vitest';
import { itemLabel } from './label';
import type { Terminal } from './types';

const term = (over: Partial<Terminal>): Terminal => ({
  id: 't1',
  name: 'shell',
  preset: '',
  cwd: '/home/breno/projects/skye',
  state: 'idle',
  sessionId: '',
  title: '',
  createdAt: '2026-09-24T00:00:00Z',
  order: 0,
  ...over,
});

describe('itemLabel', () => {
  it('shows the first prompt as the name of a default shell', () => {
    expect(itemLabel(term({ title: 'arrume o teste', state: 'running' }))).toEqual({ name: 'arrume o teste', sub: 'skye' });
  });
  it('shows the folder and the state when the shell has no session title', () => {
    expect(itemLabel(term({ state: 'shell' }))).toEqual({ name: 'skye', sub: 'terminal' });
  });
  it('falls back to shell and the state without folder', () => {
    expect(itemLabel(term({ cwd: '' }))).toEqual({ name: 'shell', sub: 'terminou' });
  });
  it('keeps a name given by the user', () => {
    expect(itemLabel(term({ name: 'api', title: 'arrume o teste' }))).toEqual({ name: 'api', sub: 'arrume o teste' });
  });
  it('keeps the preset name', () => {
    expect(itemLabel(term({ name: 'claude', preset: 'claude', title: 'oi' }))).toEqual({ name: 'claude', sub: 'oi' });
  });
  it('shows what a waiting terminal asks under the name', () => {
    expect(itemLabel(term({ title: 'arrume o teste', state: 'waiting', ask: 'Claude needs your permission to use Bash' }))).toEqual({
      name: 'arrume o teste',
      sub: 'Claude needs your permission to use Bash',
    });
    expect(itemLabel(term({ name: 'api', state: 'waiting', ask: 'precisa de permissão' })).sub).toBe('precisa de permissão');
  });
  it('ignores a stale ask once the terminal is not waiting', () => {
    expect(itemLabel(term({ state: 'running', ask: 'velho' })).sub).toBe('trabalhando');
  });
  it('shows what a running terminal is busy with', () => {
    expect(itemLabel(term({ title: 'arrume', state: 'running', activity: '2 subagentes' })).sub).toBe('2 subagentes');
    expect(itemLabel(term({ title: 'arrume', state: 'idle', activity: '1 subagente' })).sub).toBe('skye');
  });
});
