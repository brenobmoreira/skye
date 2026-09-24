import { describe, expect, it } from 'vitest';
import { usageBadge } from './usage';

const at = (iso: string) => Date.parse(iso);

describe('usageBadge', () => {
  it('shows both windows rounded', () => {
    const b = usageBadge({ fiveHour: { usedPct: 42.4, resetsAt: at('2026-09-24T15:00:00Z') / 1000 }, sevenDay: { usedPct: 18, resetsAt: at('2026-09-28T09:00:00Z') / 1000 }, updatedAt: '2026-09-24T12:00:00Z' }, at('2026-09-24T12:05:00Z'));
    expect(b?.text).toBe('5h 42% · 7d 18%');
    expect(b?.stale).toBe(false);
  });
  it('shows only the windows it knows', () => {
    const b = usageBadge({ fiveHour: null, sevenDay: { usedPct: 3, resetsAt: 0 }, updatedAt: '2026-09-24T12:00:00Z' }, at('2026-09-24T12:05:00Z'));
    expect(b?.text).toBe('7d 3%');
  });
  it('marks the numbers stale once the 5h window has reset', () => {
    const b = usageBadge({ fiveHour: { usedPct: 90, resetsAt: at('2026-09-24T12:00:00Z') / 1000 }, sevenDay: null, updatedAt: '2026-09-24T09:00:00Z' }, at('2026-09-24T12:01:00Z'));
    expect(b?.stale).toBe(true);
  });
  it('shows nothing before the first status line', () => {
    expect(usageBadge({ fiveHour: null, sevenDay: null, updatedAt: '0001-01-01T00:00:00Z' }, Date.now())).toBeNull();
    expect(usageBadge(null, Date.now())).toBeNull();
  });
});
