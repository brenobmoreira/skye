import { api } from '../bridge';
import { clock, phaseLabel } from '../lib/pomodoro';
import type { PomodoroState } from '../lib/types';

const R = 54;
const C = 2 * Math.PI * R;

type Action = 'start' | 'pause' | 'resume' | 'skip' | 'reset';

// The timer at the top of the focus panel: a ring that fills as the phase runs, the time left and
// where in the cycle it is.
export function PomodoroCard({ state, remainingMs }: { state: PomodoroState; remainingMs: number }) {
  const act = (a: Action) => { api().PomodoroAction(a).catch(() => {}); };
  const running = state.phase !== 'idle';
  const total = running ? state.totalMs : state.settings.focus * 60_000;
  const left = running ? remainingMs : total;
  const done = total > 0 ? Math.min(1, Math.max(0, 1 - left / total)) : 0;

  return (
    <section className={`pomodoro ${state.phase}${state.paused ? ' paused' : ''}`}>
      <div className="ring">
        <svg viewBox="0 0 128 128" aria-hidden="true">
          <circle className="track" cx="64" cy="64" r={R} />
          <circle className="fill" cx="64" cy="64" r={R} strokeDasharray={C} strokeDashoffset={C * (1 - done)} />
        </svg>
        <div className="time">{clock(left)}</div>
      </div>
      <div className="phase">{phaseLabel(state)}{state.paused ? ' · pausado' : ''}</div>
      <div className="controls">
        {!running && <button className="primary" onClick={() => act('start')}>começar</button>}
        {running && !state.paused && <button onClick={() => act('pause')}>pausar</button>}
        {running && state.paused && <button className="primary" onClick={() => act('resume')}>retomar</button>}
        {running && <button onClick={() => act('skip')} title="encerra esta fase agora">pular</button>}
        {(running || state.count > 0) && <button className="quiet" onClick={() => act('reset')} title="para e zera a contagem de hoje">zerar</button>}
      </div>
    </section>
  );
}
