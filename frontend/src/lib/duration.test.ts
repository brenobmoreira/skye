import { describe, expect, it } from 'vitest';
import { elapsed } from './duration';

const now = Date.parse('2026-09-24T12:00:00Z');
const ago = (min: number) => new Date(now - min * 60_000).toISOString();

describe('elapsed', () => {
  it('says agora under a minute', () => {
    expect(elapsed(ago(0.5), 'running', now)).toBe('agora');
  });
  it('counts minutes, then hours and minutes', () => {
    expect(elapsed(ago(12), 'waiting', now)).toBe('12 min');
    expect(elapsed(ago(60), 'idle', now)).toBe('1 h');
    expect(elapsed(ago(80), 'idle', now)).toBe('1 h 20');
  });
  it('shows nothing for a plain terminal or without a start', () => {
    expect(elapsed(ago(5), 'shell', now)).toBe('');
    expect(elapsed(undefined, 'running', now)).toBe('');
    expect(elapsed('lixo', 'running', now)).toBe('');
  });
  it('does not go negative when the clocks disagree', () => {
    expect(elapsed(ago(-2), 'running', now)).toBe('agora');
  });
});
