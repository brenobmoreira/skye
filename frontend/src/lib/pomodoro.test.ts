import { describe, expect, it } from 'vitest';
import { clock, phaseLabel, pillLabel } from './pomodoro';
import type { PomodoroState } from './types';

const settings = { focus: 25, break: 5, long: 15, every: 4 };
const state = (over: Partial<PomodoroState>): PomodoroState => ({ phase: 'idle', paused: false, remainingMs: 0, totalMs: 0, count: 0, settings, ...over });

describe('clock', () => {
  it('shows minutes and seconds, rounding up', () => {
    expect(clock(25 * 60_000)).toBe('25:00');
    expect(clock(245_001)).toBe('4:06');
    expect(clock(0)).toBe('0:00');
    expect(clock(-5)).toBe('0:00');
  });
});

describe('phaseLabel', () => {
  it('names the phase and where in the cycle', () => {
    expect(phaseLabel(state({ phase: 'focus', count: 1 }))).toBe('foco · 2 de 4');
    expect(phaseLabel(state({ phase: 'focus', count: 4 }))).toBe('foco · 1 de 4');
    expect(phaseLabel(state({ phase: 'break' }))).toBe('pausa');
    expect(phaseLabel(state({ phase: 'long' }))).toBe('pausa longa');
    expect(phaseLabel(state({ count: 0 }))).toBe('pronto para focar');
    expect(phaseLabel(state({ count: 1 }))).toBe('1 foco hoje');
    expect(phaseLabel(state({ count: 5 }))).toBe('5 focos hoje');
  });
});

describe('pillLabel', () => {
  it('is empty when idle and names the phase otherwise', () => {
    expect(pillLabel(state({}), 0)).toBe('');
    expect(pillLabel(state({ phase: 'focus' }), 760_000)).toBe('foco 12:40');
    expect(pillLabel(state({ phase: 'break' }), 190_000)).toBe('pausa 3:10');
    expect(pillLabel(state({ phase: 'long' }), 899_000)).toBe('pausa longa 14:59');
  });
});
