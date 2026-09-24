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
const GIVE_UP_AFTER = 2;

const messageOf = (err: unknown) =>
  typeof err === 'string' ? err : err instanceof Error ? err.message : String(err);

const portOf = (url: string) => {
  try {
    return new URL(url).port;
  } catch {
    return url;
  }
};

const neverOpened = (ep: Endpoint) =>
  `o servidor respondeu na porta ${portOf(ep.url)}, mas o WebSocket ${ep.url} não abriu (origem/token recusados, outra skye aberta ou bloqueio do WebView2)`;

export function createWindowsTransport({ connect, open, onState }: WindowsTransportOptions) {
  let endpoint: Endpoint | null = null;
  let failures = 0;
  let fruitless = 0;
  let openedSinceConnect = false;
  let halted = false;
  let parked: (() => void) | null = null;

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

    const halt = (message: string) => {
      halted = true;
      onState({ kind: 'error', message });
      queueMicrotask(() => outer.onclose?.());
    };

    const attach = (ep: Endpoint) => {
      if (closed) return;
      const s = open(`${ep.url}?token=${encodeURIComponent(ep.token)}`);
      inner = s;
      s.onopen = () => {
        opened = true;
        openedSinceConnect = true;
        failures = 0;
        fruitless = 0;
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
          openedSinceConnect = false;
          attach(ep);
        },
        (err) => halt(messageOf(err)),
      );
    };

    const begin = () => {
      if (endpoint && failures < ESCALATE_AFTER) {
        attach(endpoint);
        return;
      }
      if (endpoint && !openedSinceConnect && ++fruitless >= GIVE_UP_AFTER) {
        halt(neverOpened(endpoint));
        return;
      }
      endpoint = null;
      resolve();
    };

    if (halted) parked = begin;
    else begin();
    return outer;
  };

  const retry = () => {
    if (!halted) return;
    halted = false;
    endpoint = null;
    failures = 0;
    fruitless = 0;
    const next = parked;
    parked = null;
    next?.();
  };

  return { dial, retry };
}
