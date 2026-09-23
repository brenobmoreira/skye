import { expect, it } from 'vitest';
import { createWsClient, type SocketLike } from './wsClient';

class FakeSocket implements SocketLike {
  sent: any[] = [];
  closed = false;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  send(data: string) { this.sent.push(JSON.parse(data)); }
  close() { this.closed = true; this.onclose?.(); }
  open() { this.onopen?.(); }
  receive(frame: unknown) { this.onmessage?.({ data: JSON.stringify(frame) }); }
  drop() { this.onclose?.(); }
}

function setup() {
  const sockets: FakeSocket[] = [];
  const timers: Array<{ fn: () => void; ms: number }> = [];
  const client = createWsClient({
    connect: () => { const s = new FakeSocket(); sockets.push(s); return s; },
    schedule: (fn, ms) => { timers.push({ fn, ms }); },
  });
  const flush = () => new Promise((r) => setTimeout(r, 0));
  return { client, sockets, timers, flush, last: () => sockets[sockets.length - 1] };
}

it('queues calls until the socket opens and matches replies by id', async () => {
  const { client, last } = setup();
  const a = client.call('List');
  const b = client.call('Write', 't1', 'ls');
  expect(last().sent).toEqual([]);
  last().open();
  const [first, second] = last().sent;
  expect(first).toMatchObject({ method: 'List', args: [] });
  expect(second).toMatchObject({ method: 'Write', args: ['t1', 'ls'] });
  expect(first.id).not.toBe(second.id);
  last().receive({ id: second.id, result: null });
  last().receive({ id: first.id, result: [{ id: 't1' }] });
  await expect(a).resolves.toEqual([{ id: 't1' }]);
  await expect(b).resolves.toBeNull();
});

it('rejects a call whose reply carries an error', async () => {
  const { client, last } = setup();
  last().open();
  const p = client.call('Nope');
  last().receive({ id: last().sent[0].id, error: 'método desconhecido' });
  await expect(p).rejects.toThrow('método desconhecido');
});

it('dispatches events to every listener of that name until unsubscribed', () => {
  const { client, last } = setup();
  last().open();
  const got: unknown[] = [];
  const off = client.on('terminals', (list) => got.push(['a', list]));
  client.on('terminals', (list) => got.push(['b', list]));
  client.on('bark', () => got.push('bark'));
  last().receive({ event: 'terminals', data: ['t1'] });
  off();
  last().receive({ event: 'terminals', data: ['t2'] });
  expect(got).toEqual([['a', ['t1']], ['b', ['t1']], ['b', ['t2']]]);
});

it('rejects pending and queued calls when the socket closes', async () => {
  const { client, last } = setup();
  last().open();
  const sent = client.call('Snapshot', 't1');
  last().drop();
  const queued = client.call('List');
  await expect(sent).rejects.toThrow();
  last().drop();
  await expect(queued).rejects.toThrow();
});

it('reconnects with growing backoff and announces the reconnection', () => {
  const { client, sockets, timers, last } = setup();
  const events: string[] = [];
  client.on('reconnected', () => events.push('reconnected'));
  last().open();
  last().drop();
  expect(timers).toHaveLength(1);
  timers[0].fn();
  expect(sockets).toHaveLength(2);
  last().drop();
  expect(timers).toHaveLength(2);
  expect(timers[1].ms).toBeGreaterThan(timers[0].ms);
  timers[1].fn();
  expect(sockets).toHaveLength(3);
  expect(events).toEqual([]);
  last().open();
  expect(events).toEqual(['reconnected']);
  last().drop();
  expect(timers[2].ms).toBe(timers[0].ms);
});

it('ignores replies for unknown ids and malformed frames', () => {
  const { client, last } = setup();
  last().open();
  const p = client.call('List');
  last().onmessage?.({ data: 'not json' });
  last().receive({ id: 999, result: 1 });
  last().receive({ id: last().sent[0].id, result: [] });
  return expect(p).resolves.toEqual([]);
});
