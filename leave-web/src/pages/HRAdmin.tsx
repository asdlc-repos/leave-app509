import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { api, ApiError } from "../api/client";
import { Modal } from "../components/Modal";
import type {
  Balance,
  Blackout,
  LeaveType,
  User,
  UtilizationReportRow,
} from "../types";

type Tab = "types" | "blackouts" | "balances" | "reports";

export default function HRAdmin() {
  const [tab, setTab] = useState<Tab>("types");

  return (
    <div className="page">
      <h1>HR Administration</h1>
      <div className="tabs">
        <button className={tab === "types" ? "tab active" : "tab"} onClick={() => setTab("types")}>Leave types</button>
        <button className={tab === "blackouts" ? "tab active" : "tab"} onClick={() => setTab("blackouts")}>Blackouts</button>
        <button className={tab === "balances" ? "tab active" : "tab"} onClick={() => setTab("balances")}>Balances</button>
        <button className={tab === "reports" ? "tab active" : "tab"} onClick={() => setTab("reports")}>Reports</button>
      </div>

      {tab === "types" && <LeaveTypesPanel />}
      {tab === "blackouts" && <BlackoutsPanel />}
      {tab === "balances" && <BalanceAdjustPanel />}
      {tab === "reports" && <ReportsPanel />}
    </div>
  );
}

/* ----- Leave Types ------------------------------------------------------- */

