import { expect, it } from 'vitest';
import { fromBase64 } from './bytes';

it('decodes base64 to raw bytes, keeping split UTF-8 intact', () => {
  const bytes = fromBase64(btoa(String.fromCharCode(0x1b, 0x5b, 0xc3)));
  expect(Array.from(bytes)).toEqual([0x1b, 0x5b, 0xc3]);
});
