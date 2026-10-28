import { useState, type KeyboardEvent } from 'react';
import { FONTS, type TerminalFont } from '../lib/fonts';

type Section = 'shortcuts' | 'font' | 'settings';

const SECTIONS: { id: Section; label: string }[] = [
  { id: 'shortcuts', label: 'Atalhos' },
  { id: 'font', label: 'Fonte' },
  { id: 'settings', label: 'Configurações' },
];

const SHORTCUTS: { keys: string; does: string }[] = [
  { keys: 'Ctrl+1 … Ctrl+9', does: 'abre o 1º … 9º terminal da lista' },
  { keys: 'Ctrl+Shift+Espaço', does: 'pula para o próximo que espera você (sem nenhum, o próximo não lido)' },
  { keys: 'Ctrl+= / Ctrl+-', does: 'aumenta / diminui a fonte (também Ctrl+roda do mouse)' },
  { keys: 'Ctrl+0', does: 'volta a fonte ao tamanho padrão' },
  { keys: 'Ctrl+clique na lista', does: 'abre o terminal ao lado do atual' },
  { keys: 'Clique do meio na lista', does: 'fecha o terminal (pergunta antes se estiver em andamento)' },
  { keys: 'Clique direito na lista', does: 'marca o terminal como não lido (ou lido)' },
  { keys: 'Alt+Enter / Ctrl+Enter', does: 'quebra linha no claude' },
  { keys: 'Ctrl+Shift+V', does: 'cola no terminal' },
  { keys: 'Ctrl+clique num link', does: 'abre no navegador padrão do Windows' },
];

// The ⚙ menu: one row per section, each opening its panel beside the menu.
export function SettingsMenu(props: {
  font: TerminalFont;
  customFonts: TerminalFont[];
  onPickFont: (font: TerminalFont) => void;
  onAddFont: (name: string) => void;
  onRemoveFont: (font: TerminalFont) => void;
}) {
  const [open, setOpen] = useState(false);
  const [section, setSection] = useState<Section | null>(null);
  const [adding, setAdding] = useState(false);
  const [name, setName] = useState('');
  const [armed, setArmed] = useState<string | null>(null);

  const close = () => {
    setOpen(false);
    setSection(null);
    setAdding(false);
    setName('');
    setArmed(null);
  };

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
      <button className="flat" onClick={() => (open ? close() : setOpen(true))} title="configurações">⚙</button>
      {open && (
        <div className="menu-list right settings-menu" onMouseLeave={() => { if (!adding) close(); }}>
          {SECTIONS.map((s) => (
            <button key={s.id} className={section === s.id ? 'section on' : 'section'} onClick={() => setSection(section === s.id ? null : s.id)}>
              {s.label}<span>›</span>
            </button>
          ))}
          {section === 'shortcuts' && (
            <div className="menu-list settings-panel">
              <h4>atalhos</h4>
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
              <h4>fonte do terminal</h4>
              {FONTS.map((f) => <div key={f.id}>{fontButton(f)}</div>)}
              {props.customFonts.map((f) => (
                <div key={f.id} className="place" onMouseLeave={() => { if (armed === f.id) setArmed(null); }}>
                  {fontButton(f)}
                  <button className={armed === f.id ? 'remove armed' : 'remove'} onClick={() => remove(f)} title="remover esta fonte">
                    {armed === f.id ? 'remover?' : '×'}
                  </button>
                </div>
              ))}
              <hr />
              {adding ? (
                <div className="form">
                  <input autoFocus placeholder="nome da fonte (ex.: Fira Code)" value={name} onChange={(e) => setName(e.target.value)} onKeyDown={keys} />
                  <small>precisa estar instalada; sem ela, o terminal usa monospace</small>
                  <div className="actions">
                    <button onClick={saveFont}>salvar</button>
                    <button onClick={() => { setAdding(false); setName(''); }}>cancelar</button>
                  </div>
                </div>
              ) : (
                <button className="add" onClick={() => setAdding(true)}>+ Adicionar</button>
              )}
            </div>
          )}
          {section === 'settings' && (
            <div className="menu-list settings-panel">
              <h4>configurações</h4>
              <small className="empty">nada por aqui ainda</small>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
