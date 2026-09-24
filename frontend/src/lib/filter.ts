import { base } from './label';
import type { Conversation } from './types';

const fold = (s: string) => s.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLowerCase();

// Keeps the conversations whose title or folder has every word of the query.
export function filterConversations(list: Conversation[], query: string): Conversation[] {
  const words = fold(query).split(/\s+/).filter(Boolean);
  if (words.length === 0) return list;
  return list.filter((c) => {
    const text = fold(`${c.title} ${base(c.cwd)}`);
    return words.every((w) => text.includes(w));
  });
}
