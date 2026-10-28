import { describe, expect, it } from 'vitest';
import { asksBeforeClose } from './close';

describe('asksBeforeClose', () => {
  it('asks while claude is working or waiting for an answer', () => {
    expect(asksBeforeClose('running')).toBe(true);
    expect(asksBeforeClose('waiting')).toBe(true);
  });
  it('closes a finished claude or a plain shell right away', () => {
    expect(asksBeforeClose('idle')).toBe(false);
    expect(asksBeforeClose('shell')).toBe(false);
  });
});
