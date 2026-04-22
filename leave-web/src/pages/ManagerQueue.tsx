import { useCallback, useEffect, useState } from "react";
import { api, ApiError } from "../api/client";
import type { LeaveRequest } from "../types";
import { formatRange } from "../utils/date";
import { Modal } from "../components/Modal";

type Action = "approve" | "reject";

export default function ManagerQueue() {
  const [items, setItems] = useState<LeaveRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [target, setTarget] = useState<{ req: LeaveRequest; action: Action } | null>(null);
  const [comment, setComment] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.managerQueue();
      setItems(data || []);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : (e as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const decide = async () => {
    if (!target) return;
    if (target.action === "reject" && !comment.trim()) {
      setError("A comment is required when rejecting a request.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      if (target.action === "approve") {
        await api.approveLeaveRequest(target.req.id, comment.trim() || undefined);
      } else {
        await api.rejectLeaveRequest(target.req.id, comment.trim());
      }
      setTarget(null);
      setComment("");
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : (e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="page">
      <div className="page-header">
        <h1>Pending Approvals</h1>
        <button className="btn btn-ghost" onClick={load}>Refresh</button>
      </div>

      {error && <div className="error-banner">{error}</div>}
      {loading && <div className="text-muted">Loading…</div>}
      {!loading && items.length === 0 && <div className="empty">Nothing awaiting your decision.</div>}

      {items.length > 0 && (
        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                <th>Employee</th>
                <th>Type</th>
                <th>Dates</th>
                <th>Days</th>
                <th>Reason</th>
                <th>Submitted</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {items.map((r) => (
                <tr key={r.id}>
                  <td>{r.userName || r.userId}</td>
                  <td>{r.leaveTypeName || r.leaveTypeId}</td>
                  <td>{formatRange(r.startDate, r.endDate)}</td>
                  <td>{r.days}</td>
                  <td className="truncate" title={r.reason || ""}>{r.reason || "—"}</td>
                  <td>{new Date(r.submittedAt).toLocaleDateString()}</td>
                  <td className="actions">
                    <button
                      className="btn btn-small btn-success"
                      onClick={() => {
                        setTarget({ req: r, action: "approve" });
                        setComment("");
                      }}
                    >
                      Approve
                    </button>
                    <button
                      className="btn btn-small btn-danger"
                      onClick={() => {
                        setTarget({ req: r, action: "reject" });
                        setComment("");
                      }}
                    >
                      Reject
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Modal
        open={!!target}
        title={target?.action === "approve" ? "Approve request" : "Reject request"}
        onClose={() => {
          if (!busy) {
            setTarget(null);
            setComment("");
          }
        }}
        footer={
          <>
            <button
              className="btn btn-ghost"
              onClick={() => {
                setTarget(null);
                setComment("");
              }}
              disabled={busy}
            >
              Cancel
            </button>
            <button
              className={`btn ${target?.action === "approve" ? "btn-success" : "btn-danger"}`}
              onClick={decide}
              disabled={busy}
            >
              {busy
                ? "…"
                : target?.action === "approve"
                ? "Approve"
                : "Reject"}
            </button>
          </>
        }
      >
        {target && (
          <div className="form-grid">
            <div>
              <strong>{target.req.userName || target.req.userId}</strong> · {target.req.leaveTypeName || target.req.leaveTypeId}
            </div>
            <div className="text-muted">
              {formatRange(target.req.startDate, target.req.endDate)} · {target.req.days} day
              {target.req.days === 1 ? "" : "s"}
            </div>
            {target.req.reason && <div className="muted-box">Reason: {target.req.reason}</div>}
            <label className="field">
              <span>
                Comment{target.action === "reject" ? " (required)" : " (optional)"}
              </span>
              <textarea
                rows={3}
                value={comment}
                onChange={(e) => setComment(e.target.value)}
                placeholder={
                  target.action === "approve"
                    ? "Optional note"
                    : "Explain why this is being rejected"
                }
              />
            </label>
          </div>
        )}
      </Modal>
    </div>
  );
}
