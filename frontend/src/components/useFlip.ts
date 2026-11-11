import { useLayoutEffect, useRef, type RefObject } from 'react';
import { flipDeltas } from '../lib/flip';

const calm = () => typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;

// useFlip slides the [data-flip] children of ref to their new place after each render that moved
// them, so a list being reordered pushes its items up or down instead of jumping.
export function useFlip(ref: RefObject<HTMLElement | null>) {
  const before = useRef(new Map<string, number>());
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    const nodes = new Map<string, HTMLElement>();
    const after = new Map<string, number>();
    el.querySelectorAll<HTMLElement>('[data-flip]').forEach((n) => {
      const id = n.dataset.flip ?? '';
      nodes.set(id, n);
      after.set(id, n.offsetTop);
    });
    if (!calm()) {
      for (const [id, dy] of flipDeltas(before.current, after)) {
        nodes.get(id)?.animate([{ transform: `translateY(${dy}px)` }, { transform: 'none' }], { duration: 180, easing: 'cubic-bezier(0.2, 0.8, 0.2, 1)' });
      }
    }
    before.current = after;
  });
}
