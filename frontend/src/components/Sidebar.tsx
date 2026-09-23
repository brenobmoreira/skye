import { useState } from 'react';
import { api } from '../bridge';
import { stateLabel } from '../lib/states';
import type { Conversation, Terminal } from '../lib/types';

const base = (path: string) => path.split('/').filter(Boolean).pop() ?? path;

function TerminalItem({ t, active, onSelect }: { t: Terminal; active: boolean; onSelect: () => void }) {
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(t.name);
  const confirm = () => {
    setEditing(false);
    if (name.trim() && name.trim() !== t.name) api().Rename(t.id, name.trim());
    else setName(t.name);
  };
  return (
    <div className={`item${active ? ' active' : ''}`} onClick={onSelect} title={stateLabel[t.state]}>
      <span className="dot" style={{ background: `var(--${t.state})` }} />
      <div>
        {editing ? (
          <input
            autoFocus
            value={name}
            onChange={(e) => setName(e.target.value)}
            onBlur={confirm}
            onKeyDown={(e) => {
              if (e.key === 'Enter') confirm();
              if (e.key === 'Escape') { setName(t.name); setEditing(false); }
            }}
          />
        ) : (
          <div className="name" onDoubleClick={() => { setName(t.name); setEditing(true); }}>{t.name}</div>
        )}
        <div className="sub">{t.title || base(t.cwd) || stateLabel[t.state]}</div>
      </div>
      <button className="x" title="fechar terminal" onClick={(e) => { e.stopPropagation(); api().Close(t.id); }}>×</button>
    </div>
  );
}

export function Sidebar(props: {
  terminals: Terminal[];
  conversations: Conversation[];
  activeId: string | null;
  onSelect: (id: string) => void;
  onResume: (sessionId: string) => void;
}) {
  return (
    <aside className="sidebar">
      <h3>Terminais</h3>
      {props.terminals.map((t) => (
        <TerminalItem key={t.id} t={t} active={t.id === props.activeId} onSelect={() => props.onSelect(t.id)} />
      ))}
      {props.conversations.length > 0 && <h3>Encerradas</h3>}
      {props.conversations.map((c) => (
        <div className="item" key={c.sessionId} onClick={() => props.onResume(c.sessionId)} title="retomar">
          <span className="dot" style={{ background: 'var(--shell)' }} />
          <div>
            <div className="name">{c.title}</div>
            <div className="sub">{base(c.cwd)} · retomar</div>
          </div>
          <button className="x" title="esquecer" onClick={(e) => { e.stopPropagation(); api().Forget(c.sessionId); }}>×</button>
        </div>
      ))}
    </aside>
  );
}
