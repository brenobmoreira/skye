import { useState } from 'react';
import { api } from '../bridge';
import { base, itemLabel } from '../lib/label';
import { moveWithinGroup } from '../lib/order';
import { stateLabel } from '../lib/states';
import type { Conversation, Terminal } from '../lib/types';

type Drag = { dragging: string | null; start: (id: string) => void; over: (id: string) => boolean; drop: (id: string) => void; end: () => void };

function TerminalItem({ t, active, unread, onSelect, drag }: { t: Terminal; active: boolean; unread: boolean; onSelect: () => void; drag: Drag }) {
  const label = itemLabel(t);
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(label.name);
  const confirm = () => {
    setEditing(false);
    const next = name.trim();
    if (next && next !== label.name) api().Rename(t.id, next);
  };
  return (
    <div
      className={`item${active ? ' active' : ''}${unread ? ' unread' : ''}${drag.dragging === t.id ? ' dragging' : ''}`}
      onClick={onSelect}
      title={stateLabel[t.state]}
      draggable={!editing}
      onDragStart={(e) => { e.dataTransfer.effectAllowed = 'move'; drag.start(t.id); }}
      onDragOver={(e) => { if (drag.over(t.id)) e.preventDefault(); }}
      onDrop={(e) => { e.preventDefault(); drag.drop(t.id); }}
      onDragEnd={drag.end}
    >
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
  onReorder: (list: Terminal[]) => void;
  onResume: (sessionId: string) => void;
}) {
  const [dragging, setDragging] = useState<string | null>(null);
  const drag: Drag = {
    dragging,
    start: setDragging,
    over: (id) => dragging !== null && moveWithinGroup(props.terminals, dragging, id) !== null,
    drop: (id) => {
      const next = dragging && moveWithinGroup(props.terminals, dragging, id);
      setDragging(null);
      if (next) props.onReorder(next);
    },
    end: () => setDragging(null),
  };
  return (
    <aside className="sidebar">
      <h3>Terminais</h3>
      {props.terminals.map((t) => (
        <TerminalItem key={t.id} t={t} active={t.id === props.activeId} unread={props.unread.has(t.id)} onSelect={() => props.onSelect(t.id)} drag={drag} />
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
