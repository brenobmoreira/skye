import { describe, expect, it } from 'vitest';
import { filterConversations } from './filter';
import type { Conversation } from './types';

const c = (sessionId: string, title: string, cwd: string): Conversation => ({ sessionId, title, cwd, preset: '', endedAt: '' });
const list = [
  c('1', 'Corrigir a configuração do login', '/home/breno/projects/skye'),
  c('2', 'revisar PR', '/home/breno/projects/ai_livia_copilot'),
  c('3', 'deploy', '/home/breno/projects/wt-76440'),
];

describe('filterConversations', () => {
  it('keeps everything for an empty query', () => {
    expect(filterConversations(list, '  ')).toBe(list);
  });
  it('matches the title ignoring case and accents', () => {
    expect(filterConversations(list, 'CONFIGURACAO').map((x) => x.sessionId)).toEqual(['1']);
  });
  it('matches the folder name', () => {
    expect(filterConversations(list, 'livia').map((x) => x.sessionId)).toEqual(['2']);
  });
  it('needs every word to match somewhere', () => {
    expect(filterConversations(list, 'login skye').map((x) => x.sessionId)).toEqual(['1']);
    expect(filterConversations(list, 'login livia')).toEqual([]);
  });
  it('does not match the path above the folder', () => {
    expect(filterConversations(list, 'breno')).toEqual([]);
  });
});
