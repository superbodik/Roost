import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { DatabaseHost, ServerDatabase } from '../types';

interface Props {
  uuid: string;
}

export function DatabaseManager({ uuid }: Props) {
  const [databases, setDatabases] = useState<ServerDatabase[] | null>(null);
  const [forbidden, setForbidden] = useState(false);
  const [hosts, setHosts] = useState<DatabaseHost[]>([]);
  const [hostId, setHostId] = useState(0);
  const [name, setName] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [revealed, setRevealed] = useState<Record<number, boolean>>({});

  function refresh() {
    api
      .listServerDatabases(uuid)
      .then((d) => {
        setDatabases(d);
        setForbidden(false);
      })
      .catch(() => {
        setDatabases(null);
        setForbidden(true);
      });
  }

  useEffect(refresh, [uuid]);
  useEffect(() => {
    api.listDatabaseHosts().then(setHosts).catch(() => setHosts([]));
  }, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      await api.createServerDatabase(uuid, hostId, name);
      setName('');
      refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(db: ServerDatabase) {
    if (!window.confirm(`Delete database "${db.database_name}"? This drops it and its user permanently.`)) {
      return;
    }
    try {
      await api.deleteServerDatabase(uuid, db.id);
      refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  if (forbidden) {
    return (
      <p className="srv-desc">
        You don't have permission to view this server's databases.
      </p>
    );
  }

  if (databases === null) {
    return <p className="srv-desc">Loading…</p>;
  }

  return (
    <div>
      {error && <div className="login-error show" style={{ marginBottom: 12 }}>{error}</div>}

      {hosts.length === 0 ? (
        <p className="srv-desc">
          No database hosts registered yet — an admin needs to add one on the Nodes page first.
        </p>
      ) : (
        <div className="settings-card" style={{ marginBottom: 20 }}>
          <div className="settings-card-title">Create database</div>
          <form onSubmit={handleCreate}>
            <div className="settings-grid">
              <div className="sfield">
                <label htmlFor="db-host">Host</label>
                <select id="db-host" value={hostId} onChange={(e) => setHostId(Number(e.target.value))} required>
                  <option value={0} disabled>
                    Select a host…
                  </option>
                  {hosts.map((h) => (
                    <option key={h.id} value={h.id}>
                      {h.name}
                    </option>
                  ))}
                </select>
              </div>
              <div className="sfield">
                <label htmlFor="db-name">Name</label>
                <input
                  id="db-name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="letters, digits, underscore"
                  required
                />
              </div>
            </div>
            <div className="settings-foot">
              <button
                className="btn-primary"
                type="submit"
                disabled={submitting}
                style={{ width: 'auto', padding: '10px 20px' }}
              >
                {submitting ? 'Creating…' : 'Create database'}
              </button>
            </div>
          </form>
        </div>
      )}

      <div className="sch-list">
        {databases.map((db) => (
          <div className="sch-card" key={db.id}>
            <div className="sch-head">
              <span className="sch-name">{db.database_name}</span>
              <button className="file-act-btn del" onClick={() => handleDelete(db)}>
                Delete
              </button>
            </div>
            <div className="sch-meta" style={{ flexDirection: 'column', alignItems: 'flex-start', gap: 6 }}>
              <span>Host: {db.host}:{db.port}</span>
              <span>User: {db.username}</span>
              <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                Password: {revealed[db.id] ? db.password : '••••••••••••'}
                <button
                  className="btn-sm"
                  type="button"
                  onClick={() => setRevealed((r) => ({ ...r, [db.id]: !r[db.id] }))}
                >
                  {revealed[db.id] ? 'Hide' : 'Show'}
                </button>
                <button
                  className="btn-sm"
                  type="button"
                  onClick={() => navigator.clipboard?.writeText(db.password)}
                >
                  Copy
                </button>
              </span>
            </div>
          </div>
        ))}
        {databases.length === 0 && <p className="srv-desc">No databases yet.</p>}
      </div>
    </div>
  );
}
