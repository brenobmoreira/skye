export type Mode = 'linux-window' | 'windows-app' | 'browser';

interface Bound {
  go?: { main?: { Bridge?: unknown; Shell?: unknown } };
}

export function detectMode(win: Bound): Mode {
  if (win.go?.main?.Bridge) return 'linux-window';
  if (win.go?.main?.Shell) return 'windows-app';
  return 'browser';
}
