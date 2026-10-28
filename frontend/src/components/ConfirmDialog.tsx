// A question over the whole window. Cancel holds the focus, so an Enter typed on autopilot keeps
// things as they are; Esc or a click outside cancels too.
export function ConfirmDialog(props: { title: string; message: string; confirm: string; onConfirm: () => void; onCancel: () => void }) {
  return (
    <div className="dialog-backdrop" onMouseDown={(e) => { if (e.target === e.currentTarget) props.onCancel(); }} onKeyDown={(e) => { if (e.key === 'Escape') props.onCancel(); }}>
      <div className="dialog island" role="alertdialog" aria-modal="true" aria-label={props.title}>
        <strong>{props.title}</strong>
        <p>{props.message}</p>
        <div className="actions">
          <button autoFocus onClick={props.onCancel}>Cancelar</button>
          <button className="danger" onClick={props.onConfirm}>{props.confirm}</button>
        </div>
      </div>
    </div>
  );
}
