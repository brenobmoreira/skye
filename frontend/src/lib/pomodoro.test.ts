import { describe, expect, it } from 'vitest';
import { breakLabel, clock, parseClock, collapsedLabel, endedLabel, phaseLabel, pillLabel, pillPhase } from './pomodoro';
import type { PomodoroState } from './types';

const settings = { focus: 25, break: 5, long: 15, every: 4 };
const state = (over: Partial<PomodoroState>): PomodoroState => ({ phase: 'idle', paused: false, remainingMs: 0, totalMs: 0, count: 0, stage: 0, nextMs: 0, settings, ...over });

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
    expect(phaseLabel(state({ phase: 'focus', count: 7, stage: 1 }))).toBe('Foco · 2 de 4');
    expect(phaseLabel(state({ phase: 'focus', count: 4, stage: 3 }))).toBe('Foco · 4 de 4');
    expect(phaseLabel(state({ phase: 'break' }))).toBe('Pausa');
    expect(phaseLabel(state({ phase: 'long' }))).toBe('Pausa longa');
    expect(phaseLabel(state({ count: 0 }))).toBe('Pronto para focar');
    expect(phaseLabel(state({ count: 1 }))).toBe('1 foco hoje');
    expect(phaseLabel(state({ count: 5 }))).toBe('5 focos hoje');
  });
});

describe('pillLabel', () => {
  it('is empty when idle and names the phase otherwise', () => {
    expect(pillLabel(state({}), 0)).toBe('');
    expect(pillLabel(state({ phase: 'focus' }), 760_000)).toBe('Foco 12:40');
    expect(pillLabel(state({ phase: 'break' }), 190_000)).toBe('Pausa 3:10');
    expect(pillLabel(state({ phase: 'long' }), 899_000)).toBe('Pausa longa 14:59');
  });
});

describe('pillPhase', () => {
  it('names the running phase without the time', () => {
    expect(pillPhase(state({ phase: 'focus' }))).toBe('Foco');
    expect(pillPhase(state({ phase: 'long' }))).toBe('Pausa longa');
    expect(pillPhase(state({}))).toBe('');
  });
});

describe('endedLabel', () => {
  it('says which phase ran out', () => {
    expect(endedLabel('focus')).toBe('Foco acabou');
    expect(endedLabel('break')).toBe('Pausa acabou');
    expect(endedLabel('long')).toBe('Pausa acabou');
  });
});

describe('breakLabel', () => {
  it('names the break that follows the chosen stage', () => {
    expect(breakLabel(1, 4)).toBe('Começar pausa');
    expect(breakLabel(3, 4)).toBe('Começar pausa');
    expect(breakLabel(4, 4)).toBe('Começar pausa longa');
  });
});

describe('collapsedLabel', () => {
  it('names the phase and whether it runs, ended or has not started', () => {
    expect(collapsedLabel(state({}), null)).toBe('Foco – Não iniciado');
    expect(collapsedLabel(state({ phase: 'focus' }), null)).toBe('Foco – Em andamento');
    expect(collapsedLabel(state({ phase: 'break' }), null)).toBe('Pausa – Em andamento');
    expect(collapsedLabel(state({ phase: 'long' }), null)).toBe('Pausa longa – Em andamento');
    expect(collapsedLabel(state({ phase: 'break' }), 'focus')).toBe('Foco – Concluído');
    expect(collapsedLabel(state({}), 'break')).toBe('Pausa – Concluída');
    expect(collapsedLabel(state({}), 'long')).toBe('Pausa longa – Concluída');
  });
});

describe('parseClock', () => {
  it('reads minutes, or minutes and seconds, into seconds', () => {
    expect(parseClock('18')).toBe(1080);
    expect(parseClock(' 18:30 ')).toBe(1110);
    expect(parseClock('0:45')).toBe(45);
    expect(parseClock('180')).toBe(10800);
  });
  it('refuses what is not a time in range', () => {
    for (const bad of ['', '0', '0:00', '181', '18:60', 'abc', '1:2:3', '-5', '18,5']) expect(parseClock(bad)).toBeNull();
  });
});
