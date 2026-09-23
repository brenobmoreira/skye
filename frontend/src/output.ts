import { fromBase64 } from './lib/bytes';
import type { OutputEvent } from './lib/types';

type On = (event: string, cb: (ev: OutputEvent) => void) => void;

function decode(data: string): Uint8Array | null {
  try {
    return fromBase64(data);
  } catch {
    return null;
  }
}

export function createOutputHub(on: On) {
  const listeners = new Map<string, (bytes: Uint8Array) => void>();
  const pending = new Map<string, Uint8Array[]>();

  on('output', (ev) => {
    const bytes = decode(ev.data);
    if (!bytes) return;
    const listener = listeners.get(ev.id);
    if (listener) {
      listener(bytes);
      return;
    }
    pending.get(ev.id)?.push(bytes);
  });

  return {
    prepare(id: string) {
      pending.set(id, []);
    },
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
