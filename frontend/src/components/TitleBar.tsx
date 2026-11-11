import { useEffect, useState, type MouseEvent, type ReactElement } from 'react';
import { api, connection, isWindow, mode, runtime } from '../bridge';
import { windowControls } from '../windowControls';
import type { Place, PlaceCommand, PomodoroSettings, Preset, Repo } from '../lib/types';
import { NewMenu } from './NewMenu';
import { Logo } from './DogIcon';
import { SettingsMenu } from './SettingsMenu';
import { ConfirmDialog } from './ConfirmDialog';
import type { TerminalFont } from '../lib/fonts';
import { BellIcon, EyeIcon, FolderIcon, PanelIcon, PowerIcon, PulseIcon, TerminalIcon } from './icons';

export type View = 'terminals' | 'monitor' | 'files';

const VIEWS: { id: View; tip: string; icon: () => ReactElement }[] = [
  { id: 'terminals', tip: 'Terminais', icon: TerminalIcon },
  { id: 'monitor', tip: 'Monitor', icon: PulseIcon },
  { id: 'files', tip: 'Arquivos', icon: FolderIcon },
];

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
  view: View;
  onView: (v: View) => void;
  canFiles: boolean;
  focus: boolean;
  onToggleFocus: () => void;
  // The running phase and its time left, for the pill shown while the focus panel is closed.
  pill: { phase: string; time: string } | null;
  pillHidden: boolean;
  onTogglePillHidden: () => void;
  // Set when a phase ran out on its own: the pill pulses with it until clicked.
  ended: string;
  onPill: () => void;
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

  const at = VIEWS.findIndex((v) => v.id === props.view);
  const showPill = !props.focus && (props.ended !== '' || props.pill !== null);

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
      <div className="views" role="tablist">
        <span className="views-ind" style={{ transform: `translateX(${at * 32}px)` }} />
        {VIEWS.map(({ id, tip, icon: Icon }) => (
          <button
            key={id}
            role="tab"
            aria-selected={props.view === id}
            aria-label={tip}
            data-tip={tip}
            className={props.view === id ? 'on' : ''}
            disabled={!props.ready || (id === 'files' && !props.canFiles && props.view !== 'files')}
            onClick={() => props.onView(id)}
          >
            <Icon />
          </button>
        ))}
      </div>
      <span className="spacer" />
      {showPill && (
        <div className={`pill${props.ended ? ' ended' : ''}${props.pillHidden && !props.ended ? ' hidden' : ''}`}>
          <button className="pill-main" onClick={props.onPill} title={props.ended ? 'Ok' : 'Abrir o painel de foco'}>
            <span className="dot" />
            {props.ended || props.pill?.phase}
            {!props.ended && <span className="left">{props.pill?.time}</span>}
          </button>
          {!props.ended && (
            <button className="eye" onClick={props.onTogglePillHidden} aria-label={props.pillHidden ? 'Mostrar o tempo' : 'Esconder o tempo'} data-tip={props.pillHidden ? 'Mostrar o tempo' : 'Esconder o tempo'}>
              <EyeIcon shut={props.pillHidden} />
            </button>
          )}
        </div>
      )}
      <div className="icons">
        <button className={props.focus ? 'icon on' : 'icon'} disabled={!props.ready} onClick={props.onToggleFocus} aria-label="Painel" data-tip="Painel">
          <PanelIcon open={props.focus} />
        </button>
        <button className="icon bell" disabled={!props.ready} onClick={props.onToggleSound} aria-label={props.sound ? 'Latido ligado' : 'Latido desligado'} data-tip={props.sound ? 'Latido ligado' : 'Latido desligado'}>
          <BellIcon muted={!props.sound} />
        </button>
        <SettingsMenu
        font={props.font}
        customFonts={props.customFonts}
        onPickFont={props.onPickFont}
        onAddFont={props.onAddFont}
        onRemoveFont={props.onRemoveFont}
        pomodoro={props.pomodoro}
        onSavePomodoro={props.onSavePomodoro}
      />
        <button className="icon quit tip-end" onClick={() => setQuitting(true)} aria-label="Sair" data-tip="Sair e encerrar todos os terminais">
          <PowerIcon />
        </button>
      </div>
      {windowed && (
        <div className="window-controls">
          <button onClick={controls.minimise} title="Minimizar">—</button>
          <button onClick={controls.toggleMaximise} title={maximised ? 'Restaurar' : 'Maximizar'}>{maximised ? '❐' : '☐'}</button>
          <button className="close" onClick={controls.close} title={mode === 'windows-app' ? 'Fechar a janela (os terminais continuam)' : 'Esconder (os terminais continuam)'}>✕</button>
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
