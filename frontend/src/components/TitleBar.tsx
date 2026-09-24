import { useEffect, useState, type MouseEvent } from 'react';
import { api, isWindow, mode, runtime } from '../bridge';
import { windowControls } from '../windowControls';
import type { Preset } from '../lib/types';

const controls = windowControls(mode, api, runtime);

export function TitleBar(props: {
  presets: Preset[];
  sound: boolean;
  onNew: (preset: string) => void;
  onToggleSound: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [maximised, setMaximised] = useState(false);
  const pick = (preset: string) => { setOpen(false); props.onNew(preset); };
  const windowed = isWindow();

  useEffect(() => {
    if (!windowed) return;
    const check = () => { controls.isMaximised().then(setMaximised).catch(() => {}); };
    check();
    window.addEventListener('resize', check);
    return () => window.removeEventListener('resize', check);
  }, [windowed]);

  const onDoubleClick = (e: MouseEvent) => {
    if ((e.target as HTMLElement).closest('button, .menu')) return;
    controls.toggleMaximise();
  };

  return (
    <header className={windowed ? 'titlebar windowed' : 'titlebar'} onDoubleClick={windowed ? onDoubleClick : undefined}>
      <span className="brand">skye</span>
      <div className="menu">
        <button className="flat" onClick={() => setOpen(!open)} title="novo terminal">+</button>
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
      <button className="flat" onClick={props.onToggleSound} title="latido">{props.sound ? '🔔' : '🔕'}</button>
      <span className="spacer" />
      <button className="flat quit" onClick={() => { controls.quitAll().catch(() => {}); }} title="sair e encerrar todos os terminais">sair</button>
      {windowed && (
        <div className="window-controls">
          <button onClick={controls.minimise} title="minimizar">—</button>
          <button onClick={controls.toggleMaximise} title={maximised ? 'restaurar' : 'maximizar'}>{maximised ? '❐' : '☐'}</button>
          <button className="close" onClick={controls.close} title={mode === 'windows-app' ? 'fechar a janela (os terminais continuam)' : 'esconder (os terminais continuam)'}>✕</button>
        </div>
      )}
    </header>
  );
}