function LeaveTypesPanel() {
  const [items, setItems] = useState<LeaveType[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [edit, setEdit] = useState<{ mode: "new" | "edit"; lt: Partial<LeaveType> } | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setItems((await api.listLeaveTypes()) || []);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : (e as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const save = async (e: FormEvent) => {
    e.preventDefault();
    if (!edit) return;
    const body: Partial<LeaveType> = {
      name: edit.lt.name,
      defaultDays: Number(edit.lt.defaultDays || 0),
      carryOver: !!edit.lt.carryOver,
      paid: edit.lt.paid !== false,
    };
    setBusy(true);
    try {
      if (edit.mode === "new") {
        await api.createLeaveType(body);
      } else if (edit.lt.id) {
        await api.updateLeaveType(edit.lt.id, body);
      }
      setEdit(null);
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : (e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <section className="section">
      <div className="section-header">
        <h2>Leave Types</h2>
        <button
          className="btn btn-primary"
          onClick={() => setEdit({ mode: "new", lt: { name: "", defaultDays: 0, paid: true } })}
        >
          + New type
        </button>
      </div>
      {error && <div className="error-banner">{error}</div>}
      {loading && <div className="text-muted">Loading…</div>}
      <div className="table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Default days</th>
              <th>Paid</th>
              <th>Carry over</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {items.map((lt) => (
              <tr key={lt.id}>
                <td>{lt.name}</td>
                <td>{lt.defaultDays}</td>
                <td>{lt.paid !== false ? "Yes" : "No"}</td>
                <td>{lt.carryOver ? "Yes" : "No"}</td>
                <td>
                  <button className="btn btn-small" onClick={() => setEdit({ mode: "edit", lt })}>Edit</button>
                </td>
              </tr>
            ))}
            {items.length === 0 && !loading && (
              <tr><td colSpan={5} className="empty">No leave types defined.</td></tr>
            )}
          </tbody>
        </table>
      </div>

      <Modal
        open={!!edit}
        title={edit?.mode === "new" ? "Create leave type" : "Edit leave type"}
        onClose={() => !busy && setEdit(null)}
        footer={
          <>
            <button className="btn btn-ghost" onClick={() => setEdit(null)} disabled={busy}>Cancel</button>
            <button
              className="btn btn-primary"
              form="leave-type-form"
              type="submit"
              disabled={busy}
            >
              {busy ? "Saving…" : "Save"}
            </button>
          </>
        }
      >
        {edit && (
          <form id="leave-type-form" onSubmit={save} className="form-grid">
            <label className="field">
              <span>Name</span>
              <input
                required
                value={edit.lt.name || ""}
                onChange={(e) => setEdit({ ...edit, lt: { ...edit.lt, name: e.target.value } })}
              />
            </label>
            <label className="field">
              <span>Default days / year</span>
              <input
                type="number"
                min={0}
                required
                value={edit.lt.defaultDays ?? 0}
                onChange={(e) => setEdit({ ...edit, lt: { ...edit.lt, defaultDays: Number(e.target.value) } })}
              />
            </label>
            <label className="field inline">
              <input
                type="checkbox"
                checked={edit.lt.paid !== false}
                onChange={(e) => setEdit({ ...edit, lt: { ...edit.lt, paid: e.target.checked } })}
              />
              <span>Paid leave</span>
            </label>
            <label className="field inline">
              <input
                type="checkbox"
                checked={!!edit.lt.carryOver}
                onChange={(e) => setEdit({ ...edit, lt: { ...edit.lt, carryOver: e.target.checked } })}
              />
              <span>Allow carry-over</span>
            </label>
          </form>
        )}
      </Modal>
    </section>
  );
}

/* ----- Blackouts --------------------------------------------------------- */

function BlackoutsPanel() {
  const [items, setItems] = useState<Blackout[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showNew, setShowNew] = useState(false);
  const [form, setForm] = useState<Partial<Blackout>>({});
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setItems((await api.listBlackouts()) || []);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : (e as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const save = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    try {
      await api.createBlackout({
        name: form.name,
        startDate: form.startDate,
        endDate: form.endDate,
        department: form.department || undefined,
        reason: form.reason || undefined,
      });
      setShowNew(false);
      setForm({});
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : (e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <section className="section">
      <div className="section-header">
        <h2>Blackout Windows</h2>
        <button className="btn btn-primary" onClick={() => setShowNew(true)}>+ New blackout</button>
      </div>
      {error && <div className="error-banner">{error}</div>}
      {loading && <div className="text-muted">Loading…</div>}
      <div className="table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Dates</th>
              <th>Department</th>
              <th>Reason</th>
            </tr>
          </thead>
          <tbody>
            {items.map((b) => (
              <tr key={b.id}>
                <td>{b.name}</td>
                <td>{b.startDate} → {b.endDate}</td>
                <td>{b.department || "All"}</td>
                <td>{b.reason || "—"}</td>
              </tr>
            ))}
            {items.length === 0 && !loading && (
              <tr><td colSpan={4} className="empty">No blackouts configured.</td></tr>
            )}
          </tbody>
        </table>
      </div>

      <Modal
        open={showNew}
        title="Create blackout"
        onClose={() => !busy && setShowNew(false)}
        footer={
          <>
            <button className="btn btn-ghost" onClick={() => setShowNew(false)} disabled={busy}>Cancel</button>
            <button className="btn btn-primary" form="blackout-form" type="submit" disabled={busy}>
              {busy ? "Saving…" : "Save"}
            </button>
          </>
        }
      >
        <form id="blackout-form" onSubmit={save} className="form-grid">
          <label className="field">
            <span>Name</span>
            <input required value={form.name || ""} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          </label>
          <div className="row">
            <label className="field">
              <span>Start date</span>
              <input type="date" required value={form.startDate || ""} onChange={(e) => setForm({ ...form, startDate: e.target.value })} />
            </label>
            <label className="field">
              <span>End date</span>
              <input type="date" required value={form.endDate || ""} onChange={(e) => setForm({ ...form, endDate: e.target.value })} />
            </label>
          </div>
          <label className="field">
            <span>Department (optional)</span>
            <input value={form.department || ""} onChange={(e) => setForm({ ...form, department: e.target.value })} />
          </label>
          <label className="field">
            <span>Reason (optional)</span>
            <textarea rows={2} value={form.reason || ""} onChange={(e) => setForm({ ...form, reason: e.target.value })} />
          </label>
        </form>
      </Modal>
    </section>
  );
}

/* ----- Balance Adjust ---------------------------------------------------- */

function BalanceAdjustPanel() {
  const [users, setUsers] = useState<User[]>([]);
  const [types, setTypes] = useState<LeaveType[]>([]);
  const [selectedUser, setSelectedUser] = useState<string>("");
  const [balances, setBalances] = useState<Balance[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [form, setForm] = useState<{ leaveTypeId: string; delta: string; note: string }>({
    leaveTypeId: "",
    delta: "",
    note: "",
  });
  const [busy, setBusy] = useState(false);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const [us, ts] = await Promise.all([api.listUsers(), api.listLeaveTypes()]);
        setUsers(us || []);
        setTypes(ts || []);
      } catch (e) {
        setError((e as Error).message);
      }
    })();
  }, []);

  const loadBalances = useCallback(async () => {
    if (!selectedUser) { setBalances([]); return; }
    setLoading(true);
    try {
      setBalances((await api.getBalances(selectedUser)) || []);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : (e as Error).message);
    } finally {
      setLoading(false);
    }
  }, [selectedUser]);

  useEffect(() => { loadBalances(); }, [loadBalances]);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setSuccess(null);
    setError(null);
    if (!selectedUser) return;
    const delta = Number(form.delta);
    if (!Number.isFinite(delta) || delta === 0) { setError("Delta must be a non-zero number"); return; }
    if (!form.note.trim()) { setError("Audit note is required"); return; }
    setBusy(true);
    try {
      await api.adjustBalance(selectedUser, {
        leaveTypeId: form.leaveTypeId,
        delta,
        note: form.note.trim(),
      });
      setForm({ leaveTypeId: form.leaveTypeId, delta: "", note: "" });
      setSuccess("Balance adjusted successfully.");
      await loadBalances();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : (e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <section className="section">
      <div className="section-header"><h2>Balance Adjustments</h2></div>
      {error && <div className="error-banner">{error}</div>}
      {success && <div className="success-banner">{success}</div>}

      <div className="filter-bar">
        <label className="field">
          <span>Employee</span>
          <select value={selectedUser} onChange={(e) => setSelectedUser(e.target.value)}>
            <option value="">Select an employee…</option>
            {users.map((u) => <option key={u.id} value={u.id}>{u.name || u.email}</option>)}
          </select>
        </label>
      </div>

      {selectedUser && (
        <>
          {loading ? <div className="text-muted">Loading balances…</div> : (
            <div className="balance-grid">
              {balances.map((b) => (
                <div key={b.leaveTypeId} className="balance-card">
                  <div className="balance-name">{b.leaveTypeName}</div>
                  <div className="balance-number">{b.available}</div>
                  <div className="balance-sub">available · {b.used} used · {b.total} total</div>
                </div>
              ))}
              {balances.length === 0 && <div className="empty">No balances yet.</div>}
            </div>
          )}

          <form className="form-grid adjust-form" onSubmit={submit}>
            <h3>Adjust balance</h3>
            <label className="field">
              <span>Leave type</span>
              <select required value={form.leaveTypeId} onChange={(e) => setForm({ ...form, leaveTypeId: e.target.value })}>
                <option value="">Select…</option>
                {types.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
              </select>
            </label>
            <label className="field">
              <span>Delta (days, +/-)</span>
              <input
                type="number"
                required
                step="0.5"
                value={form.delta}
                onChange={(e) => setForm({ ...form, delta: e.target.value })}
                placeholder="e.g. 2 or -1"
              />
            </label>
            <label className="field">
              <span>Audit note (required)</span>
              <textarea
                required
                rows={2}
                value={form.note}
                onChange={(e) => setForm({ ...form, note: e.target.value })}
                placeholder="Reason for adjustment"
              />
            </label>
            <button className="btn btn-primary" type="submit" disabled={busy}>
              {busy ? "Applying…" : "Apply adjustment"}
            </button>
          </form>
        </>
      )}
    </section>
  );
}

/* ----- Reports ----------------------------------------------------------- */

function ReportsPanel() {
  const [rows, setRows] = useState<UtilizationReportRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [downloading, setDownloading] = useState<"csv" | "pdf" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [from, setFrom] = useState<string>(() => {
    const d = new Date();
    d.setMonth(0, 1);
    return d.toISOString().slice(0, 10);
  });
  const [to, setTo] = useState<string>(() => new Date().toISOString().slice(0, 10));
  const [department, setDepartment] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.utilizationReport({
        from: from || undefined,
        to: to || undefined,
        department: department || undefined,
      });
      setRows(res?.rows || []);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : (e as Error).message);
    } finally {
      setLoading(false);
    }
  }, [from, to, department]);

  useEffect(() => {
    load();
  }, [load]);

  const download = async (format: "csv" | "pdf") => {
    setDownloading(format);
    setError(null);
    try {
      const blob = await api.downloadReport(format, {
        from: from || undefined,
        to: to || undefined,
        department: department || undefined,
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `utilization-${from || "start"}-${to || "end"}.${format}`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : (e as Error).message);
    } finally {
      setDownloading(null);
    }
  };

  const totalUsed = useMemo(() => rows.reduce((acc, r) => acc + (r.used || 0), 0), [rows]);

  return (
    <section className="section">
      <div className="section-header"><h2>Utilization Report</h2></div>
      <div className="filter-bar">
        <label className="field">
          <span>From</span>
          <input type="date" value={from} onChange={(e) => setFrom(e.target.value)} />
        </label>
        <label className="field">
          <span>To</span>
          <input type="date" value={to} onChange={(e) => setTo(e.target.value)} />
        </label>
        <label className="field">
          <span>Department</span>
          <input value={department} onChange={(e) => setDepartment(e.target.value)} placeholder="Any" />
        </label>
        <button className="btn btn-ghost" onClick={load} disabled={loading}>Run</button>
        <button className="btn btn-primary" onClick={() => download("csv")} disabled={downloading !== null}>
          {downloading === "csv" ? "Exporting…" : "Export CSV"}
        </button>
        <button className="btn btn-primary" onClick={() => download("pdf")} disabled={downloading !== null}>
          {downloading === "pdf" ? "Exporting…" : "Export PDF"}
        </button>
      </div>

      {error && <div className="error-banner">{error}</div>}
      {loading && <div className="text-muted">Loading…</div>}

      <div className="table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>Employee</th>
              <th>Department</th>
              <th>Team</th>
              <th>Leave type</th>
              <th>Used</th>
              <th>Balance</th>
              <th>Utilization %</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r, i) => (
              <tr key={`${r.userId}-${r.leaveTypeName}-${i}`}>
                <td>{r.userName}</td>
                <td>{r.department || "—"}</td>
                <td>{r.team || "—"}</td>
                <td>{r.leaveTypeName}</td>
                <td>{r.used}</td>
                <td>{r.balance}</td>
                <td>{r.utilizationPct}%</td>
              </tr>
            ))}
            {rows.length === 0 && !loading && (
              <tr><td colSpan={7} className="empty">No data for the selected filters.</td></tr>
            )}
          </tbody>
          {rows.length > 0 && (
            <tfoot>
              <tr>
                <td colSpan={4}><strong>Total used</strong></td>
                <td colSpan={3}><strong>{totalUsed}</strong></td>
              </tr>
            </tfoot>
          )}
        </table>
      </div>
    </section>
  );
}
