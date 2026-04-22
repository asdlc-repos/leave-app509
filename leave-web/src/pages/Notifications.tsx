import { useEffect, useState } from "react";
import { api } from "../api/client";
import type { InAppNotification } from "../types";

export default function Notifications() {
  const [items, setItems] = useState<InAppNotification[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      try {
        const data = await api.inAppNotifications();
        if (!cancelled) setItems(data || []);
      } catch (e) {
        if (!cancelled) setError((e as Error).message);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className="page">
      <h1>Notifications</h1>
      {loading && <div className="text-muted">Loading…</div>}
      {error && <div className="error-banner">{error}</div>}
      {!loading && items.length === 0 && <div className="empty">Nothing to show.</div>}
      <ul className="notif-full-list">
        {items.map((n) => (
          <li key={n.id} className={n.read ? "read" : "unread"}>
            <div className="notif-title">{n.title}</div>
            {n.body && <div className="notif-body">{n.body}</div>}
            <div className="notif-meta">{new Date(n.createdAt).toLocaleString()}</div>
          </li>
        ))}
      </ul>
    </div>
  );
}
