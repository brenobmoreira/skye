import { describe, expect, it } from 'vitest';
import { memoryLevel, nearest, size, sparkline, strayLabel } from './monitor';

describe('size', () => {
  it('shows MB under a gigabyte and GB with one decimal above', () => {
    expect(size(812)).toBe('812 MB');
    expect(size(9041)).toBe('8,8 GB');
    expect(size(16384)).toBe('16 GB');
  });
});

describe('memoryLevel', () => {
  it('flags low and critical available memory', () => {
    expect(memoryLevel({ memTotalMb: 10000, memAvailMb: 5000 })).toBe('ok');
    expect(memoryLevel({ memTotalMb: 10000, memAvailMb: 1900 })).toBe('baixa');
    expect(memoryLevel({ memTotalMb: 10000, memAvailMb: 900 })).toBe('crítica');
    expect(memoryLevel({ memTotalMb: 0, memAvailMb: 0 })).toBe('ok');
  });
});

describe('strayLabel', () => {
  it('names each kind of leftover', () => {
    expect(strayLabel('solto')).toBe('processo solto');
    expect(strayLabel('claude')).toBe('claude fora da skye');
    expect(strayLabel('skye')).toBe('outra skye');
    expect(strayLabel('janela')).toBe('janela sem item');
  });
});

describe('sparkline', () => {
  const pts = [
    { t: '2026-09-24T10:00:00Z', availMb: 8000 },
    { t: '2026-09-24T10:01:00Z', availMb: 4000 },
    { t: '2026-09-24T10:02:00Z', availMb: 6000 },
  ];
  it('draws from the left edge to the right with 0 at the bottom and the total at the top', () => {
    expect(sparkline(pts, 8000, 200, 50)).toBe('M0,0 L100,25 L200,12.5');
  });
  it('needs two points', () => {
    expect(sparkline(pts.slice(0, 1), 8000, 200, 50)).toBe('');
  });
});

describe('nearest', () => {
  it('picks the sample under the pointer', () => {
    expect(nearest(10, 0.0)).toBe(0);
    expect(nearest(10, 0.52)).toBe(5);
    expect(nearest(10, 1.2)).toBe(9);
    expect(nearest(0, 0.5)).toBe(-1);
  });
});
