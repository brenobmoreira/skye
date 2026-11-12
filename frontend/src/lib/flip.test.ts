import { describe, expect, it } from 'vitest';
import { dropSpot, flipDeltas, translateYOf } from './flip';

describe('flipDeltas', () => {
  it('says how far each item moved, only for the ones that moved and were there before', () => {
    const before = new Map([['a', 0], ['b', 40], ['c', 80]]);
    const after = new Map([['b', 0], ['a', 40], ['c', 80], ['d', 120]]);
    expect(flipDeltas(before, after)).toEqual(new Map([['b', 40], ['a', -40]]));
  });
});

describe('dropSpot', () => {
  // The others, without the one being dragged: a short one, a tall one, a short one.
  const rows = [{ id: 'a', top: 0, height: 20 }, { id: 'b', top: 20, height: 60 }, { id: 'c', top: 80, height: 20 }];

  it('goes before the first note whose middle is below the pointer, or after the last', () => {
    expect(dropSpot(rows, 5)).toEqual({ id: 'a', side: 'before' });
    expect(dropSpot(rows, 15)).toEqual({ id: 'b', side: 'before' });
    expect(dropSpot(rows, 49)).toEqual({ id: 'b', side: 'before' });
    expect(dropSpot(rows, 51)).toEqual({ id: 'c', side: 'before' });
    expect(dropSpot(rows, 95)).toEqual({ id: 'c', side: 'after' });
    expect(dropSpot(rows, 500)).toEqual({ id: 'c', side: 'after' });
    expect(dropSpot([], 10)).toBeNull();
  });

  it('does not swing back and forth when the note dragged and the one passed differ in height', () => {
    // A tall note (60) dragged past a short one (20): once past, the pointer sits on the short note
    // moved up, and the spot must stay after it.
    const before = [{ id: 's', top: 60, height: 20 }]; // the tall one above it, at 0..60
    const spot = dropSpot(before, 75);
    expect(spot).toEqual({ id: 's', side: 'after' });
    const after = [{ id: 's', top: 0, height: 20 }]; // now the short one first, the tall one at 20..80
    expect(dropSpot(after, 75)).toEqual({ id: 's', side: 'after' });
  });
});

describe('translateYOf', () => {
  it('reads the vertical offset of a computed transform', () => {
    expect(translateYOf('none')).toBe(0);
    expect(translateYOf('')).toBe(0);
    expect(translateYOf('matrix(1, 0, 0, 1, 0, -12.5)')).toBe(-12.5);
    expect(translateYOf('matrix3d(1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 30, 0, 1)')).toBe(30);
  });
});
