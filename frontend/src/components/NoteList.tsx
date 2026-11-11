import { useEffect, useRef, useState, type DragEvent, type FocusEvent, type KeyboardEvent } from 'react';
import { api } from '../bridge';
import { canMove, dayOf, dueLabel, groupNotes, moveItems, repeatLabel, type Row } from '../lib/notes';
import { capitalize } from '../lib/text';
import type { NoteItem, Notes, Repeat } from '../lib/types';
import { useFlip } from './useFlip';

const errText = (e: unknown) => (e instanceof Error ? e.message : String(e));

const WEEKDAYS = ['Domingo', 'Segunda', 'Terça', 'Quarta', 'Quinta', 'Sexta', 'Sábado'];

const PinIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <path d="M12 17v5" />
    <path d="M9 3h6l-1 6 3 3v2H7v-2l3-3z" />
  </svg>
);

interface Draft {
  text: string;
  date: string;
  time: string;
  kind: '' | Repeat['kind'];
  weekday: number;
  day: number;
}

function draftOf(item: NoteItem, now: Date): Draft {
  const base = item.date ? new Date(`${item.date}T12:00`) : now;
  return {
    text: item.text,
    date: item.date ?? '',
    time: item.time ?? '',
    kind: item.repeat?.kind ?? '',
    weekday: item.repeat?.weekday ?? base.getDay(),
    day: item.repeat?.day ?? base.getDate(),
  };
}

function repeatOf(d: Draft): Repeat | null {
  if (d.kind === '') return null;
  if (d.kind === 'weekly') return { kind: 'weekly', weekday: d.weekday };
  if (d.kind === 'monthly') return { kind: 'monthly', day: d.day };
  return { kind: d.kind };
}

const rowId = (r: Row) => (r.kind === 'item' ? r.item.id : r.record.id);

