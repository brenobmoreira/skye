import { expect, it } from 'vitest';
import { createOutputHub } from './output';
import type { OutputEvent } from './lib/types';

it('buffers output until a terminal subscribes, then streams', () => {
  let emit: (ev: OutputEvent) => void = () => {};
  const hub = createOutputHub((_name, cb) => { emit = cb; });
  emit({ id: 't1', data: btoa('ab') });
  const got: string[] = [];
  const off = hub.subscribe('t1', (b) => got.push(new TextDecoder().decode(b)));
  emit({ id: 't1', data: btoa('cd') });
  emit({ id: 't2', data: btoa('other') });
  off();
  emit({ id: 't1', data: btoa('late') });
  expect(got).toEqual(['ab', 'cd']);
});
