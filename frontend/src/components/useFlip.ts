import { useLayoutEffect, useRef, type RefObject } from 'react';
import { flipDeltas, translateYOf } from '../lib/flip';

const calm = () => typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;

const SLIDE = 'flip';

// useFlip slides the [data-flip] children of ref to their new place after each render that moved
// them, so a list being reordered pushes its items up or down instead of jumping. An item that
// moves again mid-slide starts the new slide from where it is drawn, not from its old place.
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
      // Where each moved item is drawn now: its old place plus the slide it is in the middle of.
      const drawn = new Map(before.current);
      for (const [id, n] of nodes) {
        const was = before.current.get(id);
        if (was === undefined || was === after.get(id) || typeof n.getAnimations !== 'function') continue;
        const sliding = n.getAnimations().filter((a) => a.id === SLIDE);
        if (sliding.length === 0) continue;
        drawn.set(id, was + translateYOf(getComputedStyle(n).transform));
        sliding.forEach((a) => a.cancel());
      }
      for (const [id, dy] of flipDeltas(drawn, after)) {
        nodes.get(id)?.animate([{ transform: `translateY(${dy}px)` }, { transform: 'none' }], { id: SLIDE, duration: 180, easing: 'cubic-bezier(0.2, 0.8, 0.2, 1)' });
      }
    }
    before.current = after;
  });
}
