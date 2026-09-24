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
  activity?: string;
  context?: SessionContext;
  createdAt: string;
  since?: string;
  order: number;
}

export interface SessionContext {
  contextPct?: number;
  model?: string;
  costUsd?: number;
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

export interface Machine {
  memTotalMb: number;
  memAvailMb: number;
  swapTotalMb: number;
  swapUsedMb: number;
  psiSome60: number;
  load1: number;
}

export interface MemoryPoint {
  t: string;
  availMb: number;
}

export interface TerminalLoad {
  id: string;
  pid: number;
  procs: number;
  rssMb: number;
}

export type StrayKind = 'solto' | 'claude' | 'skye' | 'janela';

export interface Stray {
  kind: StrayKind;
  pid: number;
  name: string;
  args: string;
  cwd: string;
  terminal?: string;
  procs: number;
  rssMb: number;
}

export interface TmuxServer {
  name: string;
  alive: boolean;
}

export interface MonitorReport {
  machine: Machine;
  history: MemoryPoint[] | null;
  selfMb: number;
  terminals: TerminalLoad[] | null;
  strays: Stray[] | null;
  servers: TmuxServer[] | null;
}
