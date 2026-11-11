import { describe, expect, it, vi } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { NoteList } from './NoteList';

vi.mock('../bridge', () => ({ api: () => ({}) }));

describe('NoteList', () => {
  it('does not reuse class names of the app layout', () => {
    const html = renderToStaticMarkup(
      <NoteList notes={{ items: [{ id: 'n1', text: 'revisar', date: '2026-11-11', time: '14:00', created: '2026-11-11T09:00:00Z' }], done: [] }} now={new Date(2026, 10, 11, 10, 0)} />,
    );
    expect(html).toContain('revisar');
    // .body is the app's grid; a row inside it must not pick up its columns and padding.
    expect(html).not.toMatch(/class="(?:[^"]*\s)?body(?:\s[^"]*)?"/);
  });
});
