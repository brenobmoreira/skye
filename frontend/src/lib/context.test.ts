import { describe, expect, it } from 'vitest';
import { contextBar, contextText } from './context';

describe('contextBar', () => {
  it('gives the width and warns from 80 %', () => {
    expect(contextBar({ contextPct: 42.4 })).toEqual({ pct: 42, full: false });
    expect(contextBar({ contextPct: 80 })).toEqual({ pct: 80, full: true });
  });
  it('keeps the width inside 0..100', () => {
    expect(contextBar({ contextPct: 130 })).toEqual({ pct: 100, full: true });
  });
  it('shows nothing without a context', () => {
    expect(contextBar(undefined)).toBeNull();
    expect(contextBar({ model: 'Opus 5.5' })).toBeNull();
  });
});

describe('contextText', () => {
  it('joins context, model and cost', () => {
    expect(contextText({ contextPct: 42.4, model: 'Opus 5.5', costUsd: 1.234 })).toBe('contexto 42% · Opus 5.5 · US$ 1,23');
  });
  it('leaves out what is missing', () => {
    expect(contextText({ model: 'Opus 5.5' })).toBe('Opus 5.5');
    expect(contextText(undefined)).toBe('');
  });
});
