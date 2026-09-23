import type { State } from './types';

export const stateLabel: Record<State, string> = {
  waiting: 'esperando você',
  idle: 'terminou',
  running: 'trabalhando',
  shell: 'terminal',
};
