import { useEffect, useState } from 'react';
import { api } from '../bridge';
import { breakLabel, clock, collapsedLabel, parseClock, phaseLabel } from '../lib/pomodoro';
import { EyeIcon } from './icons';
import type { PomodoroPhase, PomodoroState } from '../lib/types';

const R = 54;
const C = 2 * Math.PI * R;

type Action = 'start' | 'pause' | 'resume' | 'skip' | 'reset';

// The timer at the top of the focus panel: a ring that fills as the phase runs, the time left and
// where in the cycle it is.
export function PomodoroCard(props: {
  state: PomodoroState;
  remainingMs: number;
  collapsed: boolean;
  onToggleCollapsed: () => void;
  ended: PomodoroPhase | null;
  onAck: () => void;
}) {
  const { state, remainingMs } = props;
  const act = (a: Action) => { api().PomodoroAction(a).catch(() => {}); };
  const running = state.phase !== 'idle';
  // Idle, the clock shows what the next start runs: a length typed in, or the focus setting.
  const total = running ? state.totalMs : state.nextMs || state.settings.focus * 60_000;
  const left = running ? remainingMs : total;
  const done = total > 0 ? Math.min(1, Math.max(0, 1 - left / total)) : 0;
  const every = state.settings.every;
  // The stage the next start uses: the next focus of the cycle, unless another one is picked.
  const [stage, setStage] = useState(Math.min(state.stage + 1, every));
  useEffect(() => setStage(Math.min(state.stage + 1, every)), [state.stage, every]);
  const begin = (phase: 'focus' | 'break') => { api().PomodoroBegin(phase, stage).catch(() => {}); };
  // A double click on the clock edits the time left: "18" or "18:30", Enter sets it, Esc gives up.
  const [typing, setTyping] = useState<string | null>(null);
  const [typeError, setTypeError] = useState(false);
  const setTime = () => {
    const seconds = typing === null ? null : parseClock(typing);
    if (seconds === null) { setTypeError(true); return; }
    api().PomodoroSetRemaining(seconds).then(() => { setTyping(null); setTypeError(false); }).catch(() => setTypeError(true));
  };

  const eye = (
    <button className="eye-toggle tip-end" onClick={props.onToggleCollapsed} aria-label={props.collapsed ? 'Mostrar o tempo' : 'Esconder o tempo'} data-tip={props.collapsed ? 'Mostrar o tempo' : 'Esconder o tempo'}>
      <EyeIcon shut={props.collapsed} />
    </button>
  );

  // Folded: one line with the phase and its status, pulsing when a phase ran out until clicked.
  if (props.collapsed) {
    return (
      <section className={`pomodoro-line ${state.phase}${props.ended ? ' ended' : ''}`}>
        <button className="line-main" onClick={props.ended ? props.onAck : props.onToggleCollapsed} title={props.ended ? 'Ok' : 'Mostrar o tempo'}>
          <span className="dot" />
          {collapsedLabel(state, props.ended)}
        </button>
        {eye}
      </section>
    );
  }

  return (
    <section className={`pomodoro ${state.phase}${state.paused ? ' paused' : ''}`}>
      {eye}
      <div className="ring">
        <svg viewBox="0 0 128 128" aria-hidden="true">
          <defs>
            <linearGradient id="pomodoro-fill" x1="0" x2="1">
              <stop offset="0" stopColor="#7aa2f7" />
              <stop offset="1" stopColor="#bb9af7" />
            </linearGradient>
          </defs>
          <circle className="track" cx="64" cy="64" r={R} />
          <circle className="fill" cx="64" cy="64" r={R} strokeDasharray={C} strokeDashoffset={C * (1 - done)} />
        </svg>
        {typing === null ? (
          <div className="time" onDoubleClick={() => { setTyping(clock(left)); setTypeError(false); }} title="Clique duas vezes para mudar o tempo">{clock(left)}</div>
        ) : (
          <input
            className={typeError ? 'time time-input wrong' : 'time time-input'}
            autoFocus
            value={typing}
            onFocus={(e) => e.currentTarget.select()}
            onChange={(e) => { setTyping(e.target.value); setTypeError(false); }}
            onKeyDown={(e) => {
              if (e.key === 'Enter') setTime();
              if (e.key === 'Escape') { setTyping(null); setTypeError(false); }
            }}
            onBlur={() => { setTyping(null); setTypeError(false); }}
            aria-label="Tempo restante (minutos, ou minutos:segundos)"
          />
        )}
      </div>
      <div className="phase">{phaseLabel(state)}{state.paused ? ' · pausado' : ''}</div>
      {!running && (
        <div className="stages" role="radiogroup" aria-label="Estágio do ciclo">
          {Array.from({ length: every }, (_, i) => i + 1).map((n) => (
            <button
              key={n}
              role="radio"
              aria-checked={n === stage}
              className={`stage${n < stage ? ' done' : ''}${n === stage ? ' current' : ''}`}
              onClick={() => setStage(n)}
              data-tip={`Estágio ${n}`}
            >
              {n}
            </button>
          ))}
        </div>
      )}
      <div className="controls">
        {!running && <button className="primary" onClick={() => begin('focus')}>Começar foco</button>}
        {!running && <button onClick={() => begin('break')}>{breakLabel(stage, every)}</button>}
        {running && !state.paused && <button onClick={() => act('pause')}>Pausar</button>}
        {running && state.paused && <button className="primary" onClick={() => act('resume')}>Retomar</button>}
        {running && <button onClick={() => act('skip')} title="Encerra esta fase agora">Pular</button>}
        {(running || state.count > 0) && <button className="quiet" onClick={() => act('reset')} title="Para e zera a contagem de hoje">Zerar</button>}
      </div>
    </section>
  );
}
