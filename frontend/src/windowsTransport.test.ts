import { expect, it } from 'vitest';
import { createWsClient, type SocketLike } from './wsClient';
import { createWindowsTransport, type ConnectionState, type Endpoint } from './windowsTransport';

class FakeSocket implements SocketLike {
  sent: string[] = [];
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  constructor(public url: string) {}
  send(data: string) { this.sent.push(data); }
  close() { this.onclose?.(); }
  open() { this.onopen?.(); }
  drop() { this.onclose?.(); }
  receive(frame: unknown) { this.onmessage?.({ data: JSON.stringify(frame) }); }
}

const flush = () => new Promise((r) => setTimeout(r, 0));

function setup(results: Array<Endpoint | Error>) {
  const sockets: FakeSocket[] = [];
  const timers: Array<() => void> = [];
  const states: ConnectionState[] = [];
  let connects = 0;
  const transport = createWindowsTransport({
    connect: () => {
      const r = results[Math.min(connects, results.length - 1)];
      connects++;
      return r instanceof Error ? Promise.reject(r.message) : Promise.resolve(r);
    },
    open: (url) => { const s = new FakeSocket(url); sockets.push(s); return s; },
    onState: (s) => states.push(s),
  });
  const client = createWsClient({ connect: transport.dial, schedule: (fn) => { timers.push(fn); } });
  return {
    client,
    transport,
    sockets,
    states,
    connects: () => connects,
    last: () => sockets[sockets.length - 1],
    fire: () => timers.shift()!(),
  };
}

const ep: Endpoint = { url: 'ws://127.0.0.1:7810/ws', token: 'ab'.repeat(32) };

it('opens the socket at the url and token that Connect returned', async () => {
  const t = setup([ep]);
  expect(t.states).toEqual([{ kind: 'connecting' }]);
  expect(t.sockets).toHaveLength(0);
  await flush();
  expect(t.sockets).toHaveLength(1);
  expect(t.last().url).toBe(`ws://127.0.0.1:7810/ws?token=${'ab'.repeat(32)}`);
  t.last().open();
  expect(t.states.at(-1)).toEqual({ kind: 'ready' });
});

it('sends calls queued while Connect was running once the socket opens', async () => {
  const t = setup([ep]);
  const reply = t.client.call('List');
  await flush();
  t.last().open();
  const frame = JSON.parse(t.last().sent[0]);
  expect(frame).toMatchObject({ method: 'List' });
  t.last().receive({ id: frame.id, result: [] });
  await expect(reply).resolves.toEqual([]);
});

it('reuses the endpoint on reconnects and calls Connect again after three failures in a row', async () => {
  const other: Endpoint = { url: 'ws://127.0.0.1:7810/ws', token: 'cd'.repeat(32) };
  const t = setup([ep, other]);
  await flush();
  t.last().open();
  t.last().drop();
  for (let i = 0; i < 3; i++) {
    t.fire();
    expect(t.connects()).toBe(1);
    expect(t.last().url).toContain(ep.token);
    t.last().drop();
  }
  t.fire();
  expect(t.connects()).toBe(2);
  expect(t.states.at(-1)).toEqual({ kind: 'connecting' });
  await flush();
  expect(t.last().url).toContain(other.token);
  t.last().open();
  expect(t.states.at(-1)).toEqual({ kind: 'ready' });
});

it('an open in between resets the failure count', async () => {
  const t = setup([ep]);
  await flush();
  t.last().open();
  for (let round = 0; round < 3; round++) {
    t.last().drop();
    t.fire();
    t.last().drop();
    t.fire();
    t.last().open();
  }
  expect(t.connects()).toBe(1);
});

it('shows the Connect error and tries again only when asked', async () => {
  const t = setup([new Error('wsl.exe -- bash -lc "skye web --no-open" falhou\nsaída'), ep]);
  const reply = t.client.call('List');
  await flush();
  expect(t.states.at(-1)).toEqual({ kind: 'error', message: 'wsl.exe -- bash -lc "skye web --no-open" falhou\nsaída' });
  expect(t.sockets).toHaveLength(0);
  expect(t.connects()).toBe(1);
  t.transport.retry();
  expect(t.states.at(-1)).toEqual({ kind: 'connecting' });
  await flush();
  expect(t.connects()).toBe(2);
  t.last().open();
  const frame = JSON.parse(t.last().sent[0]);
  t.last().receive({ id: frame.id, result: ['t1'] });
  await expect(reply).resolves.toEqual(['t1']);
});

it('retry does nothing when there is no error waiting', async () => {
  const t = setup([ep]);
  await flush();
  t.transport.retry();
  await flush();
  expect(t.connects()).toBe(1);
});
