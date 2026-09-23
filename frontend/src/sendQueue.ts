export type SendOp = { kind: 'write' | 'paste'; data: string };

export function createSendQueue(send: (op: SendOp) => Promise<unknown>) {
  const queue: SendOp[] = [];
  let busy = false;

  const next = () => {
    const op = queue.shift();
    if (!op) {
      busy = false;
      return;
    }
    busy = true;
    send(op).catch(() => {}).then(next);
  };

  const push = (op: SendOp) => {
    const last = queue[queue.length - 1];
    if (op.kind === 'write' && last?.kind === 'write') last.data += op.data;
    else queue.push(op);
    if (!busy) next();
  };

  return {
    write: (data: string) => push({ kind: 'write', data }),
    paste: (data: string) => push({ kind: 'paste', data }),
  };
}
