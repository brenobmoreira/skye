import type { Conversation, MonitorReport, OutputEvent, Preset, Repo, Terminal, Usage } from './lib/types';
import { createOutputHub } from './output';
import { createSendQueue } from './sendQueue';
import { createWsClient, type SocketLike } from './wsClient';
import { detectMode } from './mode';
import { createWindowsTransport, type ConnectionState, type Endpoint } from './windowsTransport';

interface GoBridge {
  List(): Promise<Terminal[]>;
  Conversations(): Promise<Conversation[]>;
  Presets(): Promise<Preset[]>;
  NewTerminal(preset: string): Promise<Terminal>;
  Repos(): Promise<Repo[]>;
  NewWorktree(repo: string, branch: string): Promise<Terminal>;
  Resume(sessionId: string): Promise<Terminal>;
  Write(id: string, data: string): Promise<void>;
  Resize(id: string, cols: number, rows: number): Promise<void>;
  Snapshot(id: string): Promise<string>;
  Rename(id: string, name: string): Promise<void>;
  Reorder(ids: string[]): Promise<void>;
  Close(id: string): Promise<void>;
  Forget(sessionId: string): Promise<void>;
  Sound(): Promise<boolean>;
  SetSound(on: boolean): Promise<void>;
  SetFocused(focused: boolean): Promise<void>;
  Hide(): Promise<void>;
  ToggleMaximise(): Promise<void>;
  Quit(): Promise<void>;
  Problems(): Promise<string[]>;
  Usage(): Promise<Usage>;
  Monitor(): Promise<MonitorReport>;
  OpenURL(url: string): Promise<void>;
}

interface GoShell {
  Connect(): Promise<Endpoint>;
}

interface WailsRuntime {
  EventsOn(name: string, cb: (...data: any[]) => void): () => void;
  WindowMinimise(): void;
  WindowToggleMaximise(): void;
  WindowIsMaximised?(): Promise<boolean>;
  BrowserOpenURL(url: string): void;
  Quit(): void;
}

declare global {
  interface Window {
    go?: { main?: { Bridge?: GoBridge; Shell?: GoShell } };
    runtime: WailsRuntime;
  }
}

export const mode = typeof window === 'undefined' ? 'browser' : detectMode(window);

export const isWindow = () => mode !== 'browser';

function socketAt(url: string): SocketLike {
  const ws = new WebSocket(url);
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

function browserSocket(): SocketLike {
  const scheme = location.protocol === 'https:' ? 'wss' : 'ws';
  return socketAt(`${scheme}://${location.host}/ws`);
}

let connectionState: ConnectionState = mode === 'windows-app' ? { kind: 'connecting' } : { kind: 'ready' };
const connectionListeners = new Set<(state: ConnectionState) => void>();

const windowsTransport =
  mode === 'windows-app'
    ? createWindowsTransport({
        connect: () => window.go!.main!.Shell!.Connect(),
        open: socketAt,
        onState: (state) => {
          connectionState = state;
          connectionListeners.forEach((cb) => cb(state));
        },
      })
    : null;

export const connection = () => connectionState;
export const onConnection = (cb: (state: ConnectionState) => void) => {
  connectionListeners.add(cb);
  return () => { connectionListeners.delete(cb); };
};
export const retryConnection = () => windowsTransport?.retry();

function remoteBridge(connect: () => SocketLike): { bridge: GoBridge; on: (name: string, cb: (...data: any[]) => void) => () => void } {
  const client = createWsClient({ connect });
  const call = <T>(method: string, ...args: unknown[]) => client.call<T>(method, ...args);
  const bridge: GoBridge = {
    List: () => call('List'),
    Conversations: () => call('Conversations'),
    Presets: () => call('Presets'),
    NewTerminal: (preset) => call('NewTerminal', preset),
    Repos: () => call('Repos'),
    NewWorktree: (repo, branch) => call('NewWorktree', repo, branch),
    Resume: (sessionId) => call('Resume', sessionId),
    Write: (id, data) => call('Write', id, data),
    Resize: (id, cols, rows) => call('Resize', id, cols, rows),
    Snapshot: (id) => call('Snapshot', id),
    Rename: (id, name) => call('Rename', id, name),
    Reorder: (ids) => call('Reorder', ids),
    Close: (id) => call('Close', id),
    Forget: (sessionId) => call('Forget', sessionId),
    Sound: () => call('Sound'),
    SetSound: (on) => call('SetSound', on),
    SetFocused: (focused) => call('SetFocused', focused),
    Hide: async () => {},
    ToggleMaximise: async () => {},
    Quit: () => call('Quit'),
    Problems: () => call('Problems'),
    Usage: () => call('Usage'),
    Monitor: () => call('Monitor'),
    OpenURL: async () => {},
  };
  return { bridge, on: (name, cb) => client.on(name, cb) };
}

const remote = mode === 'linux-window' ? null : remoteBridge(windowsTransport ? windowsTransport.dial : browserSocket);

export const api = (): GoBridge => remote?.bridge ?? window.go!.main!.Bridge!;
export const runtime = (): WailsRuntime => window.runtime;
export const on = (name: string, cb: (...data: any[]) => void) => (remote ? remote.on(name, cb) : window.runtime.EventsOn(name, cb));
export const output = createOutputHub((name, cb) => { on(name, cb as (ev: OutputEvent) => void); });

const queues = new Map<string, ReturnType<typeof createSendQueue>>();
export const input = (id: string) => {
  let queue = queues.get(id);
  if (!queue) {
    queue = createSendQueue((data) => api().Write(id, data));
    queues.set(id, queue);
  }
  return queue;
};

// Opens a web link in the user's browser instead of inside the app window.
export function openExternal(url: string) {
  if (mode === 'linux-window') api().OpenURL(url).catch(() => {});
  else if (mode === 'windows-app') runtime().BrowserOpenURL(url);
  else window.open(url, '_blank', 'noopener,noreferrer');
}
