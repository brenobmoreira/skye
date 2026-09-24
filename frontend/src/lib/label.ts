import { stateLabel } from './states';
import type { Terminal } from './types';

export const base = (path: string) => path.split('/').filter(Boolean).pop() ?? path;

// A shell nobody renamed takes its name from the Claude session running in it.
const defaultName = (t: Terminal) => t.preset === '' && t.name === 'shell';

export function itemLabel(t: Terminal): { name: string; sub: string } {
  const folder = base(t.cwd);
  const ask = t.state === 'waiting' ? t.ask : '';
  if (!defaultName(t)) return { name: t.name, sub: ask || t.title || folder || stateLabel[t.state] };
  if (t.title) return { name: t.title, sub: ask || folder || stateLabel[t.state] };
  return { name: folder || t.name, sub: ask || stateLabel[t.state] };
}
