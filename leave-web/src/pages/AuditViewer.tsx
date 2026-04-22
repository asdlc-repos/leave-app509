import { useCallback, useEffect, useState } from "react";
import { api, ApiError } from "../api/client";
import type { AuditEntry, User } from "../types";

export default function AuditViewer() {
  const [users, setUsers] = useState<User[]>([]);
  const [selected, setSelected] = useState<string>("");
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const us = await api.listUsers();
        setUsers(us || []);
      } catch (e) {
        setError((e as Error).message);
      }
    })();
  }, []);

  const load = useCallback(async () => {
    if (!selected) {
      setEntries([]);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const data = await api.audit(selected);
      setEntries(data || []);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : (e as Error).message);
    } finally {
      setLoading(false);
    }
  }, [selected]);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="page">
      <h1>Audit Trail</h1>
      <div className="filter-bar">
        <label className="field">
          <span>Employee</span>
          <select value={selected} onChange={(e) => setSelected(e.target.value)}>
            <option value="">Select an employee…</option>
            {users.map((u) => (
              <option key={u.id} value={u.id}>
                {u.name || u.email}
              </option>
            ))}
          </select>
        </label>
      </div>
      {error && <div className="error-banner">{error}</div>}
      {loading && <div className="text-muted">Loading…</div>}
      {!loading && selected && entries.length === 0 && (
        <div className="empty">No audit entries for this employee yet.</div>
      )}
      {entries.length > 0 && (
        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                <th>When</th>
                <th>Actor</th>
                <th>Action</th>
                <th>Target</th>
                <th>Details</th>
                <th>Note</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((e) => (
                <tr key={e.id}>
                  <td>{new Date(e.timestamp).toLocaleString()}</td>
                  <td>{e.actorName || e.actorId}</td>
                  <td>{e.action}</td>
                  <td>{e.target}</td>
                  <td className="truncate" title={e.details || ""}>{e.details || "—"}</td>
                  <td className="truncate" title={e.note || ""}>{e.note || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
