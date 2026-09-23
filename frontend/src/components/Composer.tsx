import { useState } from 'react';
import { api } from '../bridge';
import { composerSubmits } from '../lib/keys';

export function Composer({ id }: { id: string }) {
  const [text, setText] = useState('');
  const submit = () => {
    if (!text.trim()) return;
    api().Paste(id, text);
    setText('');
  };
  return (
    <div className="composer">
      <textarea
        value={text}
        placeholder="Enter quebra linha · Ctrl+Enter envia"
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => {
          if (composerSubmits(e)) {
            e.preventDefault();
            submit();
          }
        }}
      />
      <button onClick={submit}>enviar</button>
    </div>
  );
}
