import { fromBase64 } from './lib/bytes';
import type { OutputEvent } from './lib/types';

type On = (event: string, cb: (ev: OutputEvent) => void) => void;

export function createOutputHub(on: On) {
  const listeners = new Map<string, (bytes: Uint8Array) => void>();
  const pending = new Map<string, Uint8Array[]>();

  on('output', (ev) => {
    const bytes = fromBase64(ev.data);
    const listener = listeners.get(ev.id);
    if (listener) {
      listener(bytes);
      return;
    }
    const queue = pending.get(ev.id) ?? [];
    queue.push(bytes);
    pending.set(ev.id, queue);
  });

  return {
    subscribe(id: string, cb: (bytes: Uint8Array) => void): () => void {
      listeners.set(id, cb);
      pending.get(id)?.forEach(cb);
      pending.delete(id);
      return () => {
        if (listeners.get(id) === cb) listeners.delete(id);
      };
    },
  };
}
