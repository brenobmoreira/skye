import type { Terminal } from './types';

// After this long, something waiting for the user shows its time in amber.
export const OVERDUE_MIN = 10;

const asks = (t: Terminal) => t.state === 'waiting' || (t.state === 'idle' && (!!t.question || !!t.options?.length));

// Whether the terminal has been waiting for the user longer than OVERDUE_MIN.
export function overdue(t: Terminal, now: number): boolean {
  if (!t.since || !asks(t)) return false;
  const start = Date.parse(t.since);
  return !Number.isNaN(start) && now - start > OVERDUE_MIN * 60_000;
}

// The buttons that answer a finished turn from the list: sim to a question, one per choice.
export function answers(t: Terminal): { label: string; text: string }[] {
  if (t.state !== 'idle' || t.answered) return [];
  if (t.options?.length) return t.options.map((o) => ({ label: `${o.n} ${o.label}`, text: String(o.n) }));
  return t.question ? [{ label: 'sim', text: 'sim' }] : [];
}
