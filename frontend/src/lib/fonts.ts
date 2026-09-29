export type TerminalFont = { id: string; label: string; family: string; custom?: boolean };

export const FONTS: TerminalFont[] = [
  { id: 'jetbrains-mono', label: 'JetBrains Mono', family: '"JetBrains Mono", monospace' },
  { id: 'cascadia-mono', label: 'Cascadia Mono (Windows Terminal)', family: '"Cascadia Mono", monospace' },
];

export const DEFAULT_FONT = FONTS[0];

// xterm's default of 1.0 packs the lines together; 1.2 is close to what Windows Terminal draws.
export const LINE_HEIGHT = 1.2;

// A font the user added by name: it has to be installed where the page runs, so monospace stays
// as the fallback when it is not.
export function customFont(name: string): TerminalFont {
  const clean = name.trim().replace(/["\\]/g, '');
  return { id: `custom:${clean}`, label: clean, family: `"${clean}", monospace`, custom: true };
}

export function parseCustomFonts(stored: string | null): TerminalFont[] {
  try {
    const names: unknown = JSON.parse(stored ?? '[]');
    if (!Array.isArray(names)) return [];
    const fonts = names.filter((n): n is string => typeof n === 'string' && n.trim() !== '').map(customFont);
    return fonts.filter((f, i) => fonts.findIndex((g) => g.id === f.id) === i && !FONTS.some((b) => b.label === f.label));
  } catch {
    return [];
  }
}

export function parseFont(stored: string | null, custom: TerminalFont[] = []): TerminalFont {
  return [...FONTS, ...custom].find((f) => f.id === stored) ?? DEFAULT_FONT;
}
