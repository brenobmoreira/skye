// A link in the terminal opens on Ctrl+click (Cmd+click on a Mac), like in VS Code, so a plain
// click can still select text; only http and https links ever leave the terminal.
export function linkClick(e: Pick<MouseEvent, 'ctrlKey' | 'metaKey'>, uri: string): string | null {
  if (!e.ctrlKey && !e.metaKey) return null;
  try {
    const { protocol } = new URL(uri);
    return protocol === 'http:' || protocol === 'https:' ? uri : null;
  } catch {
    return null;
  }
}
