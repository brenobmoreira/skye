import { useEffect, useState } from 'react';
import { api } from '../bridge';
import { base, itemLabel } from '../lib/label';
import { filterConversations } from '../lib/filter';
import { elapsed } from '../lib/duration';
import { contextBar, contextText } from '../lib/context';
import { moveWithinGroup } from '../lib/order';
import { DogIcon } from './DogIcon';
import { stateLabel } from '../lib/states';
import { asksBeforeClose } from '../lib/close';
import { ConfirmDialog } from './ConfirmDialog';
import { ContextMenu } from './ContextMenu';
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

// How long a closing item takes to fade out of the list; the terminal is closed after it.
const CLOSE_MS = 180;

const calm = () => typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;

// A quick squeeze on the × so the click shows even when a question comes up instead.
function press(el: HTMLElement) {
  if (!calm()) el.animate([{ transform: 'scale(1)' }, { transform: 'scale(0.65)' }, { transform: 'scale(1)' }], { duration: 160, easing: 'ease-out' });
}

type Drag = { dragging: string | null; start: (id: string) => void; over: (id: string) => boolean; drop: (id: string) => void; end: () => void };

function TerminalItem({ t, active, shown, unread, closing, now, onSelect, onClose, onMenu, drag }: { t: Terminal; active: boolean; shown: boolean; unread: boolean; closing: boolean; now: number; onSelect: (side: boolean) => void; onClose: () => void; onMenu: (x: number, y: number) => void; drag: Drag }) {
  const label = itemLabel(t);
  const time = elapsed(t.since, t.state, now);
  const bar = contextBar(t.context);
  const hover = [t.state === 'waiting' && t.ask ? `${stateLabel[t.state]}: ${t.ask}` : stateLabel[t.state], contextText(t.context)]
    .filter(Boolean)
    .join('\n');
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(label.name);
  const confirm = () => {
    setEditing(false);
    const next = name.trim();
    if (next && next !== label.name) api().Rename(t.id, next);
  };
  return (
    <div
      className={`item${active ? ' active' : shown ? ' shown' : ''}${unread ? ' unread' : ''}${drag.dragging === t.id ? ' dragging' : ''}${closing ? ' closing' : ''}`}
      onClick={(e) => onSelect(e.ctrlKey || e.metaKey)}
      onMouseDown={(e) => { if (e.button === 1) e.preventDefault(); }}
      onAuxClick={(e) => { if (e.button === 1) { e.preventDefault(); onClose(); } }}
      onContextMenu={(e) => { e.preventDefault(); onMenu(e.clientX, e.clientY); }}
      title={hover}
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
          <button className="x" title="fechar terminal (ou clique do meio)" onClick={(e) => { e.stopPropagation(); press(e.currentTarget); onClose(); }}>×</button>
        </div>
        <div className="sub">
          <span>{label.sub}</span>
          {time && <span className="time">{time}</span>}
        </div>
      </div>
      {bar && <span className={bar.full ? 'ctx full' : 'ctx'}><span style={{ width: `${bar.pct}%` }} /></span>}
    </div>
  );
}

// Plan usage from the claude status line, pinned under the list like the status line under the
// prompt.
function UsagePanel({ usage }: { usage: Usage | null }) {
  const panel = usagePanel(usage, Date.now());
  if (!panel) return null;
  return (
    <div className={panel.stale ? 'usage island stale' : 'usage island'} title={`atualizado às ${panel.updated}`}>
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
  shown: string[];
  onSelect: (id: string, side: boolean) => void;
  onReorder: (list: Terminal[]) => void;
  onResume: (sessionId: string) => void;
  onMark: (id: string, unread: boolean) => void;
  usage: Usage | null;
}) {
  const [dragging, setDragging] = useState<string | null>(null);
  const [asking, setAsking] = useState<string | null>(null);
  const [closing, setClosing] = useState<Set<string>>(() => new Set());
  const [menu, setMenu] = useState<{ id: string; x: number; y: number } | null>(null);
  const [endedOpen, setEndedOpen] = useState(storedEndedOpen);
  const [query, setQuery] = useState('');
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    const tick = setInterval(() => setNow(Date.now()), 30_000);
    return () => clearInterval(tick);
  }, []);
  const ended = filterConversations(props.conversations, query);
  const toggleEnded = () => {
    setEndedOpen(!endedOpen);
    try {
      localStorage.setItem(ENDED_KEY, String(!endedOpen));
    } catch {}
  };
  const close = (id: string) => {
    setClosing((s) => new Set(s).add(id));
    const undo = () => setClosing((s) => { const next = new Set(s); next.delete(id); return next; });
    setTimeout(() => api().Close(id).catch(undo), calm() ? 0 : CLOSE_MS);
  };
  const requestClose = (t: Terminal) => (asksBeforeClose(t.state) ? setAsking(t.id) : close(t.id));
  const asked = props.terminals.find((t) => t.id === asking);
  const menuFor = menu && props.terminals.find((t) => t.id === menu.id);
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
      <div className="list island">
      <h3>Terminais</h3>
      {props.terminals.map((t) => (
        <TerminalItem key={t.id} t={t} active={t.id === props.activeId} shown={props.shown.includes(t.id)} unread={props.unread.has(t.id)} closing={closing.has(t.id)} now={now} onSelect={(side) => props.onSelect(t.id, side)} onClose={() => requestClose(t)} onMenu={(x, y) => setMenu({ id: t.id, x, y })} drag={drag} />
      ))}
      {props.conversations.length > 0 && (
        <h3 className="toggle" onClick={toggleEnded} title={endedOpen ? 'recolher' : 'mostrar'}>
          {endedOpen ? '▾' : '▸'} Encerradas ({props.conversations.length})
        </h3>
      )}
      {endedOpen && (
        <input
          className="filter"
          placeholder="filtrar por título ou pasta"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Escape') setQuery(''); }}
        />
      )}
      {endedOpen && ended.length === 0 && <div className="none">nenhuma encontrada</div>}
      {endedOpen && ended.map((c) => (
        <div className="item" key={c.sessionId} onClick={() => props.onResume(c.sessionId)} title="retomar">
          <DogIcon color="var(--shell)" title="encerrada" />
          <div>
            <div className="row">
              <div className="name">{c.title}</div>
              <button className="x" title="esquecer" onClick={(e) => { e.stopPropagation(); api().Forget(c.sessionId); }}>×</button>
            </div>
            <div className="sub"><span>{base(c.cwd)} · retomar</span></div>
          </div>
        </div>
      ))}
      </div>
      <UsagePanel usage={props.usage} />
      {menu && menuFor && (
        <ContextMenu
          x={menu.x}
          y={menu.y}
          items={[props.unread.has(menuFor.id)
            ? { label: 'marcar como lido', run: () => props.onMark(menuFor.id, false) }
            : { label: 'marcar como não lido', run: () => props.onMark(menuFor.id, true) }]}
          onClose={() => setMenu(null)}
        />
      )}
      {asked && (
        <ConfirmDialog
          title={`${itemLabel(asked).name} está em andamento`}
          message={`Ele está ${stateLabel[asked.state]}. Fechar mesmo assim? O que estiver rodando será interrompido.`}
          confirm="Fechar"
          onConfirm={() => { setAsking(null); close(asked.id); }}
          onCancel={() => setAsking(null)}
        />
      )}
    </aside>
  );
}
