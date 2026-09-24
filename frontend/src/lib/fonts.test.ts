import { describe, expect, it } from 'vitest';
import { DEFAULT_FONT, FONTS, parseFont } from './fonts';

describe('parseFont', () => {
  it('returns the stored font', () => {
    expect(parseFont('cascadia-mono').label).toBe('Cascadia Mono (Windows Terminal)');
  });
  it('falls back to the default for missing or unknown ids', () => {
    expect(parseFont(null)).toBe(DEFAULT_FONT);
    expect(parseFont('comic-sans')).toBe(DEFAULT_FONT);
  });
  it('keeps monospace as the last fallback', () => {
    for (const f of FONTS) expect(f.family.endsWith(', monospace')).toBe(true);
  });
});
