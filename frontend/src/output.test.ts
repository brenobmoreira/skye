import { expect, it } from 'vitest';
import { createOutputHub } from './output';
import type { OutputEvent } from './lib/types';

function setup() {
  let emit: (ev: OutputEvent) => void = () => {};
  const hub = createOutputHub((_name, cb) => { emit = cb; });
  const send = (id: string, text: string) => emit({ id, data: btoa(text) });
  const collect = (id: string) => {
    const got: string[] = [];
    const off = hub.subscribe(id, (b) => got.push(new TextDecoder().decode(b)));
    return { got, off };
  };
  return { hub, send, emit: (ev: OutputEvent) => emit(ev), collect };
}

it('drops output for ids that are neither prepared nor subscribed', () => {
  const { send, collect } = setup();
  send('t1', 'early');
  const { got } = collect('t1');
  send('t1', 'live');
  expect(got).toEqual(['live']);
});

it('flushes only what arrived after prepare, then streams until unsubscribed', () => {
  const { hub, send, collect } = setup();
  send('t1', 'before');
  hub.prepare('t1');
  send('t1', 'during');
  send('t2', 'other');
  const { got, off } = collect('t1');
  send('t1', 'after');
  off();
  send('t1', 'late');
  expect(got).toEqual(['during', 'after']);
});

it('prepare discards data buffered by an earlier prepare', () => {
  const { hub, send, collect } = setup();
  hub.prepare('t1');
  send('t1', 'stale');
  hub.prepare('t1');
  send('t1', 'fresh');
  expect(collect('t1').got).toEqual(['fresh']);
});

it('ignores malformed events', () => {
  const { hub, emit, send, collect } = setup();
  hub.prepare('t1');
  expect(() => emit({ id: 't1', data: '%%%not base64' })).not.toThrow();
  send('t1', 'ok');
  expect(collect('t1').got).toEqual(['ok']);
});
