import { describe, expect, it } from 'vitest';
import { capitalize } from './text';

describe('capitalize', () => {
  it('raises only the first letter', () => {
    expect(capitalize('esperando você')).toBe('Esperando você');
    expect(capitalize('ébano')).toBe('Ébano');
    expect(capitalize('')).toBe('');
    expect(capitalize('3 min')).toBe('3 min');
  });
});
