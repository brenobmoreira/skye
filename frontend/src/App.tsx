import { useEffect, useState } from 'react';
import { api, on } from './bridge';
import { bark } from './sound';
import { TitleBar } from './components/TitleBar';
import { Sidebar } from './components/Sidebar';
import { TerminalView } from './components/TerminalView';
import { Composer } from './components/Composer';
import type { Conversation, Preset, Terminal } from './lib/types';

export function App() {
  const [terminals, setTerminals] = useState<Terminal[]>([]);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [presets, setPresets] = useState<Preset[]>([]);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [sound, setSound] = useState(true);
  const [problems, setProblems] = useState<string[]>([]);

  useEffect(() => {
    api().List().then(setTerminals);
    api().Conversations().then(setConversations);
    api().Presets().then(setPresets);
    api().Sound().then(setSound);
    api().Problems().then(setProblems);
    const offs = [
      on('terminals', (list: Terminal[]) => setTerminals(list)),
      on('conversations', (list: Conversation[]) => setConversations(list)),
      on('bark', () => { bark(); }),
      on('problems', (list: string[]) => setProblems(list)),
    ];
    const focus = () => api().SetFocused(true);
    const blur = () => api().SetFocused(false);
    window.addEventListener('focus', focus);
    window.addEventListener('blur', blur);
    api().SetFocused(document.hasFocus());
    return () => {
      offs.forEach((off) => off());
      window.removeEventListener('focus', focus);
      window.removeEventListener('blur', blur);
    };
  }, []);

  useEffect(() => {
    if (activeId && terminals.some((t) => t.id === activeId)) return;
    setActiveId(terminals[0]?.id ?? null);
  }, [terminals, activeId]);

  const open = async (preset: string) => setActiveId((await api().NewTerminal(preset)).id);
  const resume = async (sessionId: string) => setActiveId((await api().Resume(sessionId)).id);
  const toggleSound = () => { api().SetSound(!sound); setSound(!sound); };

  return (
    <div className="app">
      <TitleBar presets={presets} sound={sound} onNew={open} onToggleSound={toggleSound} />
      <div className="body">
        <Sidebar terminals={terminals} conversations={conversations} activeId={activeId} onSelect={setActiveId} onResume={resume} />
        <main className="main">
          <div className="terminals">
            {problems.length > 0 && <div className="problems">{problems.join('\n')}</div>}
            {terminals.length === 0 && <div className="empty">Nenhum terminal. Abra um no +.</div>}
            {terminals.map((t) => <TerminalView key={t.id} id={t.id} active={t.id === activeId} />)}
          </div>
          {activeId && <Composer key={activeId} id={activeId} />}
        </main>
      </div>
    </div>
  );
}
