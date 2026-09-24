import { useEffect, useRef, useState } from 'react';
import { api, connection, isWindow, on, onConnection, retryConnection } from './bridge';
import { bark } from './sound';
import { TitleBar } from './components/TitleBar';
import { Sidebar } from './components/Sidebar';
import { TerminalView } from './components/TerminalView';
import type { Conversation, Preset, State, Terminal, Usage } from './lib/types';
import { nextUnread } from './lib/unread';
import type { ConnectionState } from './windowsTransport';
import { applyZoom, parseFontSize, zoomKey, type ZoomAction } from './lib/zoom';
import { parseFont, type TerminalFont } from './lib/fonts';

const FONT_KEY = 'skye:fontSize';
const FAMILY_KEY = 'skye:font';

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
        <h2>não consegui abrir a skye</h2>
        <pre>{state.message}</pre>
        <button onClick={retryConnection}>tentar de novo</button>
      </div>
    );
  }
  return <div className="connection">iniciando a skye no WSL…</div>;
}

function storedFont(): TerminalFont {
  try {
    return parseFont(localStorage.getItem(FAMILY_KEY));
  } catch {
    return parseFont(null);
  }
}

const windowFocused = () => document.visibilityState === 'visible' && document.hasFocus();

const byCreation = (a: Terminal, b: Terminal) =>
  Date.parse(a.createdAt) - Date.parse(b.createdAt) || (a.id < b.id ? -1 : a.id > b.id ? 1 : 0);

export function App() {
  const [terminals, setTerminals] = useState<Terminal[]>([]);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [presets, setPresets] = useState<Preset[]>([]);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [sound, setSound] = useState(true);
  const [problems, setProblems] = useState<string[]>([]);
  const [fontSize, setFontSize] = useState(storedFontSize);
  const [font, setFont] = useState(storedFont);
  const [epoch, setEpoch] = useState(0);
  const [link, setLink] = useState(connection);
  const [focused, setFocused] = useState(windowFocused);
  const [usage, setUsage] = useState<Usage | null>(null);
  const [unread, setUnread] = useState<Set<string>>(() => new Set());
  const lastStates = useRef<Record<string, State>>({});

  const load = () => {
    api().List().then(setTerminals).catch(() => {});
    api().Conversations().then(setConversations).catch(() => {});
    api().Presets().then(setPresets).catch(() => {});
    api().Sound().then(setSound).catch(() => {});
    api().Problems().then(setProblems).catch(() => {});
    api().Usage().then(setUsage).catch(() => {});
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
    if (activeId && terminals.some((t) => t.id === activeId)) return;
    setActiveId(terminals[0]?.id ?? null);
  }, [terminals, activeId]);

  useEffect(() => {
    const prev = lastStates.current;
    setUnread((current) => nextUnread(prev, terminals, current, focused ? activeId : null));
    lastStates.current = Object.fromEntries(terminals.map((t) => [t.id, t.state]));
  }, [terminals, activeId, focused]);

  const views = [...terminals].sort(byCreation);

  const open = async (preset: string) => setActiveId((await api().NewTerminal(preset)).id);
  const resume = async (sessionId: string) => setActiveId((await api().Resume(sessionId)).id);
  const reorder = (list: Terminal[]) => {
    setTerminals(list);
    api().Reorder(list.map((t) => t.id)).catch(() => {});
  };
  const toggleSound = () => { api().SetSound(!sound); setSound(!sound); };
  const pickFont = (next: TerminalFont) => {
    setFont(next);
    try {
      localStorage.setItem(FAMILY_KEY, next.id);
    } catch {}
  };

  return (
    <div className={isWindow() ? 'app windowed' : 'app'}>
      {link.kind !== 'ready' && <ConnectionScreen state={link} />}
      <TitleBar presets={presets} sound={sound} ready={link.kind === 'ready'} onNew={open} onToggleSound={toggleSound} font={font} onPickFont={pickFont} />
      <div className="body">
        <Sidebar terminals={terminals} conversations={conversations} activeId={activeId} unread={unread} onSelect={setActiveId} onReorder={reorder} onResume={resume} usage={usage} />
        <main className="main">
          <div className="terminals">
            {problems.length > 0 && <div className="problems">{problems.join('\n')}</div>}
            {terminals.length === 0 && <div className="empty">Nenhum terminal. Abra um no +.</div>}
            {views.map((t) => <TerminalView key={`${t.id}:${epoch}`} id={t.id} active={t.id === activeId} fontSize={fontSize} fontFamily={font.family} />)}
          </div>
        </main>
      </div>
    </div>
  );
}
