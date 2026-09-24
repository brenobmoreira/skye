// Sends one call at a time, merging writes queued while one is in flight, so keystrokes keep
// their order.
export function createSendQueue(send: (data: string) => Promise<unknown>) {
  const queue: string[] = [];
  let busy = false;

  const next = () => {
    const data = queue.shift();
    if (data === undefined) {
      busy = false;
      return;
    }
    busy = true;
    send(data).catch(() => {}).then(next);
  };

  return {
    write: (data: string) => {
      if (queue.length > 0) queue[queue.length - 1] += data;
      else queue.push(data);
      if (!busy) next();
    },
  };
}
