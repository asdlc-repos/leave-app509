import { useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import type { InAppNotification } from "../types";

const POLL_INTERVAL_MS = 30000;

export function NotificationBell() {
  const { token } = useAuth();
  const [items, setItems] = useState<InAppNotification[]>([]);
  const [open, setOpen] = useState(false);
  const wrapRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!token) return;
    let cancelled = false;
    const load = async () => {
      try {
        const data = await api.inAppNotifications();
        if (!cancelled) setItems(data || []);
      } catch {
        // ignore polling errors
      }
    };
    load();
    const id = window.setInterval(load, POLL_INTERVAL_MS);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [token]);

  useEffect(() => {
    if (!open) return;
    const onClickAway = (e: MouseEvent) => {
      if (!wrapRef.current) return;
      if (!wrapRef.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onClickAway);
    return () => document.removeEventListener("mousedown", onClickAway);
  }, [open]);

  const unread = useMemo(() => items.filter((n) => !n.read).length, [items]);

  return (
    <div className="notif-wrap" ref={wrapRef}>
      <button
        className="notif-btn"
        aria-label="Notifications"
        onClick={() => setOpen((v) => !v)}
      >
        <span aria-hidden="true">🔔</span>
        {unread > 0 && <span className="notif-badge">{unread > 99 ? "99+" : unread}</span>}
      </button>
      {open && (
        <div className="notif-popover" role="dialog" aria-label="Notifications">
          <div className="notif-header">
            <strong>Notifications</strong>
            <Link to="/notifications" onClick={() => setOpen(false)}>View all</Link>
          </div>
          {items.length === 0 ? (
            <div className="notif-empty">No notifications</div>
          ) : (
            <ul className="notif-list">
              {items.slice(0, 8).map((n) => (
                <li key={n.id} className={n.read ? "read" : "unread"}>
                  <div className="notif-title">{n.title}</div>
                  {n.body && <div className="notif-body">{n.body}</div>}
                  <div className="notif-meta">{new Date(n.createdAt).toLocaleString()}</div>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
