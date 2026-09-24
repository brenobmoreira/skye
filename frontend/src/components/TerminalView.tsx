import { useEffect, useRef } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebglAddon } from '@xterm/addon-webgl';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { api, input, openExternal, output } from '../bridge';
import { keyHandler } from '../lib/keys';
import { linkClick } from '../lib/links';
import { LINE_HEIGHT } from '../lib/fonts';
import type { Slot } from '../lib/panes';

// slot is where the terminal shows (null: hidden); focused gets the keyboard.
export function TerminalView({ id, slot, focused, onFocus, fontSize, fontFamily }: { id: string; slot: Slot | null; focused: boolean; onFocus: () => void; fontSize: number; fontFamily: string }) {
  const host = useRef<HTMLDivElement>(null);
  const term = useRef<XTerm | null>(null);
  const fit = useRef<FitAddon | null>(null);

  useEffect(() => {
    const open = (e: MouseEvent, uri: string) => {
      const url = linkClick(e, uri);
      if (url) openExternal(url);
    };
    const t = new XTerm({
      fontFamily,
      fontSize,
      lineHeight: LINE_HEIGHT,
      cursorBlink: true,
      scrollback: 5000,
      theme: { background: '#0e0e12' },
      linkHandler: { activate: (e, uri) => open(e, uri) },
    });
    const f = new FitAddon();
    t.loadAddon(f);
    t.loadAddon(new WebLinksAddon(open));
    t.open(host.current!);
    // WebGL draws block and box-drawing characters itself, so they fill the cell instead of coming from a fallback font.
    try {
      const webgl = new WebglAddon();
      webgl.onContextLoss(() => webgl.dispose());
      t.loadAddon(webgl);
    } catch {}
    term.current = t;
    fit.current = f;

    t.attachCustomKeyEventHandler(keyHandler(
      (data) => input(id).write(data),
      () => { navigator.clipboard.readText().then((text) => text && t.paste(text)).catch(() => {}); },
    ));
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
    if (!slot) return;
    fit.current?.fit();
    api().Resize(id, t.cols, t.rows);
  }, [fontSize, slot, id]);

  useEffect(() => {
    let cancelled = false;
    // Measure the cells only once the web font is in, or the grid keeps the fallback font's width.
    document.fonts.load(`${fontSize}px ${fontFamily}`).catch(() => {}).then(() => {
      const t = term.current;
      if (cancelled || !t) return;
      t.options.fontFamily = fontFamily;
      if (!slot) return;
      fit.current?.fit();
      api().Resize(id, t.cols, t.rows);
    });
    return () => { cancelled = true; };
  }, [fontFamily, slot, id]);

  useEffect(() => {
    if (!slot || !host.current) return;
    const resize = () => {
      const t = term.current;
      const f = fit.current;
      if (!t || !f) return;
      f.fit();
      api().Resize(id, t.cols, t.rows);
    };
    resize();
    const observer = new ResizeObserver(resize);
    observer.observe(host.current);
    return () => observer.disconnect();
  }, [slot, id]);

  useEffect(() => {
    if (slot && focused) term.current?.focus();
  }, [slot, focused]);

  return (
    <div
      className={`terminal ${slot ?? ''}${focused && slot !== 'full' ? ' focused' : ''}`}
      ref={host}
      onMouseDown={onFocus}
      style={{ visibility: slot ? 'visible' : 'hidden' }}
    />
  );
}
