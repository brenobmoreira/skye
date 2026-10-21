import { useEffect, useRef, useState } from 'react';
import { api } from '../bridge';
import { base } from '../lib/label';
import { changeLabel, crumbs, isMarkdown, joinPath, splitPath } from '../lib/files';
import type { Changes, CompareMode, DirEntry, FileChange, FileContent, FileDiff, Terminal } from '../lib/types';
import { CodeView } from './CodeView';
import { Markdown } from './Markdown';

const MODE_KEY = 'skye:filesMode';

function storedMode(): CompareMode {
  try {
    return localStorage.getItem(MODE_KEY) === 'branch' ? 'branch' : 'work';
  } catch {
    return 'work';
  }
}

const message = (e: unknown) => (e instanceof Error ? e.message : String(e));

type Selection = { path: string; change?: FileChange };

function OpenInEditor({ id, path }: { id: string; path: string }) {
  const [error, setError] = useState('');
  const open = () => { api().OpenInEditor(id, path).then(() => setError('')).catch((e) => setError(message(e))); };
  return (
    <>
      <button className="flat" onClick={open} title={path ? 'abrir este arquivo no VS Code' : 'abrir a pasta no VS Code'}>VS Code</button>
      {error && <span className="files-error" title={error}>{error}</span>}
    </>
  );
}

function Viewer(props: { id: string; sel: Selection; mode: CompareMode; tick: number; onOpen: (path: string) => void; onDirty: (dirty: boolean) => void; onSaved: () => void }) {
  const { id, sel, mode, tick } = props;
  const deleted = sel.change?.status === 'D';
  const [view, setView] = useState<'diff' | 'file'>(sel.change ? 'diff' : 'file');
  const [diff, setDiff] = useState<FileDiff | null>(null);
  const [file, setFile] = useState<FileContent | null>(null);
  const [rendered, setRendered] = useState(true);
  const [editing, setEditing] = useState(false);
  const [error, setError] = useState('');
  const draft = useRef('');
  const editingNow = useRef(false);
  editingNow.current = editing;

  useEffect(() => {
    if (view !== 'diff') return;
    let alive = true;
    api().Diff(id, sel.path, mode).then((d) => { if (alive) { setDiff(d); setError(''); } }).catch((e) => { if (alive) setError(message(e)); });
    return () => { alive = false; };
  }, [id, sel.path, mode, tick, view]);

  useEffect(() => {
    if (view !== 'file' || editingNow.current) return;
    let alive = true;
    api().ReadFile(id, sel.path).then((f) => { if (alive) { setFile(f); setError(''); } }).catch((e) => { if (alive) setError(message(e)); });
    return () => { alive = false; };
  }, [id, sel.path, tick, view]);

  const edit = () => {
    if (!file) return;
    draft.current = file.content;
    setEditing(true);
  };
  const stop = () => {
    setEditing(false);
    props.onDirty(false);
  };
  const save = () => {
    if (!file) return;
    api().WriteFile(id, sel.path, draft.current, file.version)
      .then((f) => { setFile(f); setError(''); stop(); props.onSaved(); })
      .catch((e) => setError(message(e)));
  };
  const cancel = () => {
    if (draft.current !== file?.content && !window.confirm('Descartar a edição não salva?')) return;
    stop();
  };

  const md = isMarkdown(sel.path);
  const unreadable = (f: { binary?: boolean; tooBig?: boolean } | null) =>
    f?.binary ? 'Arquivo binário: abra no VS Code.' : f?.tooBig ? 'Arquivo grande demais para mostrar aqui: abra no VS Code.' : '';

  let body;
  if (view === 'diff') {
    body = !diff ? null : unreadable(diff) ? <div className="empty">{unreadable(diff)}</div> : <CodeView path={sel.path} doc={diff.new} original={diff.old} />;
  } else if (!file) {
    body = null;
  } else if (unreadable(file)) {
    body = <div className="empty">{unreadable(file)}</div>;
  } else if (md && rendered && !editing) {
    body = <Markdown text={file.content} path={sel.path} onOpen={props.onOpen} />;
  } else {
    body = (
      <CodeView
        path={sel.path}
        doc={file.content}
        editable={editing}
        onChange={(doc) => { draft.current = doc; props.onDirty(doc !== file.content); }}
        onSave={save}
      />
    );
  }

  const canEdit = view === 'file' && file && !unreadable(file);
  return (
    <>
      <div className="files-bar">
        <span className="path" title={sel.path}>{sel.path}</span>
        {sel.change && <span className={`status s-${sel.change.status === '?' ? 'u' : sel.change.status}`}>{changeLabel(sel.change.status)}</span>}
        <span className="spacer" />
        {sel.change && !deleted && !editing && (
          <span className="seg">
            <button className={view === 'diff' ? 'flat on' : 'flat'} onClick={() => setView('diff')}>mudanças</button>
            <button className={view === 'file' ? 'flat on' : 'flat'} onClick={() => setView('file')}>arquivo</button>
          </span>
        )}
        {view === 'file' && md && !editing && (
          <button className="flat" onClick={() => setRendered(!rendered)}>{rendered ? 'ver código' : 'ver formatado'}</button>
        )}
        {canEdit && !editing && <button className="flat" onClick={edit} title="editar aqui">editar</button>}
        {editing && <button className="flat on" onClick={save} title="Ctrl+S">salvar</button>}
        {editing && <button className="flat" onClick={cancel}>cancelar</button>}
        {!deleted && <OpenInEditor id={id} path={sel.path} />}
      </div>
      {error && <div className="files-error block">{error}</div>}
      <div className="files-content">{body}</div>
    </>
  );
}

