import { useEffect, useState, type MouseEvent } from 'react';
import { api, connection, isWindow, mode, runtime } from '../bridge';
import { windowControls } from '../windowControls';
import type { Preset } from '../lib/types';
import { Logo } from './DogIcon';
import { FONTS, type TerminalFont } from '../lib/fonts';

const controls = windowControls(mode, api, runtime, () => connection().kind === 'ready');

export function TitleBar(props: {
  presets: Preset[];
  sound: boolean;
  ready: boolean;
  onNew: (preset: string) => void;
  onToggleSound: () => void;
  font: TerminalFont;
  onPickFont: (font: TerminalFont) => void;
  monitor: boolean;
  onToggleMonitor: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [maximised, setMaximised] = useState(false);
  const [settings, setSettings] = useState(false);
  const pick = (preset: string) => { setOpen(false); props.onNew(preset); };
  const windowed = isWindow();

  useEffect(() => {
    if (!windowed) return;
    const check = () => { controls.isMaximised().then(setMaximised).catch(() => {}); };
    check();
    window.addEventListener('resize', check);
    return () => window.removeEventListener('resize', check);
  }, [windowed]);

  const sound = (
    <button className="flat" disabled={!props.ready} onClick={props.onToggleSound} title="latido">{props.sound ? '🔔' : '🔕'}</button>
  );

  const onDoubleClick = (e: MouseEvent) => {
    if ((e.target as HTMLElement).closest('button, .menu')) return;
    controls.toggleMaximise();
  };

  return (
    <header className={windowed ? 'titlebar windowed' : 'titlebar'} onDoubleClick={windowed ? onDoubleClick : undefined}>
      <Logo />
      <span className="brand">skye</span>
      <div className="menu">
        <button className="flat" disabled={!props.ready} onClick={() => setOpen(!open)} title="novo terminal">+</button>
        {open && props.ready && (
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
      {windowed && sound}
      <span className="spacer" />
      <button className={props.monitor ? 'flat on' : 'flat'} disabled={!props.ready} onClick={props.onToggleMonitor} title="monitor: memória e processos">monitor</button>
      <div className="menu">
        <button className="flat" onClick={() => setSettings(!settings)} title="configurações">⚙</button>
        {settings && (
          <div className="menu-list right" onMouseLeave={() => setSettings(false)}>
            <h4>fonte do terminal</h4>
            {FONTS.map((f) => (
              <button key={f.id} onClick={() => { setSettings(false); props.onPickFont(f); }} style={{ fontFamily: f.family }}>
                {f.id === props.font.id ? '✓ ' : ''}{f.label}
              </button>
            ))}
          </div>
        )}
      </div>
      {!windowed && sound}
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