export function NoteList({ notes, now }: { notes: Notes; now: Date }) {
  const [add, setAdd] = useState('');
  const [addError, setAddError] = useState('');
  const [editing, setEditing] = useState<string | null>(null);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [armed, setArmed] = useState<string | null>(null);
  const [tab, setTab] = useState<'open' | 'done'>('open');
  const [allHistory, setAllHistory] = useState(false);
  const [dragging, setDragging] = useState<string | null>(null);
  // The order shown while dragging, and after the drop until the Go side sends the saved list.
  const [preview, setPreview] = useState<NoteItem[] | null>(null);
  useEffect(() => { if (!dragging) setPreview(null); }, [notes]);
  const scrollRef = useRef<HTMLDivElement>(null);
  useFlip(scrollRef);

  const shown: Notes = preview ? { ...notes, items: preview } : notes;
  const groups = groupNotes(shown, now, allHistory ? Infinity : 30);
  const fail = (id: string) => (e: unknown) => setErrors((m) => ({ ...m, [id]: errText(e) }));
  const clear = (id: string) => setErrors((m) => {
    if (!(id in m)) return m;
    const next = { ...m };
    delete next[id];
    return next;
  });

  const submitAdd = () => {
    if (!add.trim()) return;
    api().AddNote(add).then(() => { setAdd(''); setAddError(''); }).catch((e) => setAddError(errText(e)));
  };
  const addKeys = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') submitAdd();
    if (e.key === 'Escape') { setAdd(''); setAddError(''); }
  };

  const toggle = (r: Row) => {
    const id = rowId(r);
    clear(id);
    const call = r.kind === 'done' || r.done ? api().UncheckNote(id) : api().CheckNote(id);
    call.catch(fail(id));
  };

  const remove = (id: string) => {
    if (armed !== id) { setArmed(id); return; }
    setArmed(null);
    api().DeleteNote(id).catch(fail(id));
  };

  const startEdit = (item: NoteItem) => { setEditing(item.id); setDraft(draftOf(item, now)); clear(item.id); };
  const cancelEdit = () => { setEditing(null); setDraft(null); };
  const saveEdit = () => {
    if (!editing || !draft) return;
    const id = editing;
    api()
      .EditNote(id, { text: draft.text, date: draft.date, time: draft.date ? draft.time : '', repeat: draft.date ? repeatOf(draft) : null })
      .then(() => { cancelEdit(); clear(id); })
      .catch(fail(id));
  };
  const editKeys = (e: KeyboardEvent) => {
    if (e.key === 'Enter') { e.preventDefault(); saveEdit(); }
    if (e.key === 'Escape') { e.preventDefault(); cancelEdit(); }
  };
  // Leaving the form, not just moving between its fields, saves it.
  const editBlur = (e: FocusEvent<HTMLDivElement>) => {
    if (!e.currentTarget.contains(e.relatedTarget as Node | null)) saveEdit();
  };

  const editor = (id: string) => draft && (
    <div className="note-edit" onBlur={editBlur} onKeyDown={editKeys}>
      <input autoFocus className="text" value={draft.text} onChange={(e) => setDraft({ ...draft, text: e.target.value })} />
      <div className="when">
        <input type="date" value={draft.date} onChange={(e) => setDraft({ ...draft, date: e.target.value })} title="Data" />
        <input type="time" value={draft.time} disabled={!draft.date} onChange={(e) => setDraft({ ...draft, time: e.target.value })} title="Hora (avisa nesse horário)" />
      </div>
      <div className="when">
        <select value={draft.kind} disabled={!draft.date} onChange={(e) => setDraft({ ...draft, kind: e.target.value as Draft['kind'] })} title={draft.date ? 'Repetição' : 'Repetir precisa de data'}>
          <option value="">Não repete</option>
          <option value="daily">Todo dia</option>
          <option value="weekdays">Dias úteis</option>
          <option value="weekly">Toda semana</option>
          <option value="monthly">Todo mês</option>
        </select>
        {draft.kind === 'weekly' && (
          <select value={draft.weekday} onChange={(e) => setDraft({ ...draft, weekday: Number(e.target.value) })}>
            {WEEKDAYS.map((w, i) => <option key={w} value={i}>{w}</option>)}
          </select>
        )}
        {draft.kind === 'monthly' && (
          <input type="number" min={1} max={31} value={draft.day} onChange={(e) => setDraft({ ...draft, day: Number(e.target.value) })} title="Dia do mês" />
        )}
      </div>
      <div className="actions">
        <button onMouseDown={(e) => e.preventDefault()} onClick={saveEdit}>Salvar</button>
        <button onMouseDown={(e) => e.preventDefault()} onClick={cancelEdit}>Cancelar</button>
      </div>
      {errors[id] && <div className="note-error">{errors[id]}</div>}
    </div>
  );

  const pin = (item: NoteItem) => { clear(item.id); api().PinNote(item.id, !item.pinned).catch(fail(item.id)); };

  // Dragging: the dragged note stays where the pointer is and the others slide out of its way.
  const dragStart = (e: DragEvent, id: string) => {
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', id);
    setDragging(id);
    setPreview(notes.items);
  };
  const dragOver = (e: DragEvent, id: string) => {
    if (!dragging) return;
    if (dragging === id) { e.preventDefault(); return; }
    if (!canMove(shown, dragging, id, now)) return;
    e.preventDefault();
    setPreview(moveItems(shown.items, dragging, id));
  };
  const drop = (e: DragEvent) => {
    e.preventDefault();
    const id = dragging;
    setDragging(null);
    if (!id || !preview) return;
    // The drop is one move on the saved list: to the index where the preview left the note.
    const at = preview.findIndex((i) => i.id === id);
    const from = notes.items.findIndex((i) => i.id === id);
    if (at < 0 || at === from) { setPreview(null); return; }
    api().MoveNote(id, notes.items[at].id).catch((err) => { setPreview(null); fail(id)(err); });
  };
  const dragEnd = () => { if (dragging) { setDragging(null); setPreview(null); } };

  const row = (r: Row, late: boolean, past: boolean) => {
    const id = rowId(r);
    if (r.kind === 'item' && editing === r.item.id) return <li key={id} data-flip={id} className="note editing">{editor(id)}</li>;
    const done = r.kind === 'done' || r.done;
    const text = r.kind === 'item' ? r.item.text : r.record.text;
    const due = r.kind === 'item' && !done ? dueLabel(r.item, now) : '';
    const repeat = r.kind === 'item' && r.item.repeat ? repeatLabel(r.item.repeat) : r.kind === 'done' ? 'repete' : '';
    const movable = r.kind === 'item' && !done;
    return (
      <li
        key={id}
        data-flip={id}
        className={`note${done ? ' done' : ''}${late && !done ? ' late' : ''}${dragging === id ? ' dragging' : ''}${r.kind === 'item' && r.item.pinned ? ' pinned' : ''}`}
        draggable={movable}
        onDragStart={movable ? (e) => dragStart(e, id) : undefined}
        onDragOver={(e) => dragOver(e, id)}
        onDrop={drop}
        onDragEnd={dragEnd}
        onMouseLeave={() => { if (armed === id) setArmed(null); }}
      >
        <button className={done ? 'check on' : 'check'} disabled={past} onClick={() => toggle(r)} title={past ? 'Feito antes de hoje' : done ? 'Desmarcar' : 'Marcar como feito'} aria-label={done ? 'Desmarcar' : 'Marcar'} />
        <div className="note-body">
          <span className="text" onClick={() => { if (movable && r.kind === 'item') startEdit(r.item); }}>{text}</span>
          {(due || repeat) && <span className="meta">{capitalize([due, repeat].filter(Boolean).join(' · '))}</span>}
          {errors[id] && <span className="note-error">{errors[id]}</span>}
        </div>
        <div className="note-tools">
          {movable && r.kind === 'item' && (
            <button className={r.item.pinned ? 'pin on' : 'pin'} onClick={() => pin(r.item)} aria-label={r.item.pinned ? 'Desafixar' : 'Fixar'} title={r.item.pinned ? 'Desafixar' : 'Fixar no topo'}>
              <PinIcon />
            </button>
          )}
          <button className={armed === id ? 'remove armed' : 'remove'} onClick={() => remove(id)} title="Apagar">{armed === id ? 'Apagar?' : '×'}</button>
        </div>
      </li>
    );
  };

  const group = (title: string, rows: Row[], cls = '') => rows.length > 0 && (
    <section className={`note-group ${cls}`}>
      <h4>{title}</h4>
      <ul>{rows.map((r) => row(r, cls === 'overdue', false))}</ul>
    </section>
  );

  const doneCount = groups.done.reduce((n, h) => n + h.rows.length, 0);
  const openCount = groups.pinned.length + groups.overdue.length + groups.today.length + groups.upcoming.length + groups.undated.length;
  const today = dayOf(now);
  const dayTitle = (day: string) => {
    const [y, m, d] = day.split('-').map(Number);
    const diff = Math.round((new Date(`${today}T12:00`).getTime() - new Date(`${day}T12:00`).getTime()) / 86_400_000);
    if (diff === 0) return 'Hoje';
    if (diff === 1) return 'Ontem';
    return capitalize(new Date(y, m - 1, d).toLocaleDateString('pt-BR', { weekday: 'short', day: '2-digit', month: '2-digit' }));
  };

  return (
    <div className="notes">
      <div className="note-tabs" role="tablist">
        <span className="note-tabs-ind" style={{ transform: tab === 'done' ? 'translateX(100%)' : 'none' }} />
        <button role="tab" aria-selected={tab === 'open'} className={tab === 'open' ? 'on' : ''} onClick={() => setTab('open')}>
          Pendentes <span className="count">{openCount}</span>
        </button>
        <button role="tab" aria-selected={tab === 'done'} className={tab === 'done' ? 'on' : ''} onClick={() => setTab('done')}>
          Completas <span className="count">{doneCount}</span>
        </button>
      </div>
      <div className="note-scroll" ref={scrollRef}>
        {tab === 'open' && (
          <>
            {openCount === 0 && <p className="none">Nada pendente. Escreva abaixo e dê Enter.</p>}
            {group('Fixadas', groups.pinned, 'pinned')}
            {group('Atrasadas', groups.overdue, 'overdue')}
            {group('Hoje', groups.today)}
            {group('Próximas', groups.upcoming)}
            {group('Sem data', groups.undated)}
          </>
        )}
        {tab === 'done' && (
          <>
            {doneCount === 0 && <p className="none">Nada completo ainda.</p>}
            {groups.done.map((h) => (
              <section key={h.day} className="note-group">
                <h4>{dayTitle(h.day)}</h4>
                <ul>{h.rows.map((r) => row(r, false, h.day !== today))}</ul>
              </section>
            ))}
            {doneCount > 0 && !allHistory && <button className="more" onClick={() => setAllHistory(true)}>Ver mais</button>}
          </>
        )}
      </div>
      {tab === 'open' && (
        <div className="note-add">
          <input placeholder="+ Anotar…" value={add} onChange={(e) => setAdd(e.target.value)} onKeyDown={addKeys} />
          {addError && <div className="note-error">{addError}</div>}
        </div>
      )}
    </div>
  );
}
