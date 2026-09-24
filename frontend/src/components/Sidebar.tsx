import { useState } from 'react';
import { api } from '../bridge';
import { base, itemLabel } from '../lib/label';
import { stateLabel } from '../lib/states';
import type { Conversation, Terminal } from '../lib/types';

function TerminalItem({ t, active, unread, onSelect }: { t: Terminal; active: boolean; unread: boolean; onSelect: () => void }) {
  const label = itemLabel(t);
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(label.name);
  const confirm = () => {
    setEditing(false);
    const next = name.trim();
    if (next && next !== label.name) api().Rename(t.id, next);
  };
  return (
    <div className={`item${active ? ' active' : ''}${unread ? ' unread' : ''}`} onClick={onSelect} title={stateLabel[t.state]}>
      <span className="dot" style={{ background: `var(--${t.state})` }} />
      <div>
        <div className="row">
          {editing ? (
            <input
              autoFocus
              value={name}
              onChange={(e) => setName(e.target.value)}
              onBlur={confirm}
              onKeyDown={(e) => {
                if (e.key === 'Enter') confirm();
                if (e.key === 'Escape') setEditing(false);
              }}
            />
          ) : (
            <div className="name" onDoubleClick={() => { setName(label.name); setEditing(true); }}>{label.name}</div>
          )}
          {unread && <span className="badge" title="não lido" />}
          <button className="x" title="fechar terminal" onClick={(e) => { e.stopPropagation(); api().Close(t.id); }}>×</button>
        </div>
        <div className="sub">{label.sub}</div>
      </div>
    </div>
  );
}

export function Sidebar(props: {
  terminals: Terminal[];
  conversations: Conversation[];
  activeId: string | null;
  unread: Set<string>;
  onSelect: (id: string) => void;
  onResume: (sessionId: string) => void;
}) {
  return (
    <aside className="sidebar">
      <h3>Terminais</h3>
      {props.terminals.map((t) => (
        <TerminalItem key={t.id} t={t} active={t.id === props.activeId} unread={props.unread.has(t.id)} onSelect={() => props.onSelect(t.id)} />
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
