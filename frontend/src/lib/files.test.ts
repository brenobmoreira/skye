import { describe, expect, it } from 'vitest';
import { changeLabel, crumbs, isMarkdown, joinPath, linkTarget, splitPath } from './files';

describe('paths', () => {
  it('joins and splits paths relative to the terminal root', () => {
    expect(joinPath('', 'docs')).toBe('docs');
    expect(joinPath('docs', 'readme.md')).toBe('docs/readme.md');
    expect(splitPath('internal/app/app.go')).toEqual({ dir: 'internal/app', name: 'app.go' });
    expect(splitPath('.env')).toEqual({ dir: '', name: '.env' });
  });

  it('lists the folders above a path for the breadcrumb', () => {
    expect(crumbs('')).toEqual([]);
    expect(crumbs('internal/app')).toEqual([
      { name: 'internal', path: 'internal' },
      { name: 'app', path: 'internal/app' },
    ]);
  });
});

describe('isMarkdown', () => {
  it('matches markdown files by extension', () => {
    expect(isMarkdown('README.md')).toBe(true);
    expect(isMarkdown('docs/x.MARKDOWN')).toBe(true);
    expect(isMarkdown('.env')).toBe(false);
  });
});

describe('changeLabel', () => {
  it('names each git status', () => {
    expect(changeLabel('M')).toBe('modificado');
    expect(changeLabel('A')).toBe('novo');
    expect(changeLabel('?')).toBe('novo, fora do git');
    expect(changeLabel('D')).toBe('apagado');
    expect(changeLabel('T')).toBe('T');
  });
});

describe('linkTarget', () => {
  it('sends web links to the browser', () => {
    expect(linkTarget('https://example.com/x', 'docs/a.md')).toEqual({ kind: 'web', url: 'https://example.com/x' });
  });

  it('opens a relative link as a file next to the markdown', () => {
    expect(linkTarget('b.md', 'docs/a.md')).toEqual({ kind: 'file', path: 'docs/b.md' });
    expect(linkTarget('../README.md#install', 'docs/a.md')).toEqual({ kind: 'file', path: 'README.md' });
    expect(linkTarget('./img/x.png', 'a.md')).toEqual({ kind: 'file', path: 'img/x.png' });
  });

  it('ignores anchors, other schemes and links that leave the root', () => {
    expect(linkTarget('#install', 'a.md')).toEqual({ kind: 'none' });
    expect(linkTarget('javascript:alert(1)', 'a.md')).toEqual({ kind: 'none' });
    expect(linkTarget('mailto:x@y', 'a.md')).toEqual({ kind: 'none' });
    expect(linkTarget('../../x.md', 'docs/a.md')).toEqual({ kind: 'none' });
    expect(linkTarget('/etc/passwd', 'a.md')).toEqual({ kind: 'none' });
  });
});
