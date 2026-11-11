import { useState, type FocusEvent, type KeyboardEvent } from 'react';
import { api } from '../bridge';
import { dayOf, dueLabel, groupNotes, repeatLabel, type Row } from '../lib/notes';
import type { NoteItem, Notes, Repeat } from '../lib/types';

const errText = (e: unknown) => (e instanceof Error ? e.message : String(e));

const WEEKDAYS = ['domingo', 'segunda', 'terça', 'quarta', 'quinta', 'sexta', 'sábado'];

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
  const [historyOpen, setHistoryOpen] = useState(false);
  const [allHistory, setAllHistory] = useState(false);

  const groups = groupNotes(notes, now, allHistory ? Infinity : 30);
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
        <input type="date" value={draft.date} onChange={(e) => setDraft({ ...draft, date: e.target.value })} title="data" />
        <input type="time" value={draft.time} disabled={!draft.date} onChange={(e) => setDraft({ ...draft, time: e.target.value })} title="hora (avisa nesse horário)" />
      </div>
      <div className="when">
        <select value={draft.kind} disabled={!draft.date} onChange={(e) => setDraft({ ...draft, kind: e.target.value as Draft['kind'] })} title={draft.date ? 'repetição' : 'repetir precisa de data'}>
          <option value="">não repete</option>
          <option value="daily">todo dia</option>
          <option value="weekdays">dias úteis</option>
          <option value="weekly">toda semana</option>
          <option value="monthly">todo mês</option>
        </select>
        {draft.kind === 'weekly' && (
          <select value={draft.weekday} onChange={(e) => setDraft({ ...draft, weekday: Number(e.target.value) })}>
            {WEEKDAYS.map((w, i) => <option key={w} value={i}>{w}</option>)}
          </select>
        )}
        {draft.kind === 'monthly' && (
          <input type="number" min={1} max={31} value={draft.day} onChange={(e) => setDraft({ ...draft, day: Number(e.target.value) })} title="dia do mês" />
        )}
      </div>
      <div className="actions">
        <button onMouseDown={(e) => e.preventDefault()} onClick={saveEdit}>salvar</button>
        <button onMouseDown={(e) => e.preventDefault()} onClick={cancelEdit}>cancelar</button>
      </div>
      {errors[id] && <div className="note-error">{errors[id]}</div>}
    </div>
  );

  const row = (r: Row, late: boolean, past: boolean) => {
    const id = rowId(r);
    if (r.kind === 'item' && editing === r.item.id) return <li key={id} className="note editing">{editor(id)}</li>;
    const done = r.kind === 'done' || r.done;
    const text = r.kind === 'item' ? r.item.text : r.record.text;
    const due = r.kind === 'item' && !past ? dueLabel(r.item, now) : '';
    const repeat = r.kind === 'item' && r.item.repeat ? repeatLabel(r.item.repeat) : r.kind === 'done' ? 'repete' : '';
    return (
      <li key={id} className={`note${done ? ' done' : ''}${late && !done ? ' late' : ''}`} onMouseLeave={() => { if (armed === id) setArmed(null); }}>
        <button className={done ? 'check on' : 'check'} disabled={past} onClick={() => toggle(r)} title={past ? 'feito antes de hoje' : done ? 'desmarcar' : 'marcar como feito'} aria-label={done ? 'desmarcar' : 'marcar'} />
        <div className="note-body">
          <span className="text" onClick={() => { if (r.kind === 'item' && !done) startEdit(r.item); }}>{text}</span>
          {(due || repeat) && <span className="meta">{[due, repeat].filter(Boolean).join(' · ')}</span>}
          {errors[id] && <span className="note-error">{errors[id]}</span>}
        </div>
        <button className={armed === id ? 'remove armed' : 'remove'} onClick={() => remove(id)} title="apagar">{armed === id ? 'apagar?' : '×'}</button>
      </li>
    );
  };

  const group = (title: string, rows: Row[], cls = '') => rows.length > 0 && (
    <section className={`note-group ${cls}`}>
      <h4>{title}</h4>
      <ul>{rows.map((r) => row(r, cls === 'overdue', false))}</ul>
    </section>
  );

  const historyCount = groups.history.reduce((n, h) => n + h.rows.length, 0);
  const today = dayOf(now);
  const dayTitle = (day: string) => {
    const [y, m, d] = day.split('-').map(Number);
    const date = new Date(y, m - 1, d);
    const diff = Math.round((new Date(`${today}T12:00`).getTime() - new Date(`${day}T12:00`).getTime()) / 86_400_000);
    if (diff === 1) return 'ontem';
    return date.toLocaleDateString('pt-BR', { weekday: 'short', day: '2-digit', month: '2-digit' });
  };

  const empty = !groups.overdue.length && !groups.today.length && !groups.upcoming.length && !groups.undated.length;

  return (
    <div className="notes">
      <div className="note-scroll">
        {empty && <p className="none">Nada anotado. Escreva abaixo e dê Enter.</p>}
        {group('Atrasadas', groups.overdue, 'overdue')}
        {group('Hoje', groups.today)}
        {group('Próximas', groups.upcoming)}
        {group('Sem data', groups.undated)}
        {historyCount > 0 && (
          <section className="note-group history">
            <h4 className="toggle" onClick={() => setHistoryOpen(!historyOpen)}>{historyOpen ? '▾' : '▸'} Feitas · {historyCount}</h4>
            {historyOpen && groups.history.map((h) => (
              <div key={h.day}>
                <h5>{dayTitle(h.day)}</h5>
                <ul>{h.rows.map((r) => row(r, false, true))}</ul>
              </div>
            ))}
            {historyOpen && !allHistory && <button className="more" onClick={() => setAllHistory(true)}>ver mais</button>}
          </section>
        )}
      </div>
      <div className="note-add">
        <input placeholder="+ anotar…" value={add} onChange={(e) => setAdd(e.target.value)} onKeyDown={addKeys} />
        {addError && <div className="note-error">{addError}</div>}
      </div>
    </div>
  );
}
