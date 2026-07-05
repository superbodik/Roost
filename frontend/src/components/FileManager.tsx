import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { FileEntry } from '../types';

interface Props {
  uuid: string;
}

function joinPath(dir: string, name: string): string {
  return dir.replace(/\/$/, '') + '/' + name;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const kb = bytes / 1024;
  if (kb < 1024) return `${kb.toFixed(1)} KB`;
  return `${(kb / 1024).toFixed(1)} MB`;
}

function formatDate(unixSeconds: number): string {
  return new Date(unixSeconds * 1000).toLocaleString();
}

export function FileManager({ uuid }: Props) {
  const [path, setPath] = useState('/');
  const [entries, setEntries] = useState<FileEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editingFile, setEditingFile] = useState<string | null>(null);
  const [fileContent, setFileContent] = useState('');
  const [saving, setSaving] = useState(false);

  function refresh() {
    setLoading(true);
    setError(null);
    api
      .listFiles(uuid, path)
      .then(setEntries)
      .catch((err) => setError(err instanceof Error ? err.message : String(err)))
      .finally(() => setLoading(false));
  }

  useEffect(refresh, [uuid, path]);

  const segments = path.split('/').filter(Boolean);

  function goToSegment(index: number) {
    setPath('/' + segments.slice(0, index + 1).join('/'));
  }

  async function openEntry(entry: FileEntry) {
    if (entry.is_directory) {
      setPath(joinPath(path, entry.name));
      return;
    }
    try {
      const content = await api.readFile(uuid, joinPath(path, entry.name));
      setEditingFile(joinPath(path, entry.name));
      setFileContent(content);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function handleSave() {
    if (!editingFile) return;
    setSaving(true);
    try {
      await api.writeFile(uuid, editingFile, fileContent);
      setEditingFile(null);
      refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(entry: FileEntry, e: React.MouseEvent) {
    e.stopPropagation();
    if (!window.confirm(`Delete ${entry.name}?`)) return;
    try {
      await api.deleteFile(uuid, joinPath(path, entry.name));
      refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function handleNewFolder() {
    const name = window.prompt('Folder name');
    if (!name) return;
    try {
      await api.createDirectory(uuid, joinPath(path, name));
      refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  if (editingFile) {
    return (
      <div>
        <div className="files-toolbar">
          <div className="files-path">{editingFile}</div>
          <button className="btn-sm primary" onClick={handleSave} disabled={saving}>
            {saving ? 'Saving…' : 'Save'}
          </button>
          <button className="btn-sm" onClick={() => setEditingFile(null)}>
            Cancel
          </button>
        </div>
        <textarea
          value={fileContent}
          onChange={(e) => setFileContent(e.target.value)}
          spellCheck={false}
          style={{
            width: '100%',
            minHeight: 380,
            background: '#070508',
            color: 'var(--text)',
            border: '1px solid var(--border)',
            borderRadius: 12,
            padding: 14,
            fontFamily: 'var(--font-mono)',
            fontSize: 12,
            resize: 'vertical',
          }}
        />
      </div>
    );
  }

  return (
    <div>
      <div className="files-toolbar">
        <div className="files-path">
          <span className="path-seg" onClick={() => setPath('/')}>
            /
          </span>
          {segments.map((seg, i) => (
            <span key={i}>
              <span className="path-sep"> / </span>
              <span className="path-seg" onClick={() => goToSegment(i)}>
                {seg}
              </span>
            </span>
          ))}
        </div>
        <button className="btn-sm primary" onClick={handleNewFolder}>
          + Folder
        </button>
      </div>

      {error && (
        <div className="login-error show" style={{ marginBottom: 12 }}>
          {error}
        </div>
      )}

      <div className="files-table">
        <div className="files-table-head">
          <span>Name</span>
          <span>Size</span>
          <span>Modified</span>
          <span>Actions</span>
        </div>
        {loading ? (
          <p className="srv-desc" style={{ padding: 16 }}>
            Loading…
          </p>
        ) : (
          entries.map((entry) => (
            <div className="file-row" key={entry.name} onClick={() => openEntry(entry)}>
              <div className="file-name">
                <span className="file-icon">{entry.is_directory ? '📁' : '📄'}</span>
                <span>{entry.name}</span>
              </div>
              <span className="file-size">
                {entry.is_directory ? '—' : formatSize(entry.size_bytes)}
              </span>
              <span className="file-modified">{formatDate(entry.modified_at)}</span>
              <div className="file-actions">
                <button className="file-act-btn del" onClick={(e) => handleDelete(entry, e)}>
                  Delete
                </button>
              </div>
            </div>
          ))
        )}
        {!loading && entries.length === 0 && (
          <p className="srv-desc" style={{ padding: 16 }}>
            Empty directory.
          </p>
        )}
      </div>
    </div>
  );
}
