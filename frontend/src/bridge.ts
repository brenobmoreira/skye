import type { Conversation, OutputEvent, Preset, Terminal } from './lib/types';
import { createOutputHub } from './output';
import { createSendQueue } from './sendQueue';

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
    go: { main: { Bridge: GoBridge } };
    runtime: WailsRuntime;
  }
}

export const api = (): GoBridge => window.go.main.Bridge;
export const runtime = (): WailsRuntime => window.runtime;
export const on = (name: string, cb: (...data: any[]) => void) => window.runtime.EventsOn(name, cb);
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
