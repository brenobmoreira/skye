export interface SocketLike {
  send(data: string): void;
  close(): void;
  onopen: (() => void) | null;
  onclose: (() => void) | null;
  onmessage: ((ev: { data: string }) => void) | null;
  onerror: (() => void) | null;
}

export interface WsClientOptions {
  connect: () => SocketLike;
  schedule?: (fn: () => void, ms: number) => void;
}

type Listener = (data: any) => void;
type Pending = { resolve: (value: any) => void; reject: (err: Error) => void };

const BASE_DELAY = 250;
const MAX_DELAY = 5000;

export function createWsClient({ connect, schedule = (fn, ms) => { setTimeout(fn, ms); } }: WsClientOptions) {
  const listeners = new Map<string, Set<Listener>>();
  const pending = new Map<number, Pending>();
  let outbox: string[] = [];
  let nextId = 1;
  let attempt = 0;
  let everOpened = false;
  let socket: SocketLike;
  let open = false;

  const emit = (name: string, data?: unknown) => {
    listeners.get(name)?.forEach((cb) => cb(data));
  };

  const failAll = () => {
    const err = new Error('conexão com a skye perdida');
    pending.forEach((p) => p.reject(err));
    pending.clear();
    outbox = [];
  };

  const receive = (raw: string) => {
    let msg: any;
    try {
      msg = JSON.parse(raw);
    } catch {
      return;
    }
    if (typeof msg?.event === 'string') {
      emit(msg.event, msg.data);
      return;
    }
    const p = pending.get(msg?.id);
    if (!p) return;
    pending.delete(msg.id);
    if (typeof msg.error === 'string') p.reject(new Error(msg.error));
    else p.resolve(msg.result);
  };

  const dial = () => {
    const s = connect();
    socket = s;
    s.onopen = () => {
      open = true;
      attempt = 0;
      const frames = outbox;
      outbox = [];
      frames.forEach((f) => s.send(f));
      if (everOpened) emit('reconnected');
      everOpened = true;
    };
    s.onmessage = (ev) => receive(ev.data);
    s.onerror = () => {};
    s.onclose = () => {
      if (socket !== s) return;
      open = false;
      failAll();
      const delay = Math.min(MAX_DELAY, BASE_DELAY * 2 ** attempt);
      attempt++;
      schedule(dial, delay);
    };
  };

  dial();

  return {
    call<T = unknown>(method: string, ...args: unknown[]): Promise<T> {
      const id = nextId++;
      const frame = JSON.stringify({ id, method, args });
      return new Promise<T>((resolve, reject) => {
        pending.set(id, { resolve, reject });
        if (open) socket.send(frame);
        else outbox.push(frame);
      });
    },
    on(name: string, cb: Listener): () => void {
      let set = listeners.get(name);
      if (!set) {
        set = new Set();
        listeners.set(name, set);
      }
      set.add(cb);
      return () => { set.delete(cb); };
    },
  };
}
