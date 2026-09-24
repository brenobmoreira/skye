import { stateLabel } from './states';
import type { Terminal } from './types';

export const base = (path: string) => path.split('/').filter(Boolean).pop() ?? path;

// A shell nobody renamed takes its name from the Claude session running in it.
const defaultName = (t: Terminal) => t.preset === '' && t.name === 'shell';

export function itemLabel(t: Terminal): { name: string; sub: string } {
  const folder = base(t.cwd);
  // What a waiting terminal asks or a running one is busy with takes the line under the name.
  const live = t.state === 'waiting' ? t.ask : t.state === 'running' ? t.activity : '';
  if (!defaultName(t)) return { name: t.name, sub: live || t.title || folder || stateLabel[t.state] };
  if (t.title) return { name: t.title, sub: live || folder || stateLabel[t.state] };
  return { name: folder || t.name, sub: live || stateLabel[t.state] };
}
