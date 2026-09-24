import { useEffect, useRef } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebglAddon } from '@xterm/addon-webgl';
import { api, input, output } from '../bridge';
import { terminalKey } from '../lib/keys';

export function TerminalView({ id, active, fontSize, fontFamily }: { id: string; active: boolean; fontSize: number; fontFamily: string }) {
  const host = useRef<HTMLDivElement>(null);
  const term = useRef<XTerm | null>(null);
  const fit = useRef<FitAddon | null>(null);

  useEffect(() => {
    const t = new XTerm({
      fontFamily,
      fontSize,
      cursorBlink: true,
      scrollback: 5000,
      theme: { background: '#0e0e12' },
    });
    const f = new FitAddon();
    t.loadAddon(f);
    t.open(host.current!);
    // WebGL draws block and box-drawing characters itself, so they fill the cell instead of coming from a fallback font.
    try {
      const webgl = new WebglAddon();
      webgl.onContextLoss(() => webgl.dispose());
      t.loadAddon(webgl);
    } catch {}
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
    output.prepare(id);
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
    const t = term.current;
    if (!t || t.options.fontSize === fontSize) return;
    t.options.fontSize = fontSize;
    if (!active) return;
    fit.current?.fit();
    api().Resize(id, t.cols, t.rows);
  }, [fontSize, active, id]);

  useEffect(() => {
    let cancelled = false;
    // Measure the cells only once the web font is in, or the grid keeps the fallback font's width.
    document.fonts.load(`${fontSize}px ${fontFamily}`).catch(() => {}).then(() => {
      const t = term.current;
      if (cancelled || !t) return;
      t.options.fontFamily = fontFamily;
      if (!active) return;
      fit.current?.fit();
      api().Resize(id, t.cols, t.rows);
    });
    return () => { cancelled = true; };
  }, [fontFamily, active, id]);

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
