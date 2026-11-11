import { useEffect, useState, type MouseEvent } from 'react';
import { api, connection, isWindow, mode, runtime } from '../bridge';
import { windowControls } from '../windowControls';
import type { Place, PlaceCommand, PomodoroSettings, Preset, Repo } from '../lib/types';
import { NewMenu } from './NewMenu';
import { Logo } from './DogIcon';
import { SettingsMenu } from './SettingsMenu';
import { ConfirmDialog } from './ConfirmDialog';
import type { TerminalFont } from '../lib/fonts';

const controls = windowControls(mode, api, runtime, () => connection().kind === 'ready');

export function TitleBar(props: {
  presets: Preset[];
  repos: Repo[];
  places: Place[];
  onNewWorktree: (repo: string, branch: string) => Promise<void>;
  onOpenPlace: (name: string, command: PlaceCommand) => void;
  onAddPlace: (name: string, path: string) => Promise<void>;
  onRemovePlace: (name: string) => Promise<void>;
  sound: boolean;
  ready: boolean;
  onNew: (preset: string) => void;
  onToggleSound: () => void;
  font: TerminalFont;
  customFonts: TerminalFont[];
  onPickFont: (font: TerminalFont) => void;
  onAddFont: (name: string) => void;
  onRemoveFont: (font: TerminalFont) => void;
  monitor: boolean;
  onToggleMonitor: () => void;
  files: boolean;
  canFiles: boolean;
  onToggleFiles: () => void;
  focus: boolean;
  onToggleFocus: () => void;
  // The phase and time left, shown while the focus panel is closed; empty when the timer is idle.
  pill: string;
  pomodoro: PomodoroSettings | null;
  onSavePomodoro: (s: PomodoroSettings) => Promise<void>;
}) {
  const [maximised, setMaximised] = useState(false);
  const [quitting, setQuitting] = useState(false);
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
    if ((e.target as HTMLElement).closest('button, .menu, .dialog-backdrop')) return;
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
      {props.pill && !props.focus && (
        <button className="pill" onClick={props.onToggleFocus} title="abrir o painel de foco">{props.pill}</button>
      )}
      <button className={props.focus ? 'flat on' : 'flat'} disabled={!props.ready} onClick={props.onToggleFocus} title="foco: pomodoro e anotações">foco</button>
      <button className={props.files ? 'flat on' : 'flat'} disabled={!props.ready || (!props.canFiles && !props.files)} onClick={props.onToggleFiles} title="arquivos do terminal: o que mudou, leitura e edição">arquivos</button>
      <button className={props.monitor ? 'flat on' : 'flat'} disabled={!props.ready} onClick={props.onToggleMonitor} title="monitor: memória e processos">monitor</button>
      <SettingsMenu
        font={props.font}
        customFonts={props.customFonts}
        onPickFont={props.onPickFont}
        onAddFont={props.onAddFont}
        onRemoveFont={props.onRemoveFont}
        pomodoro={props.pomodoro}
        onSavePomodoro={props.onSavePomodoro}
      />
      {!windowed && sound}
      <button className="flat quit" onClick={() => setQuitting(true)} title="sair e encerrar todos os terminais">sair</button>
      {windowed && (
        <div className="window-controls">
          <button onClick={controls.minimise} title="minimizar">—</button>
          <button onClick={controls.toggleMaximise} title={maximised ? 'restaurar' : 'maximizar'}>{maximised ? '❐' : '☐'}</button>
          <button className="close" onClick={controls.close} title={mode === 'windows-app' ? 'fechar a janela (os terminais continuam)' : 'esconder (os terminais continuam)'}>✕</button>
        </div>
      )}
      {quitting && (
        <ConfirmDialog
          title="Sair da skye?"
          message="Tem certeza que deseja sair? Todos os terminais abertos serão encerrados."
          confirm="Sair"
          onConfirm={() => { setQuitting(false); controls.quitAll().catch(() => {}); }}
          onCancel={() => setQuitting(false)}
        />
      )}
    </header>
  );
}
