import type { Notes, PomodoroState } from '../lib/types';
import { NoteList } from './NoteList';
import { PomodoroCard } from './PomodoroCard';

// The right column: the pomodoro and the notes, each on its own island, like the terminal list and
// the plan usage on the left.
export function FocusPanel(props: { notes: Notes; pomodoro: PomodoroState | null; remainingMs: number; now: Date }) {
  return (
    <aside className="focus">
      {props.pomodoro && (
        <div className="island pomodoro-island">
          <PomodoroCard state={props.pomodoro} remainingMs={props.remainingMs} />
        </div>
      )}
      <div className="island notes-island">
        <NoteList notes={props.notes} now={props.now} />
      </div>
    </aside>
  );
}
