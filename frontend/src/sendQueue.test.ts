import { expect, it } from 'vitest';
import { createSendQueue } from './sendQueue';

function fakeSender() {
  const calls: string[] = [];
  const resolvers: Array<() => void> = [];
  let inFlight = 0;
  let maxInFlight = 0;
  const send = (data: string) => {
    calls.push(data);
    inFlight++;
    maxInFlight = Math.max(maxInFlight, inFlight);
    return new Promise<void>((resolve) => {
      resolvers.push(() => { inFlight--; resolve(); });
    });
  };
  const settle = async () => {
    resolvers.shift()?.();
    await new Promise((r) => setTimeout(r, 0));
  };
  return { send, calls, settle, maxInFlight: () => maxInFlight };
}

it('sends one call at a time and merges writes queued meanwhile', async () => {
  const f = fakeSender();
  const q = createSendQueue(f.send);
  q.write('a');
  q.write('b');
  q.write('c');
  expect(f.calls).toEqual(['a']);
  await f.settle();
  expect(f.calls).toEqual(['a', 'bc']);
  await f.settle();
  expect(f.calls).toHaveLength(2);
  expect(f.maxInFlight()).toBe(1);
});

it('keeps going after a failed call', async () => {
  const calls: string[] = [];
  let fail = true;
  const q = createSendQueue((data) => {
    calls.push(data);
    if (fail) { fail = false; return Promise.reject(new Error('boom')); }
    return Promise.resolve();
  });
  q.write('a');
  q.write('b');
  await new Promise((r) => setTimeout(r, 0));
  expect(calls).toEqual(['a', 'b']);
});
