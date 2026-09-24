import { describe, expect, it } from 'vitest';
import { linkClick } from './links';

const click = (over: Partial<MouseEvent>) => ({ ctrlKey: false, metaKey: false, ...over }) as MouseEvent;

describe('linkClick', () => {
  it('opens web links on Ctrl+click or Cmd+click', () => {
    expect(linkClick(click({ ctrlKey: true }), 'https://example.com')).toBe('https://example.com');
    expect(linkClick(click({ metaKey: true }), 'http://localhost:3000/x')).toBe('http://localhost:3000/x');
  });
  it('ignores a plain click', () => {
    expect(linkClick(click({}), 'https://example.com')).toBeNull();
  });
  it('refuses anything but http and https', () => {
    for (const uri of ['file:///etc/passwd', 'javascript:alert(1)', '/home/demo', 'not a url']) {
      expect(linkClick(click({ ctrlKey: true }), uri)).toBeNull();
    }
  });
});
