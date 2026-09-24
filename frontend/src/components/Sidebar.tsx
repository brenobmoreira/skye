import { useState } from 'react';
import { api } from '../bridge';
import { base, itemLabel } from '../lib/label';
import { moveWithinGroup } from '../lib/order';
import { DogIcon } from './DogIcon';
import { stateLabel } from '../lib/states';
import type { Conversation, Terminal, Usage } from '../lib/types';
import { usagePanel } from '../lib/usage';

const ENDED_KEY = 'skye:endedOpen';

function storedEndedOpen(): boolean {
  try {
    return localStorage.getItem(ENDED_KEY) === 'true';
  } catch {
    return false;
  }
}

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
      title={t.state === 'waiting' && t.ask ? `${stateLabel[t.state]}: ${t.ask}` : stateLabel[t.state]}
      draggable={!editing}
      onDragStart={(e) => { e.dataTransfer.effectAllowed = 'move'; drag.start(t.id); }}
      onDragOver={(e) => { if (drag.over(t.id)) e.preventDefault(); }}
      onDrop={(e) => { e.preventDefault(); drag.drop(t.id); }}
      onDragEnd={drag.end}
    >
      <DogIcon color={`var(--${t.state})`} title={stateLabel[t.state]} />
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

// Plan usage from the claude status line, pinned under the list like the status line under the
// prompt.
function UsagePanel({ usage }: { usage: Usage | null }) {
  const panel = usagePanel(usage, Date.now());
  if (!panel) return null;
  return (
    <div className={panel.stale ? 'usage stale' : 'usage'} title={`atualizado às ${panel.updated}`}>
      {panel.rows.map((r) => (
        <div className="usage-row" key={r.label}>
          <span className="label">{r.label}</span>
          <span className="bar"><span style={{ width: `${r.pct}%` }} /></span>
          <span className="pct">{r.pct}%</span>
          <span className="resets">{r.resets}</span>
        </div>
      ))}
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
  usage: Usage | null;
}) {
  const [dragging, setDragging] = useState<string | null>(null);
  const [endedOpen, setEndedOpen] = useState(storedEndedOpen);
  const toggleEnded = () => {
    setEndedOpen(!endedOpen);
    try {
      localStorage.setItem(ENDED_KEY, String(!endedOpen));
    } catch {}
  };
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
      <div className="list">
      <h3>Terminais</h3>
      {props.terminals.map((t) => (
        <TerminalItem key={t.id} t={t} active={t.id === props.activeId} unread={props.unread.has(t.id)} onSelect={() => props.onSelect(t.id)} drag={drag} />
      ))}
      {props.conversations.length > 0 && (
        <h3 className="toggle" onClick={toggleEnded} title={endedOpen ? 'recolher' : 'mostrar'}>
          {endedOpen ? '▾' : '▸'} Encerradas ({props.conversations.length})
        </h3>
      )}
      {endedOpen && props.conversations.map((c) => (
        <div className="item" key={c.sessionId} onClick={() => props.onResume(c.sessionId)} title="retomar">
          <DogIcon color="var(--shell)" title="encerrada" />
          <div>
            <div className="row">
              <div className="name">{c.title}</div>
              <button className="x" title="esquecer" onClick={(e) => { e.stopPropagation(); api().Forget(c.sessionId); }}>×</button>
            </div>
            <div className="sub">{base(c.cwd)} · retomar</div>
          </div>
        </div>
      ))}
      </div>
      <UsagePanel usage={props.usage} />
    </aside>
  );
}
