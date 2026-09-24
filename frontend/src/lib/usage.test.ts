import { describe, expect, it } from 'vitest';
import { usagePanel } from './usage';

const at = (iso: string) => Date.parse(iso) / 1000;

describe('usagePanel', () => {
  it('lists both windows rounded, 5h first', () => {
    const p = usagePanel({ fiveHour: { usedPct: 29.4, resetsAt: at('2026-09-24T16:00:00Z') }, sevenDay: { usedPct: 4, resetsAt: at('2026-09-28T09:00:00Z') }, updatedAt: '2026-09-24T12:00:00Z' }, Date.parse('2026-09-24T12:05:00Z'));
    expect(p?.rows.map((r) => [r.label, r.pct])).toEqual([['5h', 29], ['7d', 4]]);
    expect(p?.rows[0].resets).not.toBe('');
    expect(p?.stale).toBe(false);
  });
  it('lists only the windows it knows and keeps the bar within 0..100', () => {
    const p = usagePanel({ fiveHour: null, sevenDay: { usedPct: 130, resetsAt: 0 }, updatedAt: '2026-09-24T12:00:00Z' }, Date.parse('2026-09-24T12:05:00Z'));
    expect(p?.rows).toEqual([{ label: '7d', pct: 100, resets: '' }]);
  });
  it('marks the numbers stale once the 5h window has reset', () => {
    const p = usagePanel({ fiveHour: { usedPct: 90, resetsAt: at('2026-09-24T12:00:00Z') }, sevenDay: null, updatedAt: '2026-09-24T09:00:00Z' }, Date.parse('2026-09-24T12:01:00Z'));
    expect(p?.stale).toBe(true);
  });
  it('shows nothing before the first status line', () => {
    expect(usagePanel({ fiveHour: null, sevenDay: null, updatedAt: '0001-01-01T00:00:00Z' }, Date.now())).toBeNull();
    expect(usagePanel(null, Date.now())).toBeNull();
  });
});
