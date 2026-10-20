import { useMemo, type MouseEvent } from 'react';
import { marked } from 'marked';
import DOMPurify from 'dompurify';
import { openExternal } from '../bridge';
import { linkTarget } from '../lib/files';

// A markdown file rendered. The HTML is sanitized first: the page can call the skye, so a file
// must not be able to run script in it.
export function Markdown({ text, path, onOpen }: { text: string; path: string; onOpen: (path: string) => void }) {
  const html = useMemo(() => DOMPurify.sanitize(marked.parse(text, { async: false, gfm: true })), [text]);
  const click = (e: MouseEvent) => {
    const a = (e.target as HTMLElement).closest('a');
    if (!a) return;
    e.preventDefault();
    const target = linkTarget(a.getAttribute('href') ?? '', path);
    if (target.kind === 'web') openExternal(target.url);
    else if (target.kind === 'file') onOpen(target.path);
  };
  return <div className="md" onClick={click} dangerouslySetInnerHTML={{ __html: html }} />;
}
