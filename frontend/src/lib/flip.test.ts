import { describe, expect, it } from 'vitest';
import { flipDeltas } from './flip';

describe('flipDeltas', () => {
  it('says how far each item moved, only for the ones that moved and were there before', () => {
    const before = new Map([['a', 0], ['b', 40], ['c', 80]]);
    const after = new Map([['b', 0], ['a', 40], ['c', 80], ['d', 120]]);
    expect(flipDeltas(before, after)).toEqual(new Map([['b', 40], ['a', -40]]));
  });
});
