import type { Usage, UsageWindow } from './types';

const time = (epoch: number) => new Date(epoch * 1000).toLocaleString([], { weekday: 'short', hour: '2-digit', minute: '2-digit' });

// The badge in the title bar. The numbers only change while some claude session redraws its
// status line, so they go stale once the 5h window resets with nobody running.
export function usageBadge(u: Usage | null, now: number): { text: string; title: string; stale: boolean } | null {
  if (!u || (!u.fiveHour && !u.sevenDay)) return null;
  const part = (label: string, w: UsageWindow | null) => (w ? `${label} ${Math.round(w.usedPct)}%` : '');
  const reset = (label: string, w: UsageWindow | null) => (w?.resetsAt ? `${label} renova ${time(w.resetsAt)}` : '');
  const updated = new Date(u.updatedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  return {
    text: [part('5h', u.fiveHour), part('7d', u.sevenDay)].filter(Boolean).join(' · '),
    title: [reset('5h', u.fiveHour), reset('7d', u.sevenDay), `atualizado às ${updated}`].filter(Boolean).join('\n'),
    stale: !!u.fiveHour?.resetsAt && now > u.fiveHour.resetsAt * 1000,
  };
}
