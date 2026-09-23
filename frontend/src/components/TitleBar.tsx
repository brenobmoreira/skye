import { useState } from 'react';
import { api, isWindow, runtime } from '../bridge';
import type { Preset } from '../lib/types';

export function TitleBar(props: {
  presets: Preset[];
  sound: boolean;
  onNew: (preset: string) => void;
  onToggleSound: () => void;
}) {
  const [open, setOpen] = useState(false);
  const pick = (preset: string) => { setOpen(false); props.onNew(preset); };
  const windowed = isWindow();
  return (
    <header className="titlebar" onDoubleClick={windowed ? () => api().ToggleMaximise() : undefined}>
      <span className="brand">skye</span>
      <div className="menu">
        <button onClick={() => setOpen(!open)} title="novo terminal">+</button>
        {open && (
          <div className="menu-list" onMouseLeave={() => setOpen(false)}>
            <button onClick={() => pick('')}>terminal vazio</button>
            {props.presets.map((p) => (
              <button key={p.name} onClick={() => pick(p.name)}>
                {p.name}
                <small>{p.command}</small>
              </button>
            ))}
          </div>
        )}
      </div>
      <span className="spacer" />
      <button onClick={props.onToggleSound} title="latido">{props.sound ? '🔔' : '🔕'}</button>
      {windowed && (
        <>
          <button onClick={() => runtime().WindowMinimise()} title="minimizar">–</button>
          <button onClick={() => api().ToggleMaximise()} title="maximizar">□</button>
          <button onClick={() => api().Hide()} title="esconder (os terminais continuam)">×</button>
        </>
      )}
      <button onClick={() => api().Quit()} title="sair e encerrar todos os terminais">sair</button>
    </header>
  );
}
