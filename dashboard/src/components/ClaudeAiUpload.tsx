import { useState, useCallback } from 'react';

export function ClaudeAiUpload() {
  const [status, setStatus] = useState<'idle' | 'uploading' | 'done' | 'error'>('idle');
  const [message, setMessage] = useState('');
  const [dragOver, setDragOver] = useState(false);

  const upload = useCallback(async (file: File) => {
    setStatus('uploading');
    setMessage('');
    const form = new FormData();
    form.append('file', file);
    try {
      const res = await fetch('/api/upload/claude-ai', { method: 'POST', body: form });
      if (!res.ok) {
        if (res.status === 404) {
          throw new Error('アップロードエンドポイントは未実装です (Phase 3)。');
        }
        throw new Error(`HTTP ${res.status}`);
      }
      setStatus('done');
      setMessage('アップロード完了');
    } catch (err: unknown) {
      setStatus('error');
      setMessage(err instanceof Error ? err.message : String(err));
    }
  }, []);

  const onDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setDragOver(false);
      const file = e.dataTransfer.files[0];
      if (file) upload(file);
    },
    [upload]
  );

  const onFileChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (file) upload(file);
    },
    [upload]
  );

  return (
    <div className="card">
      <h3>claude.ai データアップロード</h3>
      <div
        className={`drop-zone ${dragOver ? 'drop-zone-active' : ''}`}
        onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
        onDragLeave={() => setDragOver(false)}
        onDrop={onDrop}
      >
        <p>ZIPファイルをドラッグ＆ドロップ、またはクリックして選択</p>
        <input type="file" accept=".zip" onChange={onFileChange} />
      </div>
      {status === 'uploading' && <p className="info">アップロード中...</p>}
      {status === 'done' && <p className="success">{message}</p>}
      {status === 'error' && <p className="error">{message}</p>}
    </div>
  );
}
