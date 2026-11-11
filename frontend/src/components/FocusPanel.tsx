import type { Notes, PomodoroPhase, PomodoroState } from '../lib/types';
import { NoteList } from './NoteList';
import { PomodoroCard } from './PomodoroCard';

// The right column: the pomodoro and the notes, each on its own island, like the terminal list and
// the plan usage on the left.
export function FocusPanel(props: {
  notes: Notes;
  pomodoro: PomodoroState | null;
  remainingMs: number;
  now: Date;
  collapsed: boolean;
  onToggleCollapsed: () => void;
  ended: PomodoroPhase | null;
  onAck: () => void;
}) {
  return (
    <aside className="focus">
      {props.pomodoro && (
        <div className={props.collapsed ? 'island pomodoro-island collapsed' : 'island pomodoro-island'}>
          <PomodoroCard state={props.pomodoro} remainingMs={props.remainingMs} collapsed={props.collapsed} onToggleCollapsed={props.onToggleCollapsed} ended={props.ended} onAck={props.onAck} />
        </div>
      )}
      <div className="island notes-island">
        <NoteList notes={props.notes} now={props.now} />
      </div>
    </aside>
  );
}
