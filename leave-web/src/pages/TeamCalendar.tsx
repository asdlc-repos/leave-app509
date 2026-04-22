import { useCallback, useEffect, useState } from "react";
import { api, ApiError } from "../api/client";
import type { CalendarResponse } from "../types";
import { addDays, isoDate } from "../utils/date";
import { CalendarGrid } from "../components/CalendarGrid";

export default function TeamCalendar() {
  const [data, setData] = useState<CalendarResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const today = isoDate(new Date());
  const end = isoDate(addDays(new Date(), 89));

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.teamCalendar({ from: today, to: end });
      setData(res);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : (e as Error).message);
    } finally {
      setLoading(false);
    }
  }, [today, end]);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1>Team Calendar</h1>
          <p className="text-muted">Upcoming 90 days for your direct reports.</p>
        </div>
        <button className="btn btn-ghost" onClick={load}>Refresh</button>
      </div>
      {error && <div className="error-banner">{error}</div>}
      {loading && <div className="text-muted">Loading…</div>}
      {data && (
        <CalendarGrid
          from={today}
          days={90}
          events={data.events || []}
          capacities={data.capacities}
          holidays={data.holidays}
        />
      )}
    </div>
  );
}
