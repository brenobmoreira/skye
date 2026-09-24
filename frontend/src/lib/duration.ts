import type { State } from './types';

// How long a terminal has been in its state, for the list; empty when there is nothing to say.
export function elapsed(since: string | undefined, state: State, now: number): string {
  if (!since || state === 'shell') return '';
  const start = Date.parse(since);
  if (Number.isNaN(start)) return '';
  const min = Math.floor(Math.max(0, now - start) / 60_000);
  if (min < 1) return 'agora';
  if (min < 60) return `${min} min`;
  const h = Math.floor(min / 60);
  const rest = min % 60;
  return rest ? `${h} h ${rest}` : `${h} h`;
}
