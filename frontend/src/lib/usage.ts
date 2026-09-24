import type { Usage, UsageWindow } from './types';

export type UsageRow = { label: string; pct: number; resets: string };

// Today the reset shows only the hour, like the status line; further out it also names the day.
const resetLabel = (epoch: number, now: number) => {
  const d = new Date(epoch * 1000);
  const hour = d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  return d.toDateString() === new Date(now).toDateString() ? hour : `${d.toLocaleDateString([], { weekday: 'short' })} ${hour}`;
};

// The panel at the bottom of the sidebar. The numbers only change while some claude session
// redraws its status line, so they go stale once the 5h window resets with nobody running.
export function usagePanel(u: Usage | null, now: number): { rows: UsageRow[]; updated: string; stale: boolean } | null {
  if (!u || (!u.fiveHour && !u.sevenDay)) return null;
  const row = (label: string, w: UsageWindow | null): UsageRow[] =>
    w ? [{ label, pct: Math.min(100, Math.max(0, Math.round(w.usedPct))), resets: w.resetsAt ? resetLabel(w.resetsAt, now) : '' }] : [];
  return {
    rows: [...row('5h', u.fiveHour), ...row('7d', u.sevenDay)],
    updated: new Date(u.updatedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    stale: !!u.fiveHour?.resetsAt && now > u.fiveHour.resetsAt * 1000,
  };
}
