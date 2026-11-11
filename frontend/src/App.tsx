import { lazy, Suspense, useEffect, useRef, useState } from 'react';
import { api, connection, isWindow, on, onConnection, retryConnection } from './bridge';
import { bark } from './sound';
import { TitleBar } from './components/TitleBar';
import { Sidebar } from './components/Sidebar';
import { TerminalView } from './components/TerminalView';
import { Monitor } from './components/Monitor';
import { FocusPanel } from './components/FocusPanel';
import { clock, endedLabel, pillPhase } from './lib/pomodoro';
import type { Conversation, Notes, Place, PlaceCommand, PomodoroPhase, PomodoroSettings, PomodoroState, Preset, Repo, State, Terminal, Usage } from './lib/types';
import { nextUnread, withMarks } from './lib/unread';
import { shortcut } from './lib/shortcuts';
import { active, closeSplit, prune, select, selectSide, slotOf, visible, type Panes } from './lib/panes';
import type { ConnectionState } from './windowsTransport';
import { applyZoom, parseFontSize, zoomKey, type ZoomAction } from './lib/zoom';
import { customFont, DEFAULT_FONT, FONTS, parseCustomFonts, parseFont, type TerminalFont } from './lib/fonts';

// The editor behind the files panel is large; it loads the first time the panel opens.
const FilesPanel = lazy(() => import('./components/FilesPanel').then((m) => ({ default: m.FilesPanel })));

const FONT_KEY = 'skye:fontSize';
const FAMILY_KEY = 'skye:font';
const CUSTOM_FONTS_KEY = 'skye:customFonts';
const FOCUS_KEY = 'skye:focusOpen';
const PILL_HIDDEN_KEY = 'skye:pillHidden';

function storedFlag(key: string): boolean {
  try {
    return localStorage.getItem(key) === '1';
  } catch {
    return false;
  }
}

function storeFlag(key: string, on: boolean) {
  try {
    localStorage.setItem(key, on ? '1' : '0');
  } catch {}
}

const storedFocusOpen = () => storedFlag(FOCUS_KEY);

function storedFontSize(): number {
  try {
    return parseFontSize(localStorage.getItem(FONT_KEY));
  } catch {
    return parseFontSize(null);
  }
}

function ConnectionScreen({ state }: { state: ConnectionState }) {
  if (state.kind === 'error') {
    return (
      <div className="connection error">
        <h2>Não consegui abrir a skye</h2>
        <pre>{state.message}</pre>
        <button onClick={retryConnection}>Tentar de novo</button>
      </div>
    );
  }
  return <div className="connection">Iniciando a skye no WSL…</div>;
}

function storedCustomFonts(): TerminalFont[] {
  try {
    return parseCustomFonts(localStorage.getItem(CUSTOM_FONTS_KEY));
  } catch {
    return [];
  }
}

function storedFont(): TerminalFont {
  try {
    return parseFont(localStorage.getItem(FAMILY_KEY), storedCustomFonts());
  } catch {
    return parseFont(null);
  }
}

const windowFocused = () => document.visibilityState === 'visible' && document.hasFocus();

// Two terminals side by side need room for both next to the sidebar, and next to the focus panel
// when it is open; narrower, only the focused one shows.
const SPLIT_QUERY = '(min-width: 1100px)';
const SPLIT_QUERY_WITH_FOCUS = '(min-width: 1360px)';
const splitQuery = (focus: boolean) => (focus ? SPLIT_QUERY_WITH_FOCUS : SPLIT_QUERY);
const splitRoom = (focus: boolean) => typeof window.matchMedia === 'function' && window.matchMedia(splitQuery(focus)).matches;
const NO_NOTES: Notes = { items: [], done: [] };

const byCreation = (a: Terminal, b: Terminal) =>
  Date.parse(a.createdAt) - Date.parse(b.createdAt) || (a.id < b.id ? -1 : a.id > b.id ? 1 : 0);

