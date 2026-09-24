import { stateLabel } from './states';
import type { Terminal } from './types';

export const base = (path: string) => path.split('/').filter(Boolean).pop() ?? path;

// A shell nobody renamed takes its name from the Claude session running in it.
const defaultName = (t: Terminal) => t.preset === '' && t.name === 'shell';

export function itemLabel(t: Terminal): { name: string; sub: string } {
  const folder = base(t.cwd);
  if (!defaultName(t)) return { name: t.name, sub: t.title || folder || stateLabel[t.state] };
  if (t.title) return { name: t.title, sub: folder || stateLabel[t.state] };
  return { name: folder || t.name, sub: stateLabel[t.state] };
}
