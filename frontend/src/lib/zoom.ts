export const DEFAULT_FONT_SIZE = 16;
export const MIN_FONT_SIZE = 10;
export const MAX_FONT_SIZE = 32;

export type ZoomAction = 'in' | 'out' | 'reset';

type Keyish = Pick<KeyboardEvent, 'key' | 'ctrlKey' | 'altKey' | 'metaKey'>;

export function zoomKey(e: Keyish): ZoomAction | null {
  if (!e.ctrlKey || e.altKey || e.metaKey) return null;
  if (e.key === '=' || e.key === '+') return 'in';
  if (e.key === '-' || e.key === '_') return 'out';
  if (e.key === '0') return 'reset';
  return null;
}

export function applyZoom(size: number, action: ZoomAction): number {
  if (action === 'reset') return DEFAULT_FONT_SIZE;
  const next = size + (action === 'in' ? 1 : -1);
  return Math.min(MAX_FONT_SIZE, Math.max(MIN_FONT_SIZE, next));
}

export function parseFontSize(stored: string | null): number {
  const n = Number(stored);
  if (!Number.isInteger(n) || n < MIN_FONT_SIZE || n > MAX_FONT_SIZE) return DEFAULT_FONT_SIZE;
  return n;
}