function Panel({ terminal, onBack }: { terminal: Terminal; onBack: () => void }) {
  const id = terminal.id;
  const [tab, setTab] = useState<'changes' | 'tree'>('changes');
  const [mode, setMode] = useState<CompareMode>(storedMode);
  const [changes, setChanges] = useState<Changes | null>(null);
  const [changesError, setChangesError] = useState('');
  const [dir, setDir] = useState('');
  const [entries, setEntries] = useState<DirEntry[]>([]);
  const [selected, setSelected] = useState<Selection | null>(null);
  const [tick, setTick] = useState(0);
  const dirty = useRef(false);

  // Claude finishing a turn or stopping to ask is when the files are worth a new look.
  const first = useRef(true);
  useEffect(() => {
    if (first.current) first.current = false;
    else setTick((n) => n + 1);
  }, [terminal.state]);

  useEffect(() => {
    let alive = true;
    api().Changes(id, mode)
      .then((c) => { if (alive) { setChanges(c); setChangesError(''); } })
      .catch((e) => { if (alive) { setChanges(null); setChangesError(message(e)); } });
    return () => { alive = false; };
  }, [id, mode, tick]);

  useEffect(() => {
    if (tab !== 'tree') return;
    let alive = true;
    api().ListDir(id, dir).then((list) => { if (alive) setEntries(list); }).catch(() => { if (alive) setEntries([]); });
    return () => { alive = false; };
  }, [id, dir, tab, tick]);

  const pick = (sel: Selection) => {
    if (dirty.current && !window.confirm('Descartar a edição não salva?')) return;
    dirty.current = false;
    setSelected(sel);
  };
  const pickMode = (next: CompareMode) => {
    setMode(next);
    try {
      localStorage.setItem(MODE_KEY, next);
    } catch {}
  };
  const openFile = (path: string) => pick({ path, change: changes?.files.find((c) => c.path === path) });
  const refresh = () => setTick((n) => n + 1);

  const files = changes?.files ?? [];
  return (
    <div className="files">
      <header className="files-head">
        <button className="flat" onClick={onBack} title="voltar ao terminal">← terminal</button>
        <span className="where" title={changes?.root ?? terminal.cwd}>
          {base(changes?.root ?? terminal.cwd)}
          {changes?.branch && <span className="branch">{changes.branch}</span>}
        </span>
        <span className="spacer" />
        <button className="flat" onClick={refresh} title="recarregar">↻</button>
        <OpenInEditor id={id} path="" />
      </header>
      <div className="files-body">
        <nav className="files-side">
          <div className="seg tabs">
            <button className={tab === 'changes' ? 'flat on' : 'flat'} onClick={() => setTab('changes')}>mudanças{changes ? ` (${files.length})` : ''}</button>
            <button className={tab === 'tree' ? 'flat on' : 'flat'} onClick={() => setTab('tree')}>arquivos</button>
          </div>
          {tab === 'changes' ? (
            <>
              <div className="seg modes">
                <button className={mode === 'work' ? 'flat on' : 'flat'} onClick={() => pickMode('work')} title="o que ainda não foi commitado">não commitadas</button>
                <button className={mode === 'branch' ? 'flat on' : 'flat'} onClick={() => pickMode('branch')} title="tudo o que a branch mudou desde a base, commitado ou não">
                  {mode === 'branch' && changes ? `desde ${changes.base}` : 'na branch'}
                </button>
              </div>
              {changesError && <div className="none">{changesError}</div>}
              {changes && files.length === 0 && <div className="none">nada mudou</div>}
              {files.map((c) => {
                const { dir: folder, name } = splitPath(c.path);
                return (
                  <div key={c.path} className={selected?.path === c.path ? 'file active' : 'file'} onClick={() => pick({ path: c.path, change: c })} title={`${c.path} · ${changeLabel(c.status)}`}>
                    <span className={`status s-${c.status === '?' ? 'u' : c.status}`}>{c.status === '?' ? 'U' : c.status}</span>
                    <span className="name">{name}</span>
                    <span className="dir">{folder}</span>
                    <span className="count">
                      {c.binary ? 'bin' : (
                        <>
                          {c.added > 0 && <span className="add">+{c.added}</span>}
                          {c.removed > 0 && <span className="del">−{c.removed}</span>}
                        </>
                      )}
                    </span>
                  </div>
                );
              })}
            </>
          ) : (
            <>
              <div className="crumbs">
                <button className="flat" onClick={() => setDir('')}>{base(changes?.root ?? terminal.cwd)}</button>
                {crumbs(dir).map((c) => (
                  <span key={c.path}>/<button className="flat" onClick={() => setDir(c.path)}>{c.name}</button></span>
                ))}
              </div>
              {dir && <div className="file" onClick={() => setDir(splitPath(dir).dir)}><span className="icon">↰</span><span className="name">..</span></div>}
              {entries.map((e) => {
                const path = joinPath(dir, e.name);
                return (
                  <div key={e.name} className={selected?.path === path ? 'file active' : 'file'} onClick={() => (e.dir ? setDir(path) : openFile(path))}>
                    <span className="icon">{e.dir ? '▸' : ''}</span>
                    <span className={e.dir ? 'name folder' : 'name'}>{e.name}</span>
                  </div>
                );
              })}
            </>
          )}
        </nav>
        <section className="files-view">
          {selected ? (
            <Viewer
              key={`${id}:${selected.path}:${selected.change ? 'c' : 'f'}`}
              id={id}
              sel={selected}
              mode={mode}
              tick={tick}
              onOpen={openFile}
              onDirty={(d) => { dirty.current = d; }}
              onSaved={refresh}
            />
          ) : (
            <div className="empty">Escolha um arquivo.</div>
          )}
        </section>
      </div>
    </div>
  );
}

// What changed in a terminal's folder and its files, next to where claude works on them.
export function FilesPanel({ terminal, onBack }: { terminal: Terminal | undefined; onBack: () => void }) {
  if (!terminal) return <div className="files"><div className="empty">Escolha um terminal na lista.</div></div>;
  return <Panel key={terminal.id} terminal={terminal} onBack={onBack} />;
}
