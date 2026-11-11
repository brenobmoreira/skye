import { useState, type KeyboardEvent } from 'react';
import { FONTS, type TerminalFont } from '../lib/fonts';
import type { PomodoroSettings } from '../lib/types';
import { GearIcon } from './icons';

type Section = 'shortcuts' | 'font' | 'focus' | 'update' | 'settings';

const SECTIONS: { id: Section; label: string }[] = [
  { id: 'shortcuts', label: 'Atalhos' },
  { id: 'font', label: 'Fonte' },
  { id: 'focus', label: 'Foco' },
  { id: 'update', label: 'Atualizar' },
  { id: 'settings', label: 'Configurações' },
];

const SHORTCUTS: { keys: string; does: string }[] = [
  { keys: 'Ctrl+1 … Ctrl+9', does: 'Abre o 1º … 9º terminal da lista' },
  { keys: 'Ctrl+Shift+Espaço', does: 'Pula para o próximo que espera você (sem nenhum, o próximo não lido)' },
  { keys: 'Ctrl+= / Ctrl+-', does: 'Aumenta / diminui a fonte (também Ctrl+roda do mouse)' },
  { keys: 'Ctrl+0', does: 'Volta a fonte ao tamanho padrão' },
  { keys: 'Ctrl+clique na lista', does: 'Abre o terminal ao lado do atual' },
  { keys: 'Clique do meio na lista', does: 'Fecha o terminal (pergunta antes se estiver em andamento)' },
  { keys: 'Clique direito na lista', does: 'Marca o terminal como não lido (ou lido)' },
  { keys: 'Alt+Enter / Ctrl+Enter', does: 'Quebra linha no claude' },
  { keys: 'Ctrl+Shift+V', does: 'Cola no terminal' },
  { keys: 'Ctrl+clique num link', does: 'Abre no navegador padrão do Windows' },
];

