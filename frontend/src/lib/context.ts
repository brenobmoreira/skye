import type { SessionContext } from './types';

const FULL = 80;

// The bar under a terminal item: how much of the context window the session has used.
export function contextBar(c: SessionContext | undefined): { pct: number; full: boolean } | null {
  if (c?.contextPct == null) return null;
  const pct = Math.min(100, Math.max(0, Math.round(c.contextPct)));
  return { pct, full: pct >= FULL };
}

export function contextText(c: SessionContext | undefined): string {
  if (!c) return '';
  const parts: string[] = [];
  if (c.contextPct != null) parts.push(`contexto ${Math.round(c.contextPct)}%`);
  if (c.model) parts.push(c.model);
  if (c.costUsd != null) parts.push(`US$ ${c.costUsd.toFixed(2).replace('.', ',')}`);
  return parts.join(' · ');
}
