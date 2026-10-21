// Paths in the files panel are relative to the terminal's root and use forward slashes.

export const joinPath = (dir: string, name: string) => (dir ? `${dir}/${name}` : name);

export function splitPath(path: string): { dir: string; name: string } {
  const i = path.lastIndexOf('/');
  return i < 0 ? { dir: '', name: path } : { dir: path.slice(0, i), name: path.slice(i + 1) };
}

export function crumbs(path: string): { name: string; path: string }[] {
  const parts = path.split('/').filter(Boolean);
  return parts.map((name, i) => ({ name, path: parts.slice(0, i + 1).join('/') }));
}

export const isMarkdown = (path: string) => /\.(md|markdown)$/i.test(path);

const labels: Record<string, string> = { M: 'modificado', A: 'novo', '?': 'novo, fora do git', D: 'apagado' };

export const changeLabel = (status: string) => labels[status] ?? status;

export type LinkTarget = { kind: 'web'; url: string } | { kind: 'file'; path: string } | { kind: 'none' };

// Where a link clicked in a rendered markdown file leads: the browser for web links, the panel for
// a file next to it, nowhere for anything else.
export function linkTarget(href: string, from: string): LinkTarget {
  if (/^https?:\/\//i.test(href)) return { kind: 'web', url: href };
  if (href === '' || href.startsWith('#') || href.startsWith('/') || /^[a-z][a-z0-9+.-]*:/i.test(href)) return { kind: 'none' };
  const parts = splitPath(from).dir.split('/').filter(Boolean);
  for (const part of href.split(/[?#]/)[0].split('/')) {
    if (part === '' || part === '.') continue;
    if (part === '..') {
      if (parts.length === 0) return { kind: 'none' };
      parts.pop();
    } else parts.push(part);
  }
  return parts.length ? { kind: 'file', path: parts.join('/') } : { kind: 'none' };
}
