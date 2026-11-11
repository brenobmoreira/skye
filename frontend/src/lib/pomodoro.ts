import type { PomodoroPhase, PomodoroState } from './types';

export function clock(ms: number): string {
  const s = Math.max(0, Math.ceil(ms / 1000));
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
}

export function phaseLabel(s: PomodoroState): string {
  switch (s.phase) {
    case 'focus':
      return `Foco · ${s.stage + 1} de ${s.settings.every}`;
    case 'break':
      return 'Pausa';
    case 'long':
      return 'Pausa longa';
    case 'idle':
      if (s.pending) return s.pending === 'long' ? 'Hora da pausa longa' : 'Hora da pausa';
      if (s.count === 0) return 'Pronto para focar';
      return s.count === 1 ? '1 foco hoje' : `${s.count} focos hoje`;
  }
}

const NAMES = { focus: 'Foco', break: 'Pausa', long: 'Pausa longa' } as const;

export function pillLabel(s: PomodoroState, remainingMs: number): string {
  if (s.phase === 'idle') return '';
  return `${NAMES[s.phase]} ${clock(remainingMs)}`;
}

// pillPhase is the phase alone, for the pill when the time left is hidden.
export function pillPhase(s: PomodoroState): string {
  return s.phase === 'idle' ? '' : NAMES[s.phase];
}

export function endedLabel(phase: PomodoroPhase): string {
  return phase === 'focus' ? 'Foco acabou' : 'Pausa acabou';
}

// breakLabel names the button that starts the break after stage n of a cycle of `every`.
export function breakLabel(n: number, every: number): string {
  return n >= every ? 'Começar pausa longa' : 'Começar pausa';
}

const PHASE_NAMES = { idle: 'Foco', focus: 'Foco', break: 'Pausa', long: 'Pausa longa' } as const;

// collapsedLabel is the one line the timer island shows when collapsed: the phase, and whether
// it runs, ran out (until acknowledged) or has not started. Pausa takes the feminine ending.
export function collapsedLabel(s: PomodoroState, ended: PomodoroPhase | null): string {
  if (ended) return `${PHASE_NAMES[ended]} – ${ended === 'focus' ? 'Concluído' : 'Concluída'}`;
  if (s.phase === 'idle' && s.pending) return `${PHASE_NAMES[s.pending]} – Não iniciada`;
  if (s.phase === 'idle') return 'Foco – Não iniciado';
  return `${PHASE_NAMES[s.phase]} – Em andamento`;
}

// parseClock reads "18" (minutes) or "18:30" (minutes and seconds) into seconds; null when it is
// not a time from 1 s to 180 min.
export function parseClock(text: string): number | null {
  const m = /^(\d{1,3})(?::(\d{2}))?$/.exec(text.trim());
  if (!m) return null;
  const seconds = Number(m[2] ?? 0);
  if (seconds > 59) return null;
  const total = Number(m[1]) * 60 + seconds;
  return total >= 1 && total <= 180 * 60 ? total : null;
}
