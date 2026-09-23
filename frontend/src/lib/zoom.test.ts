import { describe, expect, it } from 'vitest';
import { DEFAULT_FONT_SIZE, MAX_FONT_SIZE, MIN_FONT_SIZE, applyZoom, parseFontSize, zoomKey } from './zoom';

const key = (over: Partial<KeyboardEvent>) =>
  ({ key: 'a', ctrlKey: true, altKey: false, metaKey: false, ...over }) as KeyboardEvent;

describe('zoomKey', () => {
  it('maps Ctrl with =, +, -, 0', () => {
    expect(zoomKey(key({ key: '=' }))).toBe('in');
    expect(zoomKey(key({ key: '+' }))).toBe('in');
    expect(zoomKey(key({ key: '-' }))).toBe('out');
    expect(zoomKey(key({ key: '0' }))).toBe('reset');
  });
  it('ignores keys without Ctrl or with Alt', () => {
    expect(zoomKey(key({ key: '=', ctrlKey: false }))).toBeNull();
    expect(zoomKey(key({ key: '-', altKey: true }))).toBeNull();
    expect(zoomKey(key({ key: 'c' }))).toBeNull();
  });
});

describe('applyZoom', () => {
  it('steps by one and clamps', () => {
    expect(applyZoom(16, 'in')).toBe(17);
    expect(applyZoom(16, 'out')).toBe(15);
    expect(applyZoom(MAX_FONT_SIZE, 'in')).toBe(MAX_FONT_SIZE);
    expect(applyZoom(MIN_FONT_SIZE, 'out')).toBe(MIN_FONT_SIZE);
    expect(applyZoom(25, 'reset')).toBe(DEFAULT_FONT_SIZE);
  });
});

describe('parseFontSize', () => {
  it('falls back to the default for missing or invalid values', () => {
    expect(parseFontSize('20')).toBe(20);
    expect(parseFontSize(null)).toBe(DEFAULT_FONT_SIZE);
    expect(parseFontSize('abc')).toBe(DEFAULT_FONT_SIZE);
    expect(parseFontSize('99')).toBe(DEFAULT_FONT_SIZE);
  });
});
