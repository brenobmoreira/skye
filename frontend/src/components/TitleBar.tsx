import { useEffect, useState, type MouseEvent } from 'react';
import { api, connection, isWindow, mode, runtime } from '../bridge';
import { windowControls } from '../windowControls';
import type { Place, Preset, Repo } from '../lib/types';
import { NewMenu } from './NewMenu';
import { Logo } from './DogIcon';
import { FONTS, type TerminalFont } from '../lib/fonts';

const controls = windowControls(mode, api, runtime, () => connection().kind === 'ready');

export function TitleBar(props: {
  presets: Preset[];
  repos: Repo[];
  places: Place[];
  onNewWorktree: (repo: string, branch: string) => Promise<void>;
  onOpenPlace: (name: string) => void;
  onAddPlace: (name: string, path: string) => Promise<void>;
  onRemovePlace: (name: string) => Promise<void>;
  sound: boolean;
  ready: boolean;
  onNew: (preset: string) => void;
  onToggleSound: () => void;
  font: TerminalFont;
  onPickFont: (font: TerminalFont) => void;
  monitor: boolean;
  onToggleMonitor: () => void;
}) {
  const [maximised, setMaximised] = useState(false);
  const [settings, setSettings] = useState(false);
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
      <NewMenu
        ready={props.ready}
        places={props.places}
        presets={props.presets}
        repos={props.repos}
        onNew={props.onNew}
        onOpenPlace={props.onOpenPlace}
        onAddPlace={props.onAddPlace}
        onRemovePlace={props.onRemovePlace}
        onNewWorktree={props.onNewWorktree}
      />
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
