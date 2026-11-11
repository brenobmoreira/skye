import { describe, expect, it } from 'vitest';
import { answers, overdue, OVERDUE_MIN } from './reply';
import type { Terminal } from './types';

const term = (over: Partial<Terminal>): Terminal => ({
  id: 't1',
  name: 'shell',
  preset: '',
  cwd: '/home/breno/projects/skye',
  state: 'idle',
  sessionId: 's1',
  title: '',
  createdAt: '2026-11-11T00:00:00Z',
  order: 0,
  turn: 1,
  ...over,
});

const since = '2026-11-11T12:00:00Z';
const after = (min: number) => Date.parse(since) + min * 60_000;

describe('overdue', () => {
  it('flags a question left for more than ten minutes', () => {
    expect(OVERDUE_MIN).toBe(10);
    expect(overdue(term({ since, question: true }), after(9))).toBe(false);
    expect(overdue(term({ since, question: true }), after(11))).toBe(true);
  });
  it('flags choices and a waiting terminal too', () => {
    expect(overdue(term({ since, options: [{ n: 1, label: 'a' }, { n: 2, label: 'b' }] }), after(11))).toBe(true);
    expect(overdue(term({ since, state: 'waiting' }), after(11))).toBe(true);
  });
  it('leaves alone a finished turn that asks nothing', () => {
    expect(overdue(term({ since, reply: 'Instalei.' }), after(30))).toBe(false);
    expect(overdue(term({ since, state: 'running' }), after(30))).toBe(false);
    expect(overdue(term({ question: true }), after(30))).toBe(false);
  });
});

describe('answers', () => {
  it('offers sim to a question', () => {
    expect(answers(term({ question: true }))).toEqual([{ label: 'sim', text: 'sim' }]);
  });
  it('offers one button per choice', () => {
    const options = [{ n: 1, label: 'Publicar tudo' }, { n: 2, label: 'Só críticos' }, { n: 3, label: 'Cancelar' }];
    expect(answers(term({ options }))).toEqual([
      { label: '1 Publicar tudo', text: '1' },
      { label: '2 Só críticos', text: '2' },
      { label: '3 Cancelar', text: '3' },
    ]);
  });
  it('offers nothing once answered or while working', () => {
    expect(answers(term({ question: true, answered: 'sim' }))).toEqual([]);
    expect(answers(term({ question: true, state: 'running' }))).toEqual([]);
    expect(answers(term({}))).toEqual([]);
  });
});
