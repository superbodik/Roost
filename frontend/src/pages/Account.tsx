import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { ApiKey, CreateApiKeyResponse } from '../types';

export function Account() {
  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [name, setName] = useState('');
  const [justCreated, setJustCreated] = useState<CreateApiKeyResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  function refresh() {
    api.listApiKeys().then(setKeys).catch(() => {});
  }

  useEffect(refresh, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      const created = await api.createApiKey(name);
      setJustCreated(created);
      setName('');
      refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(id: number) {
    try {
      await api.deleteApiKey(id);
      refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <div className="view active">
      <div className="dash-head">
        <h1>Account</h1>
        <p>API keys for programmatic access.</p>
      </div>

      <div className="acc-grid">
        <div className="acc-card">
          <div className="acc-card-title">API Keys</div>

          {justCreated && (
            <div className="api-item" style={{ marginBottom: 12 }}>
              <span className="api-key">{justCreated.token}</span>
              <button
                className="btn-sm"
                onClick={() => navigator.clipboard?.writeText(justCreated.token)}
              >
                Copy
              </button>
            </div>
          )}

          <div className="api-list">
            {keys.map((k) => (
              <div className="api-item" key={k.id}>
                <span className="api-memo">{k.name}</span>
                <span className="api-used">
                  {k.last_used_at ? new Date(k.last_used_at).toLocaleDateString() : 'never used'}
                </span>
                <button className="file-act-btn del" onClick={() => handleDelete(k.id)}>
                  Delete
                </button>
              </div>
            ))}
            {keys.length === 0 && <p className="srv-desc">No API keys yet.</p>}
          </div>

          <form onSubmit={handleCreate} style={{ marginTop: 16 }}>
            <div className="sfield">
              <label htmlFor="key-name">New key name</label>
              <input
                id="key-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. CI deploy"
                required
              />
            </div>
            {error && (
              <div className="login-error show" style={{ marginTop: 12 }}>
                {error}
              </div>
            )}
            <div className="settings-foot">
              <button
                className="btn-primary"
                type="submit"
                disabled={submitting}
                style={{ width: 'auto', padding: '10px 20px' }}
              >
                {submitting ? 'Creating…' : 'Create key'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}
