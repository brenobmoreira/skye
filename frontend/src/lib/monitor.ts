import type { Machine, MemoryPoint, StrayKind } from './types';

export function size(mb: number): string {
  if (mb < 1024) return `${Math.round(mb)} MB`;
  const gb = Math.round((mb / 1024) * 10) / 10;
  return `${String(gb).replace('.', ',')} GB`;
}

export type MemoryLevel = 'ok' | 'baixa' | 'crítica';

// Available memory under 20 % of the total is low, under 10 % critical.
export function memoryLevel(m: Pick<Machine, 'memTotalMb' | 'memAvailMb'>): MemoryLevel {
  if (m.memTotalMb <= 0) return 'ok';
  const share = m.memAvailMb / m.memTotalMb;
  if (share < 0.1) return 'crítica';
  if (share < 0.2) return 'baixa';
  return 'ok';
}

const strayLabels: Record<StrayKind, string> = {
  solto: 'processo solto',
  claude: 'claude fora da skye',
  skye: 'outra skye',
  janela: 'janela sem item',
};

export const strayLabel = (kind: StrayKind) => strayLabels[kind] ?? kind;

// SVG path of the available memory over time, evenly spaced, 0 MB at the bottom and the total
// at the top.
export function sparkline(points: MemoryPoint[], totalMb: number, width: number, height: number): string {
  if (points.length < 2 || totalMb <= 0) return '';
  const step = width / (points.length - 1);
  return points
    .map((p, i) => {
      const y = height - (Math.min(p.availMb, totalMb) / totalMb) * height;
      return `${i === 0 ? 'M' : 'L'}${round(i * step)},${round(y)}`;
    })
    .join(' ');
}

const round = (n: number) => Math.round(n * 100) / 100;

// Index of the sample under a pointer at `fraction` of the chart width.
export function nearest(count: number, fraction: number): number {
  if (count <= 0) return -1;
  return Math.min(count - 1, Math.max(0, Math.round(fraction * (count - 1))));
}
