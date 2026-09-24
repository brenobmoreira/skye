import { describe, expect, it } from 'vitest';
import { active, closeSplit, prune, select, selectSide, slotOf, visible, type Panes } from './panes';

const one: Panes = { left: 'a', right: null, focus: 'left' };
const two: Panes = { left: 'a', right: 'b', focus: 'right' };

describe('select', () => {
  it('shows the terminal in the focused pane', () => {
    expect(select(one, 'c')).toEqual({ left: 'c', right: null, focus: 'left' });
    expect(select(two, 'c')).toEqual({ left: 'a', right: 'c', focus: 'right' });
  });
  it('only moves the focus when the terminal is already on screen', () => {
    expect(select(two, 'a')).toEqual({ left: 'a', right: 'b', focus: 'left' });
    expect(select(one, 'a')).toBe(one);
  });
});

describe('selectSide', () => {
  it('opens the terminal on the right and focuses it', () => {
    expect(selectSide(one, 'c')).toEqual({ left: 'a', right: 'c', focus: 'right' });
    expect(selectSide(two, 'c')).toEqual({ left: 'a', right: 'c', focus: 'right' });
  });
  it('focuses a terminal already on screen', () => {
    expect(selectSide({ ...two, focus: 'right' }, 'a')).toEqual({ left: 'a', right: 'b', focus: 'left' });
    expect(selectSide(one, 'a')).toBe(one);
  });
  it('fills the left pane first', () => {
    expect(selectSide({ left: null, right: null, focus: 'left' }, 'c')).toEqual({ left: 'c', right: null, focus: 'left' });
  });
});

describe('closeSplit', () => {
  it('keeps the focused terminal alone', () => {
    expect(closeSplit(two)).toEqual({ left: 'b', right: null, focus: 'left' });
    expect(closeSplit({ ...two, focus: 'left' })).toEqual({ left: 'a', right: null, focus: 'left' });
  });
});

describe('prune', () => {
  it('drops terminals that closed and falls back to the first one', () => {
    expect(prune(two, ['a', 'c'])).toEqual({ left: 'a', right: null, focus: 'left' });
    expect(prune(two, ['b', 'c'])).toEqual({ left: 'b', right: null, focus: 'left' });
    expect(prune(one, ['c'])).toEqual({ left: 'c', right: null, focus: 'left' });
    expect(prune(one, [])).toEqual({ left: null, right: null, focus: 'left' });
  });
  it('keeps the same object when nothing changed', () => {
    expect(prune(two, ['a', 'b'])).toBe(two);
  });
});

describe('slots', () => {
  it('splits only when there is room', () => {
    expect(slotOf(two, 'a', true)).toBe('left');
    expect(slotOf(two, 'b', true)).toBe('right');
    expect(slotOf(two, 'c', true)).toBeNull();
    expect(slotOf(two, 'b', false)).toBe('full');
    expect(slotOf(two, 'a', false)).toBeNull();
    expect(slotOf(one, 'a', true)).toBe('full');
  });
  it('knows the focused and the visible terminals', () => {
    expect(active(two)).toBe('b');
    expect(visible(two, true)).toEqual(['a', 'b']);
    expect(visible(two, false)).toEqual(['b']);
    expect(visible({ left: null, right: null, focus: 'left' }, true)).toEqual([]);
  });
});
