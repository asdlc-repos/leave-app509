import { FormEvent, useMemo, useState } from "react";
import { Modal } from "../components/Modal";
import { api, ApiError } from "../api/client";
import type { Balance, LeaveType } from "../types";
import { daysBetweenInclusive } from "../utils/date";

const MAX_ATTACHMENT_SIZE = 5 * 1024 * 1024; // 5 MB
const ALLOWED_MIME = ["application/pdf", "image/jpeg", "image/png"];

export function CreateRequestModal({
  open,
  onClose,
  onCreated,
  leaveTypes,
  balances,
}: {
  open: boolean;
  onClose: () => void;
  onCreated: () => void;
  leaveTypes: LeaveType[];
  balances: Balance[];
}) {
  const [leaveTypeId, setLeaveTypeId] = useState("");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [reason, setReason] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const days = useMemo(() => {
    if (!startDate || !endDate) return 0;
    if (endDate < startDate) return 0;
    return daysBetweenInclusive(startDate, endDate);
  }, [startDate, endDate]);

  const balanceForSelected = useMemo(
    () => balances.find((b) => b.leaveTypeId === leaveTypeId),
    [balances, leaveTypeId]
  );

  const reset = () => {
    setLeaveTypeId("");
    setStartDate("");
    setEndDate("");
    setReason("");
    setFile(null);
    setFileError(null);
    setError(null);
  };

  const onFileChange = (f: File | null) => {
    setFileError(null);
    if (!f) {
      setFile(null);
      return;
    }
    if (f.size > MAX_ATTACHMENT_SIZE) {
      setFileError("File exceeds 5 MB limit");
      return;
    }
    if (!ALLOWED_MIME.includes(f.type)) {
      setFileError("Only PDF, JPG, PNG files are allowed");
      return;
    }
    setFile(f);
  };

  const readAsBase64 = (f: File): Promise<string> =>
    new Promise((resolve, reject) => {
      const r = new FileReader();
      r.onload = () => {
        const s = String(r.result || "");
        const idx = s.indexOf(",");
        resolve(idx >= 0 ? s.slice(idx + 1) : s);
      };
      r.onerror = () => reject(r.error);
      r.readAsDataURL(f);
    });

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    if (!leaveTypeId || !startDate || !endDate) {
      setError("Please complete all required fields");
      return;
    }
    if (endDate < startDate) {
      setError("End date must be on or after start date");
      return;
    }
    setSubmitting(true);
    try {
      const created = await api.createLeaveRequest({
        leaveTypeId,
        startDate,
        endDate,
        reason: reason.trim() || undefined,
      });
      if (file) {
        const data = await readAsBase64(file);
        await api.uploadAttachment(created.id, {
          filename: file.name,
          mimeType: file.type,
          data,
        });
      }
      reset();
      onCreated();
      onClose();
    } catch (e) {
      const msg = e instanceof ApiError ? e.message : (e as Error).message;
      setError(msg || "Could not submit request");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      open={open}
      title="Request Leave"
      onClose={() => {
        if (!submitting) {
          reset();
          onClose();
        }
      }}
      footer={
        <>
          <button
            className="btn btn-ghost"
            type="button"
            onClick={() => {
              reset();
              onClose();
            }}
            disabled={submitting}
          >
            Cancel
          </button>
          <button
            className="btn btn-primary"
            type="submit"
            form="create-request-form"
            disabled={submitting}
          >
            {submitting ? "Submitting…" : "Submit request"}
          </button>
        </>
      }
    >
      <form id="create-request-form" onSubmit={onSubmit} className="form-grid">
        <label className="field">
          <span>Leave type</span>
          <select
            required
            value={leaveTypeId}
            onChange={(e) => setLeaveTypeId(e.target.value)}
          >
            <option value="">Select…</option>
            {leaveTypes.map((lt) => (
              <option key={lt.id} value={lt.id}>
                {lt.name}
              </option>
            ))}
          </select>
          {balanceForSelected && (
            <small className="text-muted">
              Available: {balanceForSelected.available} day(s)
            </small>
          )}
        </label>

        <div className="row">
          <label className="field">
            <span>Start date</span>
            <input
              type="date"
              required
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
            />
          </label>
          <label className="field">
            <span>End date</span>
            <input
              type="date"
              required
              value={endDate}
              min={startDate || undefined}
              onChange={(e) => setEndDate(e.target.value)}
            />
          </label>
        </div>

        {days > 0 && (
          <div className="text-muted">Total: {days} day{days === 1 ? "" : "s"}</div>
        )}

        <label className="field">
          <span>Reason (optional)</span>
          <textarea
            rows={3}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="Brief reason for this request"
          />
        </label>

        <label className="field">
          <span>Attachment (PDF/JPG/PNG, ≤ 5 MB)</span>
          <input
            type="file"
            accept="application/pdf,image/jpeg,image/png"
            onChange={(e) => onFileChange(e.target.files?.[0] || null)}
          />
          {file && (
            <small className="text-muted">
              {file.name} ({Math.round(file.size / 1024)} KB)
            </small>
          )}
          {fileError && <small className="error">{fileError}</small>}
        </label>

        {error && <div className="error-banner">{error}</div>}
      </form>
    </Modal>
  );
}