export function App() {
  const [terminals, setTerminals] = useState<Terminal[]>([]);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [presets, setPresets] = useState<Preset[]>([]);
  const [repos, setRepos] = useState<Repo[]>([]);
  const [places, setPlaces] = useState<Place[]>([]);
  const [panes, setPanes] = useState<Panes>({ left: null, right: null, focus: 'left' });
  const [focusOpen, setFocusOpen] = useState(storedFocusOpen);
  const [room, setRoom] = useState(() => splitRoom(storedFocusOpen()));
  const activeId = active(panes);
  const [sound, setSound] = useState(true);
  const [problems, setProblems] = useState<string[]>([]);
  const [fontSize, setFontSize] = useState(storedFontSize);
  const [font, setFont] = useState(storedFont);
  const [customFonts, setCustomFonts] = useState(storedCustomFonts);
  const [epoch, setEpoch] = useState(0);
  const [link, setLink] = useState(connection);
  const [focused, setFocused] = useState(windowFocused);
  const [usage, setUsage] = useState<Usage | null>(null);
  const [notes, setNotes] = useState<Notes>(NO_NOTES);
  // The timer as the Go side last sent it, and when it arrived: the time left is counted from there.
  const [pomodoro, setPomodoroState] = useState<{ state: PomodoroState; at: number } | null>(null);
  const [now, setNow] = useState(() => Date.now());
  const [pillHidden, setPillHidden] = useState(() => storedFlag(PILL_HIDDEN_KEY));
  // The phase that last ran out on its own, until the pill is clicked or the panel opened.
  const [ended, setEnded] = useState('');
  const receivePomodoro = (state: PomodoroState) => { setPomodoroState({ state, at: Date.now() }); setNow(Date.now()); };
  const [unread, setUnread] = useState<Set<string>>(() => new Set());
  // Marked unread from the right-click menu; only picking the terminal clears these.
  const [marked, setMarked] = useState<Set<string>>(() => new Set());
  const shownUnread = withMarks(unread, marked, terminals);
  // What the main area shows instead of the terminals: the monitor or the active terminal's files.
  const [view, setView] = useState<'terminals' | 'monitor' | 'files'>('terminals');
  const monitor = view === 'monitor';
  const covered = view !== 'terminals';
  const lastStates = useRef<Record<string, State>>({});
  const unmark = (id: string) =>
    setMarked((m) => {
      if (!m.has(id)) return m;
      const next = new Set(m);
      next.delete(id);
      return next;
    });
  const mark = (id: string, on: boolean) => {
    if (on) {
      setMarked((m) => new Set(m).add(id));
      return;
    }
    unmark(id);
    setUnread((u) => {
      if (!u.has(id)) return u;
      const next = new Set(u);
      next.delete(id);
      return next;
    });
  };
  const jump = useRef({ terminals, activeId, unread });
  jump.current = { terminals, activeId, unread: shownUnread };

  const load = () => {
    api().List().then(setTerminals).catch(() => {});
    api().Conversations().then(setConversations).catch(() => {});
    api().Presets().then(setPresets).catch(() => {});
    api().Repos().then(setRepos).catch(() => {});
    api().Places().then(setPlaces).catch(() => {});
    api().Sound().then(setSound).catch(() => {});
    api().Problems().then(setProblems).catch(() => {});
    api().Usage().then(setUsage).catch(() => {});
    api().Notes().then(setNotes).catch(() => {});
    api().Pomodoro().then(receivePomodoro).catch(() => {});
  };

  useEffect(() => {
    let previous = connection().kind;
    const off = onConnection((state) => {
      if (previous === 'error' && state.kind !== 'error') load();
      previous = state.kind;
      setLink(state);
    });
    setLink(connection());
    return off;
  }, []);

  useEffect(() => {
    const zoom = (action: ZoomAction) =>
      setFontSize((size) => {
        const next = applyZoom(size, action);
        try {
          localStorage.setItem(FONT_KEY, String(next));
        } catch {}
        return next;
      });
    const onKey = (e: KeyboardEvent) => {
      const { terminals, activeId, unread } = jump.current;
      const target = shortcut(e, terminals, activeId, unread);
      if (target) {
        e.preventDefault();
        e.stopPropagation();
        unmark(target);
        setPanes((p) => select(p, target));
        setView('terminals');
        return;
      }
      const action = zoomKey(e);
      if (!action) return;
      e.preventDefault();
      e.stopPropagation();
      zoom(action);
    };
    const onWheel = (e: WheelEvent) => {
      if (!e.ctrlKey || e.deltaY === 0) return;
      e.preventDefault();
      zoom(e.deltaY < 0 ? 'in' : 'out');
    };
    window.addEventListener('keydown', onKey, true);
    window.addEventListener('wheel', onWheel, { passive: false, capture: true });
    return () => {
      window.removeEventListener('keydown', onKey, true);
      window.removeEventListener('wheel', onWheel, true);
    };
  }, []);

  useEffect(() => {
    const report = () => {
      setFocused(windowFocused());
      api().SetFocused(windowFocused()).catch(() => {});
    };
    load();
    const offs = [
      on('terminals', (list: Terminal[]) => setTerminals(list)),
      on('conversations', (list: Conversation[]) => setConversations(list)),
      on('bark', () => { bark(); }),
      on('problems', (list: string[]) => setProblems(list)),
      on('usage', (u: Usage) => setUsage(u)),
      on('places', (list: Place[]) => setPlaces(list)),
      on('notes', (n: Notes) => setNotes(n)),
      on('pomodoro', (st: PomodoroState) => receivePomodoro(st)),
      on('pomodoroEnded', (phase: PomodoroPhase) => setEnded(endedLabel(phase))),
      on('reconnected', () => {
        load();
        report();
        setEpoch((n) => n + 1);
      }),
    ];
    const focus = () => { setFocused(true); api().SetFocused(true).catch(() => {}); };
    const blur = () => { setFocused(false); api().SetFocused(false).catch(() => {}); };
    window.addEventListener('focus', focus);
    window.addEventListener('blur', blur);
    const web = !isWindow();
    if (web) document.addEventListener('visibilitychange', report);
    api().SetFocused(document.hasFocus()).catch(() => {});
    return () => {
      offs.forEach((off) => off());
      window.removeEventListener('focus', focus);
      window.removeEventListener('blur', blur);
      if (web) document.removeEventListener('visibilitychange', report);
    };
  }, []);

  useEffect(() => {
    setPanes((p) => prune(p, terminals.map((t) => t.id)));
  }, [terminals]);

  // With the panel open the end of a phase is already in view; nothing is left to acknowledge.
  useEffect(() => {
    if (focusOpen && ended) setEnded('');
  }, [focusOpen, ended]);

  useEffect(() => {
    if (typeof window.matchMedia !== 'function') return;
    const query = window.matchMedia(splitQuery(focusOpen));
    const change = () => setRoom(query.matches);
    change();
    query.addEventListener('change', change);
    return () => query.removeEventListener('change', change);
  }, [focusOpen]);

  // The countdown ticks every second while a phase runs; otherwise a slower tick is enough to move
  // notes into Atrasadas when their time comes.
  const ticking = pomodoro !== null && pomodoro.state.phase !== 'idle' && !pomodoro.state.paused;
  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), ticking ? 1000 : 30_000);
    return () => window.clearInterval(id);
  }, [ticking]);
  const remainingMs = pomodoro === null ? 0 : ticking ? Math.max(0, pomodoro.state.remainingMs - (now - pomodoro.at)) : pomodoro.state.remainingMs;

  useEffect(() => {
    const prev = lastStates.current;
    const seen = focused && !covered ? visible(panes, room) : [];
    setUnread((current) => nextUnread(prev, terminals, current, seen));
    lastStates.current = Object.fromEntries(terminals.map((t) => [t.id, t.state]));
  }, [terminals, panes, room, focused, covered]);

  const views = [...terminals].sort(byCreation);

  const show = (id: string) => { unmark(id); setPanes((p) => select(p, id)); setView('terminals'); };
  // With the files open, picking another terminal shows its files.
  const pickFromList = (id: string, side: boolean) => {
    unmark(id);
    setPanes((p) => (side ? selectSide(p, id) : select(p, id)));
    setView((v) => (v === 'files' ? v : 'terminals'));
  };
  const toggle = (next: 'monitor' | 'files') => setView((v) => (v === next ? 'terminals' : next));
  const split = !covered && panes.right !== null && room;
  const open = async (preset: string) => show((await api().NewTerminal(preset)).id);
  const openPlace = async (name: string, command: PlaceCommand) => show((await api().OpenPlace(name, command)).id);
  const addPlace = (name: string, path: string) => api().AddPlace(name, path);
  const removePlace = (name: string) => api().RemovePlace(name);
  const newWorktree = async (repo: string, branch: string) => show((await api().NewWorktree(repo, branch)).id);
  const resume = async (sessionId: string) => show((await api().Resume(sessionId)).id);
  const reorder = (list: Terminal[]) => {
    setTerminals(list);
    api().Reorder(list.map((t) => t.id)).catch(() => {});
  };
  const toggleFocus = () =>
    setFocusOpen((open) => {
      storeFlag(FOCUS_KEY, !open);
      if (!open) setEnded('');
      return !open;
    });
  const togglePillHidden = () =>
    setPillHidden((hidden) => {
      storeFlag(PILL_HIDDEN_KEY, !hidden);
      return !hidden;
    });
  // A pulsing pill is only acknowledged by the click; otherwise the pill opens the panel.
  const clickPill = () => (ended ? setEnded('') : toggleFocus());
  const pickView = (next: 'terminals' | 'monitor' | 'files') => setView(next);
  const running = pomodoro !== null && pomodoro.state.phase !== 'idle';

  const savePomodoro = (s: PomodoroSettings) => api().SetPomodoroSettings(s);
  const toggleSound = () => { api().SetSound(!sound); setSound(!sound); };
  const pickFont = (next: TerminalFont) => {
    setFont(next);
    try {
      localStorage.setItem(FAMILY_KEY, next.id);
    } catch {}
  };
  const saveCustomFonts = (list: TerminalFont[]) => {
    setCustomFonts(list);
    try {
      localStorage.setItem(CUSTOM_FONTS_KEY, JSON.stringify(list.map((f) => f.label)));
    } catch {}
  };
  const addFont = (name: string) => {
    const list = parseCustomFonts(JSON.stringify([...customFonts.map((f) => f.label), name]));
    saveCustomFonts(list);
    const added = customFont(name);
    pickFont(FONTS.find((f) => f.label === added.label) ?? parseFont(added.id, list));
  };
  const removeFont = (gone: TerminalFont) => {
    saveCustomFonts(customFonts.filter((f) => f.id !== gone.id));
    if (font.id === gone.id) pickFont(DEFAULT_FONT);
  };

  return (
    <div className={isWindow() ? 'app windowed' : 'app'}>
      {link.kind !== 'ready' && <ConnectionScreen state={link} />}
      <TitleBar presets={presets} repos={repos} places={places} onOpenPlace={openPlace} onAddPlace={addPlace} onRemovePlace={removePlace} onNewWorktree={newWorktree} sound={sound} ready={link.kind === 'ready'} onNew={open} onToggleSound={toggleSound} font={font} customFonts={customFonts} onPickFont={pickFont} onAddFont={addFont} onRemoveFont={removeFont} view={view} onView={pickView} canFiles={activeId !== null} focus={focusOpen} onToggleFocus={toggleFocus} pill={running && pomodoro ? { phase: pillPhase(pomodoro.state), time: clock(remainingMs) } : null} pillHidden={pillHidden} onTogglePillHidden={togglePillHidden} ended={ended} onPill={clickPill} pomodoro={pomodoro?.state.settings ?? null} onSavePomodoro={savePomodoro} />
      <div className={focusOpen ? 'body with-focus' : 'body'}>
        <Sidebar terminals={terminals} conversations={conversations} activeId={activeId} unread={shownUnread} shown={visible(panes, room)} onSelect={pickFromList} onReorder={reorder} onResume={resume} onMark={mark} usage={usage} />
        <main className="main island">
          {monitor && <Monitor terminals={terminals} onOpen={show} />}
          {view === 'files' && (
            <Suspense fallback={<div className="empty">Abrindo…</div>}>
              <FilesPanel terminal={terminals.find((t) => t.id === activeId)} onBack={() => setView('terminals')} />
            </Suspense>
          )}
          <div className="terminals" hidden={covered}>
            {problems.length > 0 && <div className="problems">{problems.join('\n')}</div>}
            {terminals.length === 0 && <div className="empty">Nenhum terminal. Abra um no +.</div>}
            {split && <button className="unsplit" onClick={() => setPanes(closeSplit)} title="Voltar a um terminal só">×</button>}
            {views.map((t) => (
              <TerminalView
                key={`${t.id}:${epoch}`}
                id={t.id}
                slot={covered ? null : slotOf(panes, t.id, room)}
                focused={!covered && t.id === activeId}
                onFocus={() => { unmark(t.id); setPanes((p) => select(p, t.id)); }}
                fontSize={fontSize}
                fontFamily={font.family}
              />
            ))}
          </div>
        </main>
        {focusOpen && <FocusPanel notes={notes} pomodoro={pomodoro?.state ?? null} remainingMs={remainingMs} now={new Date(now)} />}
      </div>
    </div>
  );
}
