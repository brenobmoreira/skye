import type { State } from './types';

// Closing kills whatever runs in the terminal, so a claude mid-task or waiting for an answer asks
// first; a finished claude or a plain shell has nothing to lose.
export const asksBeforeClose = (state: State) => state === 'running' || state === 'waiting';
