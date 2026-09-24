import type { SocketLike } from './wsClient';

export interface Endpoint {
  url: string;
  token: string;
}

export type ConnectionState = { kind: 'connecting' } | { kind: 'ready' } | { kind: 'error'; message: string };

export interface WindowsTransportOptions {
  connect: () => Promise<Endpoint>;
  open: (url: string) => SocketLike;
  onState: (state: ConnectionState) => void;
}

const ESCALATE_AFTER = 3;

const messageOf = (err: unknown) =>
  typeof err === 'string' ? err : err instanceof Error ? err.message : String(err);

export function createWindowsTransport({ connect, open, onState }: WindowsTransportOptions) {
  let endpoint: Endpoint | null = null;
  let failures = 0;
  let waiting: (() => void) | null = null;

  const dial = (): SocketLike => {
    let inner: SocketLike | null = null;
    let closed = false;
    let opened = false;
    const outer: SocketLike = {
      send: (data) => inner?.send(data),
      close: () => {
        closed = true;
        inner?.close();
      },
      onopen: null,
      onclose: null,
      onmessage: null,
      onerror: null,
    };

    const attach = (ep: Endpoint) => {
      if (closed) return;
      const s = open(`${ep.url}?token=${encodeURIComponent(ep.token)}`);
      inner = s;
      s.onopen = () => {
        opened = true;
        failures = 0;
        onState({ kind: 'ready' });
        outer.onopen?.();
      };
      s.onmessage = (ev) => outer.onmessage?.(ev);
      s.onerror = () => outer.onerror?.();
      s.onclose = () => {
        if (!opened) failures++;
        outer.onclose?.();
      };
    };

    const resolve = () => {
      onState({ kind: 'connecting' });
      connect().then(
        (ep) => {
          endpoint = ep;
          failures = 0;
          attach(ep);
        },
        (err) => {
          waiting = () => {
            waiting = null;
            resolve();
          };
          onState({ kind: 'error', message: messageOf(err) });
        },
      );
    };

    if (endpoint && failures < ESCALATE_AFTER) {
      attach(endpoint);
    } else {
      endpoint = null;
      resolve();
    }
    return outer;
  };

  return { dial, retry: () => waiting?.() };
}
