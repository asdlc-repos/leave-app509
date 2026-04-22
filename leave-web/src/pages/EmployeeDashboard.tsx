import { useCallback, useEffect, useState } from "react";
import { api, ApiError } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import type { Balance, LeaveRequest, LeaveType } from "../types";
import { formatRange } from "../utils/date";
import { CreateRequestModal } from "./CreateRequestModal";

function statusClass(s: LeaveRequest["status"]): string {
  return `status-pill status-${s}`;
}

export default function EmployeeDashboard() {
  const { user } = useAuth();
  const [balances, setBalances] = useState<Balance[]>([]);
  const [requests, setRequests] = useState<LeaveRequest[]>([]);
  const [leaveTypes, setLeaveTypes] = useState<LeaveType[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [cancelling, setCancelling] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!user) return;
    setLoading(true);
    setError(null);
    try {
      const [bals, myReqs, types] = await Promise.all([
        api.getBalances(user.id),
        api.listLeaveRequests({ userId: user.id }),
        api.listLeaveTypes(),
      ]);
      setBalances(bals || []);
      setRequests(myReqs || []);
      setLeaveTypes(types || []);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : (e as Error).message);
    } finally {
      setLoading(false);
    }
  }, [user]);

  useEffect(() => {
    load();
  }, [load]);

  const onCancel = async (id: string) => {
    if (!confirm("Cancel this request?")) return;
    setCancelling(id);
    try {
      await api.cancelLeaveRequest(id);
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : (e as Error).message);
    } finally {
      setCancelling(null);
    }
  };

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1>Welcome{user?.name ? `, ${user.name}` : ""}</h1>
          <p className="text-muted">Your leave overview at a glance.</p>
        </div>
        <button className="btn btn-primary" onClick={() => setCreateOpen(true)}>
          + Request Leave
        </button>
      </div>

      {error && <div className="error-banner">{error}</div>}
      {loading && <div className="text-muted">Loading…</div>}

      <section className="section">
        <h2>My Balances</h2>
        {balances.length === 0 && !loading ? (
          <div className="empty">No balances configured yet.</div>
        ) : (
          <div className="balance-grid">
            {balances.map((b) => (
              <div key={b.leaveTypeId} className="balance-card">
                <div className="balance-name">{b.leaveTypeName}</div>
                <div className="balance-number">{b.available}</div>
                <div className="balance-sub">
                  available · {b.used} used · {b.reserved} reserved · {b.total} total
                </div>
                <div className="balance-meter">
                  <div
                    className="balance-meter-fill"
                    style={{
                      width: `${Math.max(
                        0,
                        Math.min(100, ((b.used + b.reserved) / Math.max(1, b.total)) * 100)
                      )}%`,
                    }}
                  />
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      <section className="section">
        <h2>My Requests</h2>
        {requests.length === 0 && !loading ? (
          <div className="empty">You haven't submitted any requests yet.</div>
        ) : (
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Type</th>
                  <th>Dates</th>
                  <th>Days</th>
                  <th>Status</th>
                  <th>Submitted</th>
                  <th>Attachments</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {requests.map((r) => (
                  <tr key={r.id}>
                    <td>{r.leaveTypeName || r.leaveTypeId}</td>
                    <td>{formatRange(r.startDate, r.endDate)}</td>
                    <td>{r.days}</td>
                    <td>
                      <span className={statusClass(r.status)}>{r.status}</span>
                      {r.comment && <div className="row-comment">“{r.comment}”</div>}
                    </td>
                    <td>{new Date(r.submittedAt).toLocaleDateString()}</td>
                    <td>{r.attachments?.length ? `${r.attachments.length}` : "—"}</td>
                    <td>
                      {r.status === "pending" ? (
                        <button
                          className="btn btn-small"
                          onClick={() => onCancel(r.id)}
                          disabled={cancelling === r.id}
                        >
                          {cancelling === r.id ? "…" : "Cancel"}
                        </button>
                      ) : (
                        <span className="text-muted">—</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <CreateRequestModal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onCreated={load}
        leaveTypes={leaveTypes}
        balances={balances}
      />
    </div>
  );
}
