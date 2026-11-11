import type { Notes, PomodoroState } from '../lib/types';
import { NoteList } from './NoteList';
import { PomodoroCard } from './PomodoroCard';

// The island on the right: the pomodoro on top, the notes under it.
export function FocusPanel(props: { notes: Notes; pomodoro: PomodoroState | null; remainingMs: number; now: Date }) {
  return (
    <aside className="focus island">
      {props.pomodoro && <PomodoroCard state={props.pomodoro} remainingMs={props.remainingMs} />}
      <NoteList notes={props.notes} now={props.now} />
    </aside>
  );
}
