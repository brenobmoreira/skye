export type State = 'shell' | 'running' | 'waiting' | 'idle';

export interface Terminal {
  id: string;
  name: string;
  preset: string;
  cwd: string;
  state: State;
  sessionId: string;
  title: string;
  ask?: string;
  createdAt: string;
  order: number;
}

export interface Conversation {
  sessionId: string;
  cwd: string;
  title: string;
  preset: string;
  endedAt: string;
}

export interface UsageWindow {
  usedPct: number;
  resetsAt: number;
}

export interface Usage {
  fiveHour: UsageWindow | null;
  sevenDay: UsageWindow | null;
  updatedAt: string;
}

export interface Preset {
  name: string;
  command: string;
}

export interface OutputEvent {
  id: string;
  data: string;
}
