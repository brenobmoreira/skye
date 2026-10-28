import { useEffect, useRef } from 'react';

export type MenuItem = { label: string; run: () => void };

// A right-click menu at the pointer; Esc, a click elsewhere or leaving the window closes it.
export function ContextMenu({ x, y, items, onClose }: { x: number; y: number; items: MenuItem[]; onClose: () => void }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const outside = (e: MouseEvent) => { if (!ref.current?.contains(e.target as Node)) onClose(); };
    const key = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('mousedown', outside, true);
    window.addEventListener('keydown', key, true);
    window.addEventListener('blur', onClose);
    return () => {
      window.removeEventListener('mousedown', outside, true);
      window.removeEventListener('keydown', key, true);
      window.removeEventListener('blur', onClose);
    };
  }, [onClose]);
  return (
    <div ref={ref} className="menu-list context-menu" style={{ left: Math.min(x, window.innerWidth - 200), top: Math.min(y, window.innerHeight - 40 * items.length) }} onContextMenu={(e) => e.preventDefault()}>
      {items.map((i) => (
        <button key={i.label} onClick={() => { onClose(); i.run(); }}>{i.label}</button>
      ))}
    </div>
  );
}
