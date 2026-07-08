import { useEffect, useMemo, useState } from 'react';
import { api } from '../api/client';
import type { ActivityEntry } from '../types';

const PAGE_SIZE = 100;

export function Activity() {
  const [entries, setEntries] = useState<ActivityEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState('');

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return entries;
    return entries.filter(
      (e) => (e.username ?? 'system').toLowerCase().includes(q) || e.event.toLowerCase().includes(q),
    );
  }, [entries, query]);

  useEffect(() => {
    api
      .listActivity()
      .then((page) => {
        setEntries(page);
        setHasMore(page.length === PAGE_SIZE);
      })
      .catch((err) => setError(err instanceof Error ? err.message : String(err)))
      .finally(() => setLoading(false));
  }, []);

  async function loadMore() {
    if (entries.length === 0) return;
    setLoadingMore(true);
    try {
      const page = await api.listActivity(entries[entries.length - 1].id);
      setEntries((prev) => [...prev, ...page]);
      setHasMore(page.length === PAGE_SIZE);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoadingMore(false);
    }
  }

  if (error) return <p className="srv-desc">Only admins can view the activity log.</p>;

  return (
    <div className="view active">
      <div className="dash-head">
        <h1>Activity</h1>
        <p>Recent actions across the panel.</p>
      </div>

      {!loading && (
        <div className="dash-toolbar">
          <div className="search-wrap">
            <span className="search-icon">⌕</span>
            <input
              type="text"
              placeholder="Search by user or event…"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
          </div>
        </div>
      )}

      {loading ? (
        <p className="srv-desc">Loading…</p>
      ) : (
        <div className="act-table">
          <div className="act-head">
            <span>User</span>
            <span>Event</span>
            <span>IP</span>
            <span>Time</span>
          </div>
          {filtered.map((entry) => (
            <div className="act-row" key={entry.id}>
              <div className="act-user">
                <div className="act-ava">
                  {(entry.username ?? '?').slice(0, 1).toUpperCase()}
                </div>
                <span>{entry.username ?? 'system'}</span>
              </div>
              <span className="act-event">{entry.event}</span>
              <span className="act-ip">{entry.ip_address ?? '—'}</span>
              <span className="act-time">{new Date(entry.created_at).toLocaleString()}</span>
            </div>
          ))}
          {entries.length === 0 && (
            <p className="srv-desc" style={{ padding: 16 }}>
              No activity yet.
            </p>
          )}
          {entries.length > 0 && filtered.length === 0 && (
            <p className="srv-desc" style={{ padding: 16 }}>
              No activity matches your search.
            </p>
          )}
        </div>
      )}

      {!loading && hasMore && entries.length > 0 && (
        <div style={{ display: 'flex', justifyContent: 'center', marginTop: 16 }}>
          <button className="btn-sm" onClick={loadMore} disabled={loadingMore}>
            {loadingMore ? 'Loading…' : 'Load more'}
          </button>
        </div>
      )}
    </div>
  );
}
