import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { Allocation, CreateNodeResponse, Node } from '../types';

const INSTALL_SCRIPT_URL = 'https://raw.githubusercontent.com/superbodik/sbPanel/main/install.sh';

function nodeInstallCommand(daemonToken: string): string {
  return `WINGSD_DAEMON_TOKEN=${daemonToken} bash <(curl -sSL ${INSTALL_SCRIPT_URL})`;
}

export function Nodes() {
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [justCreated, setJustCreated] = useState<CreateNodeResponse | null>(null);

  const [form, setForm] = useState({
    name: '',
    fqdn: '',
    location_id: 1,
    memory_mb: 8192,
    disk_mb: 102400,
  });
  const [submitting, setSubmitting] = useState(false);

  const [allocationNodeId, setAllocationNodeId] = useState(0);
  const [allocations, setAllocations] = useState<Allocation[]>([]);
  const [allocForm, setAllocForm] = useState({ ip: '', port: 25565 });
  const [allocError, setAllocError] = useState<string | null>(null);
  const [allocSubmitting, setAllocSubmitting] = useState(false);

  function refreshAllocations(nodeId: number) {
    if (!nodeId) {
      setAllocations([]);
      return;
    }
    api
      .listAllocations(nodeId)
      .then(setAllocations)
      .catch(() => setAllocations([]));
  }

  useEffect(() => {
    refreshAllocations(allocationNodeId);
  }, [allocationNodeId]);

  async function handleCreateAllocation(e: React.FormEvent) {
    e.preventDefault();
    setAllocSubmitting(true);
    setAllocError(null);
    try {
      await api.createAllocation({
        node_id: allocationNodeId,
        ip: allocForm.ip,
        port: allocForm.port,
      });
      setAllocForm((f) => ({ ...f, port: f.port + 1 }));
      refreshAllocations(allocationNodeId);
    } catch (err) {
      setAllocError(err instanceof Error ? err.message : String(err));
    } finally {
      setAllocSubmitting(false);
    }
  }

  function refresh() {
    setLoading(true);
    api
      .listNodes()
      .then(setNodes)
      .catch((err) => setError(err instanceof Error ? err.message : String(err)))
      .finally(() => setLoading(false));
  }

  useEffect(refresh, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      const created = await api.createNode(form);
      setJustCreated(created);
      setForm((f) => ({ ...f, name: '', fqdn: '' }));
      refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="view active">
      <div className="dash-head">
        <h1>Nodes</h1>
        <p>Machines running wingsd, ready to host servers.</p>
      </div>

      {justCreated && (
        <div className="acc-card" style={{ marginBottom: 20 }}>
          <div className="acc-card-title">Node created — run this on the node</div>
          <p className="srv-desc" style={{ marginBottom: 10 }}>
            Copy this command and run it on the node's server (as root). It installs Docker
            and wingsd and registers the daemon token automatically — no prompts.
          </p>
          <div className="api-item">
            <span className="api-key">{nodeInstallCommand(justCreated.daemon_token)}</span>
            <button
              className="btn-sm"
              onClick={() => navigator.clipboard?.writeText(nodeInstallCommand(justCreated.daemon_token))}
            >
              Copy
            </button>
          </div>
          <p className="srv-desc" style={{ marginTop: 12, marginBottom: 6 }}>
            Raw token, shown once, in case you're installing manually:
          </p>
          <div className="api-item">
            <span className="api-key">{justCreated.daemon_token}</span>
            <button
              className="btn-sm"
              onClick={() => navigator.clipboard?.writeText(justCreated.daemon_token)}
            >
              Copy
            </button>
          </div>
          <div className="settings-foot">
            <button className="btn-sm" onClick={() => setJustCreated(null)}>
              Done
            </button>
          </div>
        </div>
      )}

      {error && <div className="login-error show" style={{ marginBottom: 16 }}>{error}</div>}

      <div className="settings-card" style={{ marginBottom: 24 }}>
        <div className="settings-card-title">Add node</div>
        <form onSubmit={handleCreate}>
          <div className="settings-grid">
            <div className="sfield">
              <label htmlFor="node-name">Name</label>
              <input
                id="node-name"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                placeholder="node-1"
                required
              />
            </div>
            <div className="sfield">
              <label htmlFor="node-fqdn">FQDN / IP</label>
              <input
                id="node-fqdn"
                value={form.fqdn}
                onChange={(e) => setForm((f) => ({ ...f, fqdn: e.target.value }))}
                placeholder="node1.example.com"
                required
              />
            </div>
            <div className="sfield">
              <label htmlFor="node-memory">Memory (MB)</label>
              <input
                id="node-memory"
                type="number"
                value={form.memory_mb}
                onChange={(e) => setForm((f) => ({ ...f, memory_mb: Number(e.target.value) }))}
                required
              />
            </div>
            <div className="sfield">
              <label htmlFor="node-disk">Disk (MB)</label>
              <input
                id="node-disk"
                type="number"
                value={form.disk_mb}
                onChange={(e) => setForm((f) => ({ ...f, disk_mb: Number(e.target.value) }))}
                required
              />
            </div>
            <div className="sfield">
              <label htmlFor="node-location">Location ID</label>
              <input
                id="node-location"
                type="number"
                value={form.location_id}
                onChange={(e) => setForm((f) => ({ ...f, location_id: Number(e.target.value) }))}
                required
              />
            </div>
          </div>
          <div className="settings-foot">
            <button className="btn-primary" type="submit" disabled={submitting} style={{ width: 'auto', padding: '10px 20px' }}>
              {submitting ? 'Creating…' : 'Create node'}
            </button>
          </div>
        </form>
      </div>

      {loading ? (
        <p className="srv-desc">Loading nodes…</p>
      ) : (
        <div className="db-table">
          <div className="db-head">
            <span>Name</span>
            <span>Address</span>
            <span>Memory / Disk</span>
            <span>Status</span>
          </div>
          {nodes.map((node) => (
            <div className="db-row" key={node.id}>
              <span className="db-name">{node.name}</span>
              <span className="db-pw">
                {node.scheme}://{node.fqdn}:{node.daemon_port}
              </span>
              <span>{node.memory_mb} MB / {node.disk_mb} MB</span>
              <span>{node.maintenance_mode ? 'Maintenance' : node.last_seen_at ? 'Online' : 'Never seen'}</span>
            </div>
          ))}
          {nodes.length === 0 && <p className="srv-desc" style={{ padding: 16 }}>No nodes yet.</p>}
        </div>
      )}

      <div className="settings-card" style={{ marginTop: 24 }}>
        <div className="settings-card-title">Allocations</div>
        <div className="settings-grid" style={{ marginBottom: 16 }}>
          <div className="sfield">
            <label htmlFor="alloc-node">Node</label>
            <select
              id="alloc-node"
              value={allocationNodeId}
              onChange={(e) => setAllocationNodeId(Number(e.target.value))}
            >
              <option value={0} disabled>
                Select a node…
              </option>
              {nodes.map((n) => (
                <option key={n.id} value={n.id}>
                  {n.name}
                </option>
              ))}
            </select>
          </div>
        </div>

        {allocationNodeId > 0 && (
          <>
            <form onSubmit={handleCreateAllocation}>
              <div className="settings-grid">
                <div className="sfield">
                  <label htmlFor="alloc-ip">IP</label>
                  <input
                    id="alloc-ip"
                    value={allocForm.ip}
                    onChange={(e) => setAllocForm((f) => ({ ...f, ip: e.target.value }))}
                    placeholder="node's public IP"
                    required
                  />
                </div>
                <div className="sfield">
                  <label htmlFor="alloc-port">Port</label>
                  <input
                    id="alloc-port"
                    type="number"
                    value={allocForm.port}
                    onChange={(e) => setAllocForm((f) => ({ ...f, port: Number(e.target.value) }))}
                    required
                  />
                </div>
              </div>
              {allocError && <div className="login-error show" style={{ marginTop: 12 }}>{allocError}</div>}
              <div className="settings-foot">
                <button
                  className="btn-sm primary"
                  type="submit"
                  disabled={allocSubmitting}
                >
                  {allocSubmitting ? 'Adding…' : 'Add allocation'}
                </button>
              </div>
            </form>

            <div className="db-table" style={{ marginTop: 16 }}>
              <div className="db-head">
                <span>Address</span>
                <span>Status</span>
                <span />
                <span />
              </div>
              {allocations.map((a) => (
                <div className="db-row" key={a.id}>
                  <span className="db-name">
                    {a.ip}:{a.port}
                  </span>
                  <span>{a.server_id ? 'In use' : 'Free'}</span>
                  <span />
                  <span />
                </div>
              ))}
              {allocations.length === 0 && (
                <p className="srv-desc" style={{ padding: 16 }}>
                  No allocations on this node yet.
                </p>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
