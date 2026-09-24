export type TerminalFont = { id: string; label: string; family: string };

export const FONTS: TerminalFont[] = [
  { id: 'jetbrains-mono', label: 'JetBrains Mono', family: '"JetBrains Mono", monospace' },
  { id: 'cascadia-mono', label: 'Cascadia Mono (Windows Terminal)', family: '"Cascadia Mono", monospace' },
];

export const DEFAULT_FONT = FONTS[0];

export function parseFont(stored: string | null): TerminalFont {
  return FONTS.find((f) => f.id === stored) ?? DEFAULT_FONT;
}