// The ⚙ menu: one row per section, each opening its panel beside the menu.
export function SettingsMenu(props: {
  font: TerminalFont;
  customFonts: TerminalFont[];
  onPickFont: (font: TerminalFont) => void;
  onAddFont: (name: string) => void;
  onRemoveFont: (font: TerminalFont) => void;
  pomodoro: PomodoroSettings | null;
  onSavePomodoro: (s: PomodoroSettings) => Promise<void>;
  onUpdate: () => Promise<void>;
}) {
  const [open, setOpen] = useState(false);
  const [section, setSection] = useState<Section | null>(null);
  const [adding, setAdding] = useState(false);
  const [name, setName] = useState('');
  const [armed, setArmed] = useState<string | null>(null);
  const [times, setTimes] = useState<PomodoroSettings | null>(null);
  const [timesError, setTimesError] = useState('');
  // The update's progress: idle, building, restarting (the window reloads by itself), or an error.
  const [updating, setUpdating] = useState<'idle' | 'building' | 'restarting'>('idle');
  const [updateError, setUpdateError] = useState('');
  const runUpdate = () => {
    setUpdating('building');
    setUpdateError('');
    props.onUpdate().then(
      () => setUpdating('restarting'),
      (e) => { setUpdating('idle'); setUpdateError(e instanceof Error ? e.message : String(e)); },
    );
  };

  const close = () => {
    setOpen(false);
    setSection(null);
    setAdding(false);
    setName('');
    setArmed(null);
    setTimes(null);
    setTimesError('');
  };

  const openSection = (next: Section) => {
    setSection(section === next ? null : next);
    if (next === 'focus') { setTimes(props.pomodoro); setTimesError(''); }
  };

  const saveTimes = () => {
    if (!times) return;
    props.onSavePomodoro(times).then(close).catch((e) => setTimesError(e instanceof Error ? e.message : String(e)));
  };

  const timeField = (key: keyof PomodoroSettings, label: string, unit: string) => times && (
    <label className="time-field">
      <span>{label}</span>
      <input type="number" min={1} value={times[key]} onChange={(e) => setTimes({ ...times, [key]: Number(e.target.value) })} onKeyDown={(e) => { if (e.key === 'Enter') saveTimes(); }} />
      <small>{unit}</small>
    </label>
  );

  const saveFont = () => {
    if (!name.trim()) return;
    props.onAddFont(name);
    setAdding(false);
    setName('');
  };

  const keys = (e: KeyboardEvent) => {
    if (e.key === 'Enter') saveFont();
    if (e.key === 'Escape') { setAdding(false); setName(''); }
  };

  const remove = (f: TerminalFont) => {
    if (armed !== f.id) {
      setArmed(f.id);
      return;
    }
    setArmed(null);
    props.onRemoveFont(f);
  };

  const fontButton = (f: TerminalFont) => (
    <button onClick={() => { close(); props.onPickFont(f); }} style={{ fontFamily: f.family }}>
      {f.id === props.font.id ? '✓ ' : ''}{f.label}
    </button>
  );

  return (
    <div className="menu">
      <button className={open ? 'icon gear on' : 'icon gear'} onClick={() => (open ? close() : setOpen(true))} aria-label="Configurações" data-tip={open ? undefined : 'Configurações'}>
        <GearIcon />
      </button>
      {open && (
        <div className="menu-list right settings-menu" onMouseLeave={() => { if (!adding && section !== 'focus' && section !== 'update') close(); }}>
          {SECTIONS.map((s) => (
            <button key={s.id} className={section === s.id ? 'section on' : 'section'} onClick={() => openSection(s.id)}>
              <span>‹</span>{s.label}
            </button>
          ))}
          {section === 'shortcuts' && (
            <div className="menu-list settings-panel">
              <h4>Atalhos</h4>
              <dl className="shortcuts">
                {SHORTCUTS.map((s) => (
                  <div key={s.keys}>
                    <dt><kbd>{s.keys}</kbd></dt>
                    <dd>{s.does}</dd>
                  </div>
                ))}
              </dl>
            </div>
          )}
          {section === 'font' && (
            <div className="menu-list settings-panel">
              <h4>Fonte do terminal</h4>
              {FONTS.map((f) => <div key={f.id}>{fontButton(f)}</div>)}
              {props.customFonts.map((f) => (
                <div key={f.id} className="place" onMouseLeave={() => { if (armed === f.id) setArmed(null); }}>
                  {fontButton(f)}
                  <button className={armed === f.id ? 'remove armed' : 'remove'} onClick={() => remove(f)} title="Remover esta fonte">
                    {armed === f.id ? 'remover?' : '×'}
                  </button>
                </div>
              ))}
              <hr />
              {adding ? (
                <div className="form">
                  <input autoFocus placeholder="Nome da fonte (ex.: Fira Code)" value={name} onChange={(e) => setName(e.target.value)} onKeyDown={keys} />
                  <small>Precisa estar instalada; sem ela, o terminal usa monospace</small>
                  <div className="actions">
                    <button onClick={saveFont}>Salvar</button>
                    <button onClick={() => { setAdding(false); setName(''); }}>Cancelar</button>
                  </div>
                </div>
              ) : (
                <button className="add" onClick={() => setAdding(true)}>+ Adicionar</button>
              )}
            </div>
          )}
          {section === 'focus' && (
            <div className="menu-list settings-panel">
              <h4>Pomodoro</h4>
              {times ? (
                <div className="form">
                  {timeField('focus', 'foco', 'min')}
                  {timeField('break', 'pausa', 'min')}
                  {timeField('long', 'pausa longa', 'min')}
                  {timeField('every', 'pausa longa a cada', 'focos')}
                  {timesError && <div className="error">{timesError}</div>}
                  <div className="actions">
                    <button onClick={saveTimes}>Salvar</button>
                    <button onClick={() => setTimes({ focus: 25, break: 5, long: 15, every: 4 })} title="25 / 5 / 15 / 4">Padrão</button>
                    <button onClick={close}>Fechar</button>
                  </div>
                  <small>Vale a partir da próxima fase</small>
                </div>
              ) : (
                <small className="empty">O timer ainda não respondeu</small>
              )}
            </div>
          )}
          {section === 'update' && (
            <div className="menu-list settings-panel update-panel">
              <h4>Atualizar a skye</h4>
              <p>Recompila a partir do repositório, instala e reinicia o servidor. Os terminais continuam abertos e a janela recarrega sozinha.</p>
              {updating === 'idle' && <button className="add" onClick={runUpdate}>Atualizar agora</button>}
              {updating === 'building' && <p className="busy">Compilando… pode levar um minuto.</p>}
              {updating === 'restarting' && <p className="busy">Reiniciando; a janela recarrega em instantes.</p>}
              {updateError && <div className="error">{updateError}</div>}
            </div>
          )}
          {section === 'settings' && (
            <div className="menu-list settings-panel">
              <h4>Configurações</h4>
              <small className="empty">Nada por aqui ainda</small>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
