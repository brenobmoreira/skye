import { describe, expect, it } from 'vitest';
import { shouldReload } from './version';

describe('shouldReload', () => {
  it('reloads only when a known version changed', () => {
    expect(shouldReload('', 'abc')).toBe(false);
    expect(shouldReload('abc', 'abc')).toBe(false);
    expect(shouldReload('abc', 'def')).toBe(true);
    expect(shouldReload('abc', '')).toBe(false);
  });
});
