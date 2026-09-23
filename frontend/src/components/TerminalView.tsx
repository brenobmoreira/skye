import { useEffect, useRef } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { api, input, output } from '../bridge';
import { terminalKey } from '../lib/keys';

export function TerminalView({ id, active }: { id: string; active: boolean }) {
  const host = useRef<HTMLDivElement>(null);
  const term = useRef<XTerm | null>(null);
  const fit = useRef<FitAddon | null>(null);

  useEffect(() => {
    const t = new XTerm({
      fontFamily: '"JetBrains Mono", monospace',
      fontSize: 14,
      cursorBlink: true,
      scrollback: 5000,
      theme: { background: '#0e0e12' },
    });
    const f = new FitAddon();
    t.loadAddon(f);
    t.open(host.current!);
    term.current = t;
    fit.current = f;

    t.attachCustomKeyEventHandler((e) => {
      const action = terminalKey(e);
      if (action.kind === 'send') {
        input(id).write(action.data);
        return false;
      }
      if (action.kind === 'paste') {
        navigator.clipboard.readText().then((text) => text && t.paste(text)).catch(() => {});
        return false;
      }
      return true;
    });
    t.onData((data) => input(id).write(data));
    t.onSelectionChange(() => {
      const selected = t.getSelection();
      if (selected) navigator.clipboard.writeText(selected).catch(() => {});
    });

    let unsubscribe = () => {};
    let disposed = false;
    api().Snapshot(id).then((snapshot) => {
      if (disposed) return;
      t.write(snapshot);
      unsubscribe = output.subscribe(id, (bytes) => t.write(bytes));
    });

    return () => {
      disposed = true;
      unsubscribe();
      t.dispose();
    };
  }, [id]);

  useEffect(() => {
    if (!active || !host.current) return;
    const resize = () => {
      const t = term.current;
      const f = fit.current;
      if (!t || !f) return;
      f.fit();
      api().Resize(id, t.cols, t.rows);
    };
    resize();
    term.current?.focus();
    const observer = new ResizeObserver(resize);
    observer.observe(host.current);
    return () => observer.disconnect();
  }, [active, id]);

  return <div className="terminal" ref={host} style={{ visibility: active ? 'visible' : 'hidden' }} />;
}
