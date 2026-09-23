import { expect, it } from 'vitest';
import { createSendQueue, type SendOp } from './sendQueue';

function fakeSender() {
  const calls: SendOp[] = [];
  const resolvers: Array<() => void> = [];
  let inFlight = 0;
  let maxInFlight = 0;
  const send = (op: SendOp) => {
    calls.push(op);
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
  expect(f.calls).toEqual([{ kind: 'write', data: 'a' }]);
  await f.settle();
  expect(f.calls).toEqual([{ kind: 'write', data: 'a' }, { kind: 'write', data: 'bc' }]);
  await f.settle();
  expect(f.calls).toHaveLength(2);
  expect(f.maxInFlight()).toBe(1);
});

it('keeps a paste after the writes queued before it', async () => {
  const f = fakeSender();
  const q = createSendQueue(f.send);
  q.write('x');
  q.write('y');
  q.paste('hello');
  q.write('z');
  await f.settle();
  await f.settle();
  await f.settle();
  await f.settle();
  expect(f.calls).toEqual([
    { kind: 'write', data: 'x' },
    { kind: 'write', data: 'y' },
    { kind: 'paste', data: 'hello' },
    { kind: 'write', data: 'z' },
  ]);
  expect(f.maxInFlight()).toBe(1);
});

it('keeps going after a failed call', async () => {
  const calls: SendOp[] = [];
  let fail = true;
  const q = createSendQueue((op) => {
    calls.push(op);
    if (fail) { fail = false; return Promise.reject(new Error('boom')); }
    return Promise.resolve();
  });
  q.write('a');
  q.write('b');
  await new Promise((r) => setTimeout(r, 0));
  expect(calls).toEqual([{ kind: 'write', data: 'a' }, { kind: 'write', data: 'b' }]);
});
