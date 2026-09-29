import { describe, expect, it } from 'vitest';
import { DEFAULT_FONT, FONTS, customFont, parseCustomFonts, parseFont } from './fonts';

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
  it('finds a font the user added', () => {
    const custom = [customFont('Fira Code')];
    expect(parseFont('custom:Fira Code', custom).label).toBe('Fira Code');
    expect(parseFont('custom:Fira Code')).toBe(DEFAULT_FONT);
  });
});

describe('customFont', () => {
  it('quotes the name and falls back to monospace', () => {
    expect(customFont('  Fira Code ').family).toBe('"Fira Code", monospace');
  });
  it('drops characters that would break the CSS family', () => {
    expect(customFont('Evil", serif').family).toBe('"Evil, serif", monospace');
  });
});

describe('parseCustomFonts', () => {
  it('reads the stored names', () => {
    expect(parseCustomFonts('["Fira Code","Consolas"]').map((f) => f.label)).toEqual(['Fira Code', 'Consolas']);
  });
  it('ignores garbage, blanks, duplicates and built-in fonts', () => {
    expect(parseCustomFonts(null)).toEqual([]);
    expect(parseCustomFonts('not json')).toEqual([]);
    expect(parseCustomFonts('{"a":1}')).toEqual([]);
    expect(parseCustomFonts('["Consolas", " ", 3, "Consolas", "JetBrains Mono"]').map((f) => f.label)).toEqual(['Consolas']);
  });
});
