import type { Conversation, OutputEvent, Preset, Terminal } from './lib/types';
import { createOutputHub } from './output';
import { createSendQueue } from './sendQueue';
import { createWsClient, type SocketLike } from './wsClient';

interface GoBridge {
  List(): Promise<Terminal[]>;
  Conversations(): Promise<Conversation[]>;
  Presets(): Promise<Preset[]>;
  NewTerminal(preset: string): Promise<Terminal>;
  Resume(sessionId: string): Promise<Terminal>;
  Write(id: string, data: string): Promise<void>;
  Paste(id: string, text: string): Promise<void>;
  Resize(id: string, cols: number, rows: number): Promise<void>;
  Snapshot(id: string): Promise<string>;
  Rename(id: string, name: string): Promise<void>;
  Close(id: string): Promise<void>;
  Forget(sessionId: string): Promise<void>;
  Sound(): Promise<boolean>;
  SetSound(on: boolean): Promise<void>;
  SetFocused(focused: boolean): Promise<void>;
  Hide(): Promise<void>;
  ToggleMaximise(): Promise<void>;
  Quit(): Promise<void>;
  Problems(): Promise<string[]>;
}

interface WailsRuntime {
  EventsOn(name: string, cb: (...data: any[]) => void): () => void;
  WindowMinimise(): void;
  WindowToggleMaximise(): void;
}

declare global {
  interface Window {
    go?: { main?: { Bridge?: GoBridge } };
    runtime: WailsRuntime;
  }
}

const inWindow = typeof window !== 'undefined' && !!window.go?.main?.Bridge;

export const isWindow = () => inWindow;

function browserSocket(): SocketLike {
  const scheme = location.protocol === 'https:' ? 'wss' : 'ws';
  const ws = new WebSocket(`${scheme}://${location.host}/ws`);
  const s: SocketLike = {
    send: (data) => ws.send(data),
    close: () => ws.close(),
    onopen: null,
    onclose: null,
    onmessage: null,
    onerror: null,
  };
  ws.onopen = () => s.onopen?.();
  ws.onclose = () => s.onclose?.();
  ws.onerror = () => s.onerror?.();
  ws.onmessage = (ev) => s.onmessage?.({ data: String(ev.data) });
  return s;
}

function remoteBridge(): { bridge: GoBridge; on: (name: string, cb: (...data: any[]) => void) => () => void } {
  const client = createWsClient({ connect: browserSocket });
  const call = <T>(method: string, ...args: unknown[]) => client.call<T>(method, ...args);
  const bridge: GoBridge = {
    List: () => call('List'),
    Conversations: () => call('Conversations'),
    Presets: () => call('Presets'),
    NewTerminal: (preset) => call('NewTerminal', preset),
    Resume: (sessionId) => call('Resume', sessionId),
    Write: (id, data) => call('Write', id, data),
    Paste: (id, text) => call('Paste', id, text),
    Resize: (id, cols, rows) => call('Resize', id, cols, rows),
    Snapshot: (id) => call('Snapshot', id),
    Rename: (id, name) => call('Rename', id, name),
    Close: (id) => call('Close', id),
    Forget: (sessionId) => call('Forget', sessionId),
    Sound: () => call('Sound'),
    SetSound: (on) => call('SetSound', on),
    SetFocused: (focused) => call('SetFocused', focused),
    Hide: async () => {},
    ToggleMaximise: async () => {},
    Quit: () => call('Quit'),
    Problems: () => call('Problems'),
  };
  return { bridge, on: (name, cb) => client.on(name, cb) };
}

const remote = inWindow ? null : remoteBridge();

export const api = (): GoBridge => remote?.bridge ?? window.go!.main!.Bridge!;
export const runtime = (): WailsRuntime => window.runtime;
export const on = (name: string, cb: (...data: any[]) => void) => (remote ? remote.on(name, cb) : window.runtime.EventsOn(name, cb));
export const output = createOutputHub((name, cb) => { on(name, cb as (ev: OutputEvent) => void); });

const queues = new Map<string, ReturnType<typeof createSendQueue>>();
export const input = (id: string) => {
  let queue = queues.get(id);
  if (!queue) {
    queue = createSendQueue((op) => (op.kind === 'write' ? api().Write(id, op.data) : api().Paste(id, op.data)));
    queues.set(id, queue);
  }
  return queue;
};
