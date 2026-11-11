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
  usage?: Usage;
  provider?: string;
  createdAt: string;
  since?: string;
  order: number;
  // How the agent's last turn closed, and whether the list can answer it (see lib/reply).
  reply?: string;
  replyFull?: string;
  question?: boolean;
  options?: Option[];
  turn?: number;
  answered?: string;
}

export interface Option {
  n: number;
  label: string;
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

export type PlaceCommand = 'claude' | 'agy' | 'codex';

export interface Place {
  name: string;
  path: string;
}

export interface Repo {
  name: string;
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
  cacheMb: number;
  largeFree: number;
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
  bgProcs: number;
  bgMb: number;
}

export type StrayKind = 'solto' | 'fundo' | 'claude' | 'skye' | 'janela';

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

export interface HostMemory {
  totalMb: number;
  availMb: number;
  updatedAt: string;
}

export interface MonitorReport {
  host?: HostMemory;
  machine: Machine;
  history: MemoryPoint[] | null;
  hostHistory?: MemoryPoint[] | null;
  selfMb: number;
  terminals: TerminalLoad[] | null;
  strays: Stray[] | null;
  background: Stray[] | null;
  servers: TmuxServer[] | null;
}

export type CompareMode = 'work' | 'branch';

export interface FileChange {
  path: string;
  status: string;
  added: number;
  removed: number;
  binary?: boolean;
}

export interface Changes {
  root: string;
  branch: string;
  base: string;
  files: FileChange[];
}

export interface FileDiff {
  path: string;
  old: string;
  new: string;
  binary?: boolean;
  tooBig?: boolean;
}

export interface DirEntry {
  name: string;
  dir: boolean;
  size: number;
}

export interface FileContent {
  path: string;
  content: string;
  version: string;
  size: number;
  binary?: boolean;
  tooBig?: boolean;
}

export interface Repeat {
  kind: 'daily' | 'weekdays' | 'weekly' | 'monthly';
  weekday?: number;
  day?: number;
}

export interface NoteItem {
  id: string;
  text: string;
  date?: string;
  time?: string;
  repeat?: Repeat;
  created: string;
  doneAt?: string;
}

export interface DoneRecord {
  id: string;
  itemId: string;
  text: string;
  doneAt: string;
  prevDate: string;
}

export interface Notes {
  items: NoteItem[];
  done: DoneRecord[];
}

export interface NoteEdit {
  text: string;
  date: string;
  time: string;
  repeat: Repeat | null;
}

export type PomodoroPhase = 'idle' | 'focus' | 'break' | 'long';

export interface PomodoroSettings {
  focus: number;
  break: number;
  long: number;
  every: number;
}

export interface PomodoroState {
  phase: PomodoroPhase;
  paused: boolean;
  remainingMs: number;
  totalMs: number;
  count: number;
  stage: number;
  settings: PomodoroSettings;
}

export interface FreedCache {
  freedMb: number;
  cacheMb: number;
  largeFree: number;
}
