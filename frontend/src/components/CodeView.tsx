import { useEffect, useRef } from 'react';
import { basicSetup, EditorView } from 'codemirror';
import { Compartment, EditorState, Prec, type Extension } from '@codemirror/state';
import { keymap } from '@codemirror/view';
import { LanguageDescription } from '@codemirror/language';
import { languages } from '@codemirror/language-data';
import { oneDark } from '@codemirror/theme-one-dark';
import { unifiedMergeView } from '@codemirror/merge';

// The editor sits on the terminal background, in the terminal font.
const look = EditorView.theme(
  {
    '&': { height: '100%', backgroundColor: 'var(--term-bg)', fontSize: '12.5px' },
    '.cm-scroller': { fontFamily: "'JetBrains Mono', monospace", lineHeight: '1.5' },
    '.cm-gutters': { backgroundColor: 'var(--term-bg)', borderRight: '1px solid var(--line)' },
    '&.cm-focused': { outline: 'none' },
  },
  { dark: true },
);

async function languageFor(path: string): Promise<Extension> {
  const desc = LanguageDescription.matchFilename(languages, path.split('/').pop() ?? path);
  if (!desc) return [];
  try {
    return await desc.load();
  } catch {
    return [];
  }
}

// A CodeMirror view of a file. With original it shows the changes from original inline, read
// only; with editable the user can type, and Ctrl+S calls onSave.
export function CodeView(props: { path: string; doc: string; original?: string; editable?: boolean; onChange?: (doc: string) => void; onSave?: () => void }) {
  const host = useRef<HTMLDivElement>(null);
  const handlers = useRef(props);
  handlers.current = props;

  useEffect(() => {
    const lang = new Compartment();
    const editable = !!props.editable && props.original === undefined;
    const view = new EditorView({
      parent: host.current!,
      state: EditorState.create({
        doc: props.doc,
        extensions: [
          basicSetup,
          oneDark,
          look,
          lang.of([]),
          EditorState.readOnly.of(!editable),
          EditorView.editable.of(editable),
          Prec.highest(keymap.of([{ key: 'Mod-s', preventDefault: true, run: () => { handlers.current.onSave?.(); return true; } }])),
          EditorView.updateListener.of((u) => { if (u.docChanged) handlers.current.onChange?.(u.state.doc.toString()); }),
          props.original === undefined ? [] : unifiedMergeView({ original: props.original, mergeControls: false, collapseUnchanged: { margin: 3, minSize: 6 } }),
        ],
      }),
    });
    if (editable) view.focus();
    let alive = true;
    languageFor(props.path).then((ext) => { if (alive) view.dispatch({ effects: lang.reconfigure(ext) }); });
    return () => {
      alive = false;
      view.destroy();
    };
  }, [props.path, props.doc, props.original, props.editable]);

  return <div className="code" ref={host} />;
}
