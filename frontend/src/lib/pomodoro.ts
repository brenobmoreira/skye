import type { PomodoroState } from './types';

export function clock(ms: number): string {
  const s = Math.max(0, Math.ceil(ms / 1000));
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
}

export function phaseLabel(s: PomodoroState): string {
  switch (s.phase) {
    case 'focus':
      return `Foco · ${(s.count % s.settings.every) + 1} de ${s.settings.every}`;
    case 'break':
      return 'Pausa';
    case 'long':
      return 'Pausa longa';
    case 'idle':
      if (s.count === 0) return 'Pronto para focar';
      return s.count === 1 ? '1 foco hoje' : `${s.count} focos hoje`;
  }
}

const NAMES = { focus: 'Foco', break: 'Pausa', long: 'Pausa longa' } as const;

export function pillLabel(s: PomodoroState, remainingMs: number): string {
  if (s.phase === 'idle') return '';
  return `${NAMES[s.phase]} ${clock(remainingMs)}`;
}
