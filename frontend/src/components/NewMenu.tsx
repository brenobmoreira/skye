import { useState, type KeyboardEvent } from 'react';
import type { Place, Preset, Repo } from '../lib/types';

const message = (e: unknown) => String((e as Error)?.message ?? e);

// The + menu: saved paths (claude in that folder), a new terminal, presets and worktrees from the
// config, and a form to save another path.
export function NewMenu(props: {
  ready: boolean;
  places: Place[];
  presets: Preset[];
  repos: Repo[];
  onNew: (preset: string) => void;
  onOpenPlace: (name: string) => void;
  onAddPlace: (name: string, path: string) => Promise<void>;
  onRemovePlace: (name: string) => Promise<void>;
  onNewWorktree: (repo: string, branch: string) => Promise<void>;
}) {
  const [open, setOpen] = useState(false);
  const [picking, setPicking] = useState<string | null>(null);
  const [branch, setBranch] = useState('');
  const [adding, setAdding] = useState(false);
  const [name, setName] = useState('');
  const [path, setPath] = useState('');
  const [armed, setArmed] = useState<string | null>(null);
  const [status, setStatus] = useState('');
  const busy = status === 'criando…' || status === 'salvando…';

  const close = () => {
    setOpen(false);
    setPicking(null);
    setBranch('');
    setAdding(false);
    setName('');
    setPath('');
    setArmed(null);
    setStatus('');
  };
  const run = (action: () => void) => { close(); action(); };

  const createWorktree = async () => {
    if (!picking || !branch.trim() || busy) return;
    setStatus('criando…');
    try {
      await props.onNewWorktree(picking, branch.trim());
      close();
    } catch (e) {
      setStatus(message(e));
    }
  };

  const savePlace = async () => {
    if (busy) return;
    if (!name.trim() || !path.trim()) {
      setStatus('preencha nome e path');
      return;
    }
    setStatus('salvando…');
    try {
      await props.onAddPlace(name.trim(), path.trim());
      setAdding(false);
      setName('');
      setPath('');
      setStatus('');
    } catch (e) {
      setStatus(message(e));
    }
  };

  const remove = async (place: string) => {
    if (armed !== place) {
      setArmed(place);
      return;
    }
    setArmed(null);
    try {
      await props.onRemovePlace(place);
    } catch (e) {
      setStatus(message(e));
    }
  };

  const keys = (submit: () => void) => (e: KeyboardEvent) => {
    if (e.key === 'Enter') submit();
    if (e.key === 'Escape') close();
  };

  return (
    <div className="menu">
      <button className="flat" disabled={!props.ready} onClick={() => (open ? close() : setOpen(true))} title="novo terminal">+</button>
      {open && props.ready && (
        <div className="menu-list new-menu" onMouseLeave={() => { if (!picking && !adding) close(); }}>
          {props.places.map((p) => (
            <div key={p.name} className="place" onMouseLeave={() => { if (armed === p.name) setArmed(null); }}>
              <button onClick={() => run(() => props.onOpenPlace(p.name))} title={`claude em ${p.path}`}>
                {p.name}
              </button>
              <button className={armed === p.name ? 'remove armed' : 'remove'} onClick={() => remove(p.name)} title="remover este path">
                {armed === p.name ? 'remover?' : '×'}
              </button>
            </div>
          ))}
          {props.places.length > 0 && <hr />}
          <button onClick={() => run(() => props.onNew(''))}>Novo terminal</button>
          {props.presets.map((p) => (
            <button key={p.name} onClick={() => run(() => props.onNew(p.name))}>
              {p.name}
              <small>{p.command}</small>
            </button>
          ))}
          {props.repos.map((r) =>
            picking === r.name ? (
              <div key={r.name} className="form">
                <small>nova worktree de {r.name}</small>
                <input
                  autoFocus
                  placeholder="nome da branch"
                  value={branch}
                  onChange={(e) => { setBranch(e.target.value); if (!busy) setStatus(''); }}
                  onKeyDown={keys(createWorktree)}
                />
                {status && <small className={busy ? '' : 'error'}>{status}</small>}
              </div>
            ) : (
              <button key={r.name} onClick={() => { setPicking(r.name); setAdding(false); setStatus(''); }}>
                nova worktree de {r.name}…
              </button>
            ),
          )}
          <hr />
          {adding ? (
            <div className="form">
              <input autoFocus placeholder="nome (ex.: LivIA)" value={name} onChange={(e) => setName(e.target.value)} onKeyDown={keys(savePlace)} />
              <input placeholder="path (ex.: ~/projects/ai_livia_copilot)" value={path} onChange={(e) => setPath(e.target.value)} onKeyDown={keys(savePlace)} />
              <div className="actions">
                <button onClick={savePlace} disabled={busy}>salvar</button>
                <button onClick={close}>cancelar</button>
              </div>
              {status && <small className={busy ? '' : 'error'}>{status}</small>}
            </div>
          ) : (
            <button className="add" onClick={() => { setAdding(true); setPicking(null); setStatus(''); }}>+ Adicionar path</button>
          )}
          {!adding && !picking && status && <small className="error">{status}</small>}
        </div>
      )}
    </div>
  );
}
